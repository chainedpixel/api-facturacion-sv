package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	authModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDte "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/user"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// fakeHaciendaAuth returns a fixed Hacienda token and counts the calls.
type fakeHaciendaAuth struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeHaciendaAuth) GetOrCreateHaciendaToken(context.Context, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return "hacienda-token", nil
}

func (f *fakeHaciendaAuth) GetOrCreateHaciendaTokenWithCreds(context.Context, string, authModels.HaciendaCredentials) (string, error) {
	return "hacienda-token", nil
}

type verifiedBatchCall struct {
	ids     []string
	results map[string]coreDte.ReceptionResult
	status  string
}

type existingEventRecorder struct {
	mu          sync.Mutex
	updatedDocs map[string]coreDte.DTEDetails
	verified    []verifiedBatchCall
}

// TestContingencyExistingEventSettlesDocumentsAlreadyKnownByHacienda checks that when the contingency event already
// exists the documents are verified against Hacienda, final statuses are stored, and only the documents without a
// final status are transmitted.
func TestContingencyExistingEventSettlesDocumentsAlreadyKnownByHacienda(t *testing.T) {
	test.TestMain(t)

	var consulted []string
	var consultMu sync.Mutex
	consult := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		consultMu.Lock()
		consulted = append(consulted, body["codigoGeneracion"])
		consultMu.Unlock()
		assert.Equal(t, "hacienda-token", r.Header.Get("Authorization"))
		assert.Equal(t, "0614-010101-000-0", body["nitEmisor"])
		assert.Equal(t, constants.FacturaElectronica, body["tdte"])

		w.Header().Set("Content-Type", "application/json")
		switch body["codigoGeneracion"] {
		case "DOC-A":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"estado": "PROCESADO", "selloRecibido": "STAMP-A", "fhProcesamiento": "14/11/2026 10:00:00",
			})
		case "DOC-B":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"estado": "RECHAZADO", "fhProcesamiento": "14/11/2026 10:01:00", "codigoMsg": "801", "descripcionMsg": "rejected",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer consult.Close()

	previousURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = consult.URL
	t.Cleanup(func() { config.MHPaths.ReceptionConsultURL = previousURL })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeSvc, _, _ := storeDeferredFixture(t, ctrl, &fakeContingencySigner{})
	deferred, err := storeSvc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
		constants.FacturaElectronica, constants.NoDisponibilidadMH, "Servicio no disponible")
	require.NoError(t, err)
	group := []coreDte.ContingencyDocument{
		pendingFromDeferred("DOC-A", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred),
		pendingFromDeferred("DOC-B", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred),
		pendingFromDeferred("DOC-C", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred),
	}

	recorder := &existingEventRecorder{updatedDocs: map[string]coreDte.DTEDetails{}}
	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().GetPending(gomock.Any(), gomock.Any()).Return(group, nil)
	mockRepo.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ids, _ []string, results map[string]coreDte.ReceptionResult, _, _, status string) error {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.verified = append(recorder.verified, verifiedBatchCall{ids: ids, results: results, status: status})
			return nil
		}).AnyTimes()

	mockEvent := mocks.NewMockContingencyEventSender(ctrl)
	mockEvent.EXPECT().PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
		Return(&contingency.ContingencyEventExistsError{Message: "exists", Documents: group})

	owner := &user.User{ID: 7, NIT: "06140101010000"}
	mockAuth := mocks.NewMockAuthManager(ctrl)
	mockAuth.EXPECT().GetByNIT(gomock.Any(), "0614-010101-000-0").
		Return(&user.User{ID: 7, NIT: "06140101010000", BranchOffices: []user.BranchOffice{{ID: 1, User: owner}}}, nil).AnyTimes()
	mockAuth.EXPECT().GetBranchByBranchID(gomock.Any(), gomock.Any()).
		Return(&user.BranchOffice{ID: 1, User: owner}, nil).AnyTimes()

	mockCache := mocks.NewMockCacheManager(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).Return(`{"IssuedAt":1,"ExpiresAt":4102444800}`, nil).AnyTimes()
	mockCache.EXPECT().GetCredentials(gomock.Any()).Return(&authModels.HaciendaCredentials{Username: "u", Password: "p"}, nil).AnyTimes()

	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().UpdateDTE(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uint, details coreDte.DTEDetails) error {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.updatedDocs[details.ID] = details
			return nil
		}).AnyTimes()

	auth := &fakeHaciendaAuth{}
	batch := &fakeBatchTransmitter{}
	svc := contingency.NewContingencyManager(mockAuth, mockDTE, mockRepo, auth, mockCache, fakeContingencyTokens{},
		&fakeContingencySigner{}, batch, mockEvent, mocks.NewMockSequentialNumberManager(ctrl), nil,
		&transmitterModels.TransmissionConfig{BatchSize: 100})

	require.NoError(t, svc.RetransmitPendingDocuments(context.Background()))

	assert.ElementsMatch(t, []string{"DOC-A", "DOC-B", "DOC-C"}, consulted, "every document of the event is consulted")
	assert.Equal(t, 3, auth.calls)

	sent := batch.batches[constants.FacturaElectronica]
	require.Len(t, sent, 1, "only the document without a final status is transmitted")
	assert.Equal(t, group[2].Document.SignedDocument, sent[0], "the stored JWS of the unresolved document is transmitted")

	require.Len(t, recorder.updatedDocs, 2, "documents without a final status are not updated")
	received := recorder.updatedDocs["DOC-A"]
	assert.Equal(t, constants.DocumentReceived, received.Status)
	require.NotNil(t, received.ReceptionStamp)
	assert.Equal(t, "STAMP-A", *received.ReceptionStamp)
	assert.Equal(t, constants.TransmissionContingency, received.Transmission)
	rejected := recorder.updatedDocs["DOC-B"]
	assert.Equal(t, constants.DocumentRejected, rejected.Status)
	assert.Nil(t, rejected.ReceptionStamp)
	assert.NotContains(t, recorder.updatedDocs, "DOC-C")

	verifiedByStatus := map[string]verifiedBatchCall{}
	for _, call := range recorder.verified {
		if call.status == constants.DocumentReceived || call.status == constants.DocumentRejected {
			verifiedByStatus[call.status] = call
		}
	}
	require.Contains(t, verifiedByStatus, constants.DocumentReceived)
	require.Contains(t, verifiedByStatus, constants.DocumentRejected)
	assert.Equal(t, []string{"contingency-DOC-A"}, verifiedByStatus[constants.DocumentReceived].ids)
	assert.Equal(t, "STAMP-A", verifiedByStatus[constants.DocumentReceived].results["contingency-DOC-A"].ReceptionStamp)
	assert.Equal(t, "14/11/2026 10:00:00", verifiedByStatus[constants.DocumentReceived].results["contingency-DOC-A"].ProcessingDate)
	assert.Equal(t, []string{"contingency-DOC-B"}, verifiedByStatus[constants.DocumentRejected].ids)
}

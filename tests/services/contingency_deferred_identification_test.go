package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDte "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/user"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// fakeContingencySigner builds a compact JWS whose payload is the signed document, and records every call.
type fakeContingencySigner struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (s *fakeContingencySigner) SignDTE(_ context.Context, document json.RawMessage, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString(document) + ".c2lnbmF0dXJl", nil
}

func (s *fakeContingencySigner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// fakeBatchTransmitter captures the documents of each batch and stops the flow after capturing them.
type fakeBatchTransmitter struct {
	mu      sync.Mutex
	batches map[string][]string
}

func (b *fakeBatchTransmitter) TransmitBatch(_ context.Context, _ string, dteType string, documents []string, _ string, _ authModels.HaciendaCredentials) (*transmitterModels.BatchResponse, string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.batches == nil {
		b.batches = make(map[string][]string)
	}
	b.batches[dteType] = append(b.batches[dteType], documents...)
	return nil, "", errors.New("batch captured")
}

func (b *fakeBatchTransmitter) VerifyContingencyBatchStatus(context.Context, string, string, string, uint, map[string]coreDte.ContingencyDocument) error {
	return nil
}

func (b *fakeBatchTransmitter) GetDTEVersion(string) int { return 1 }

// fakeContingencyTokens provides the secret key used to rebuild the branch token.
type fakeContingencyTokens struct{}

func (fakeContingencyTokens) GenerateToken(*authModels.AuthClaims, time.Duration) (string, error) {
	return "", nil
}
func (fakeContingencyTokens) ValidateToken(string) (*authModels.AuthClaims, error) { return nil, nil }
func (fakeContingencyTokens) RevokeToken(string) error                             { return nil }
func (fakeContingencyTokens) SaveTimestampsForContingency(time.Time, time.Time, time.Duration, *authModels.AuthClaims) error {
	return nil
}
func (fakeContingencyTokens) GetSecretKey() string { return "contingency-test-secret" }

type deferredIdentification struct {
	Identificacion struct {
		TipoModelo       int     `json:"tipoModelo"`
		TipoOperacion    int     `json:"tipoOperacion"`
		TipoContingencia *int    `json:"tipoContingencia"`
		MotivoContin     *string `json:"motivoContin"`
		CodigoGeneracion string  `json:"codigoGeneracion"`
	} `json:"identificacion"`
}

func decodeDeferredIdentification(t *testing.T, document interface{}) deferredIdentification {
	t.Helper()
	raw, err := json.Marshal(document)
	require.NoError(t, err)
	var decoded deferredIdentification
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded
}

func jwsPayload(t *testing.T, jws string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(jws, ".")
	require.Len(t, parts, 3, "the signed document must be a compact JWS")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func normalizedJSON(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &normalized))
	return normalized
}

func storeDeferredFixture(t *testing.T, ctrl *gomock.Controller, signer *fakeContingencySigner) (contingency.ContingencyManager, *coreDte.TransmissionEvidence, *interface{}) {
	t.Helper()
	var evidence coreDte.TransmissionEvidence
	var stored interface{}
	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().
		Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, gomock.Any()).
		DoAndReturn(func(_ interface{}, document interface{}, _ string, _ string, given *coreDte.TransmissionEvidence) error {
			stored = document
			if given != nil {
				evidence = *given
			}
			return nil
		}).AnyTimes()
	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeq.EXPECT().MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	svc := contingency.NewContingencyManager(nil, mockDTE, mockRepo, nil, nil, nil, signer, nil,
		mocks.NewMockContingencyEventSender(ctrl), mockSeq, nil, nil)
	return svc, &evidence, &stored
}

// TestUpdateContingencyIdentificationReasonRule checks the deferred identification and the default contingency reason:
// null for contingency types 1 to 4 and a text, never empty and at most 500 characters, for type 5.
func TestUpdateContingencyIdentificationReasonRule(t *testing.T) {
	test.TestMain(t)

	reason := "Motivo detallado de la contingencia"
	for _, contingencyType := range []int8{1, 2, 3, 4, 5} {
		updated, err := utils.UpdateContingencyIdentification(json.RawMessage(minimalDTEJSON()), &contingencyType, &reason)
		require.NoError(t, err)

		decoded := decodeDeferredIdentification(t, updated)
		assert.Equal(t, constants.ModeloFacturacionDiferido, decoded.Identificacion.TipoModelo)
		assert.Equal(t, constants.TransmisionContingencia, decoded.Identificacion.TipoOperacion)
		require.NotNil(t, decoded.Identificacion.TipoContingencia)
		assert.Equal(t, int(contingencyType), *decoded.Identificacion.TipoContingencia)
		if contingencyType == constants.OtroMotivo {
			require.NotNil(t, decoded.Identificacion.MotivoContin)
			assert.Equal(t, reason, *decoded.Identificacion.MotivoContin)
		} else {
			assert.Nil(t, decoded.Identificacion.MotivoContin, "contingency type %d defaults to a null reason", contingencyType)
		}
	}

	otherType := int8(constants.OtroMotivo)
	empty := ""
	updated, err := utils.UpdateContingencyIdentification(json.RawMessage(minimalDTEJSON()), &otherType, &empty)
	require.NoError(t, err)
	decoded := decodeDeferredIdentification(t, updated)
	require.NotNil(t, decoded.Identificacion.MotivoContin, "type 5 always carries a reason")
	assert.NotEmpty(t, *decoded.Identificacion.MotivoContin)

	long := strings.Repeat("x", 650)
	updated, err = utils.UpdateContingencyIdentification(json.RawMessage(minimalDTEJSON()), &otherType, &long)
	require.NoError(t, err)
	decoded = decodeDeferredIdentification(t, updated)
	require.NotNil(t, decoded.Identificacion.MotivoContin)
	assert.Len(t, []rune(*decoded.Identificacion.MotivoContin), 500)
}

// TestStoreDocumentInContingencySignsOnce checks that the deferred document is signed once when it is stored, that
// the stored structure and its JWS carry the deferred identification, and that the consumer receives the same
// structure with the same signature.
func TestStoreDocumentInContingencySignsOnce(t *testing.T) {
	test.TestMain(t)

	for _, tc := range []struct {
		name            string
		contingencyType int8
		wantReason      bool
	}{
		{"service unavailable defaults to a null reason", constants.NoDisponibilidadMH, false},
		{"other reason carries the reason", constants.OtroMotivo, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			signer := &fakeContingencySigner{}
			svc, evidence, stored := storeDeferredFixture(t, ctrl, signer)

			deferred, err := svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
				constants.FacturaElectronica, tc.contingencyType, "Motivo detallado de la contingencia")
			require.NoError(t, err)
			require.NotNil(t, deferred)
			assert.Equal(t, 1, signer.callCount(), "the deferred document is signed exactly once")

			assert.Equal(t, deferred.SignedDocument, evidence.SignedDocument, "the stored JWS is the delivered one")
			assert.Equal(t, normalizedJSON(t, deferred.Document), normalizedJSON(t, *stored), "the stored structure is the delivered one")
			assert.Equal(t, normalizedJSON(t, deferred.Document), jwsPayload(t, deferred.SignedDocument), "the JWS signs the delivered structure")

			response := deferred.Response()
			assert.Equal(t, deferred.SignedDocument, response["firmaElectronica"])
			delete(response, "firmaElectronica")
			assert.Equal(t, jwsPayload(t, deferred.SignedDocument), normalizedJSON(t, response))

			decoded := decodeDeferredIdentification(t, deferred.Document)
			assert.Equal(t, constants.ModeloFacturacionDiferido, decoded.Identificacion.TipoModelo)
			assert.Equal(t, constants.TransmisionContingencia, decoded.Identificacion.TipoOperacion)
			require.NotNil(t, decoded.Identificacion.TipoContingencia)
			assert.Equal(t, int(tc.contingencyType), *decoded.Identificacion.TipoContingencia)
			assert.Equal(t, "AB123456-1234-1234-1234-AB1234567890", decoded.Identificacion.CodigoGeneracion)
			if tc.wantReason {
				assert.NotNil(t, decoded.Identificacion.MotivoContin)
			} else {
				assert.Nil(t, decoded.Identificacion.MotivoContin)
			}
		})
	}
}

// TestStoreDocumentInContingencyFailsWhenSignerFails checks that a deferred document is never stored unsigned.
func TestStoreDocumentInContingencyFailsWhenSignerFails(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	svc := contingency.NewContingencyManager(nil, mockDTE, mocks.NewMockContingencyRepositoryPort(ctrl), nil, nil, nil,
		&fakeContingencySigner{err: errors.New("signer down")}, nil, mocks.NewMockContingencyEventSender(ctrl),
		mocks.NewMockSequentialNumberManager(ctrl), nil, nil)

	deferred, err := svc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
		constants.FacturaElectronica, constants.NoDisponibilidadMH, "Servicio no disponible")
	require.Error(t, err)
	assert.Nil(t, deferred)
	assert.Contains(t, err.Error(), "StoreDocumentInContingency")
}

func retransmissionFixture(t *testing.T, ctrl *gomock.Controller, signer *fakeContingencySigner, batch *fakeBatchTransmitter,
	docs []coreDte.ContingencyDocument, events *[][]coreDte.ContingencyDocument) contingency.ContingencyManager {
	t.Helper()
	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().GetPending(gomock.Any(), gomock.Any()).Return(docs, nil)
	mockEvent := mocks.NewMockContingencyEventSender(ctrl)
	mockEvent.EXPECT().PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, group []coreDte.ContingencyDocument) error {
			*events = append(*events, group)
			return nil
		}).AnyTimes()
	mockAuth := mocks.NewMockAuthManager(ctrl)
	mockAuth.EXPECT().GetBranchByBranchID(gomock.Any(), gomock.Any()).
		Return(&user.BranchOffice{ID: 1, User: &user.User{ID: 7, NIT: "06140101010000"}}, nil).AnyTimes()
	mockCache := mocks.NewMockCacheManager(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).Return(`{"IssuedAt":1,"ExpiresAt":4102444800}`, nil).AnyTimes()
	mockCache.EXPECT().GetCredentials(gomock.Any()).Return(&authModels.HaciendaCredentials{Username: "u", Password: "p"}, nil).AnyTimes()
	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().UpdateDTE(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	return contingency.NewContingencyManager(mockAuth, mockDTE, mockRepo, nil, mockCache, fakeContingencyTokens{}, signer, batch,
		mockEvent, mocks.NewMockSequentialNumberManager(ctrl), nil, &transmitterModels.TransmissionConfig{BatchSize: 100})
}

func pendingFromDeferred(id, dteType string, contingencyType int8, deferred *contingency.DeferredDocument) coreDte.ContingencyDocument {
	payload, _ := json.Marshal(deferred.Document)
	return coreDte.ContingencyDocument{
		ID:              "contingency-" + id,
		DocumentID:      id,
		BranchID:        1,
		ContingencyType: contingencyType,
		CreatedAt:       time.Now(),
		Document: &coreDte.DTEDetails{
			ID:             id,
			DTEType:        dteType,
			JSONData:       string(payload),
			SignedDocument: deferred.SignedDocument,
		},
		Branch: &user.BranchOffice{ID: 1, User: &user.User{ID: 7, NIT: "06140101010000"}},
	}
}

// TestContingencyBatchTransmitsTheDeliveredDocument checks, field by field and by signature, that the document sent
// in the contingency batch is the one delivered to the consumer, and that it is not signed again.
func TestContingencyBatchTransmitsTheDeliveredDocument(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeSigner := &fakeContingencySigner{}
	storeSvc, _, _ := storeDeferredFixture(t, ctrl, storeSigner)
	deferred, err := storeSvc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
		constants.FacturaElectronica, constants.NoDisponibilidadMH, "Servicio no disponible")
	require.NoError(t, err)
	delivered := deferred.Response()

	retransmitSigner := &fakeContingencySigner{}
	batch := &fakeBatchTransmitter{}
	var events [][]coreDte.ContingencyDocument
	pending := []coreDte.ContingencyDocument{pendingFromDeferred("AB123456-1234-1234-1234-AB1234567890", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred)}
	svc := retransmissionFixture(t, ctrl, retransmitSigner, batch, pending, &events)

	require.NoError(t, svc.RetransmitPendingDocuments(context.Background()))

	assert.Equal(t, 0, retransmitSigner.callCount(), "the stored JWS is transmitted without signing again")
	require.Len(t, batch.batches[constants.FacturaElectronica], 1)
	transmitted := batch.batches[constants.FacturaElectronica][0]
	assert.Equal(t, delivered["firmaElectronica"], transmitted, "the batch carries the delivered signature")

	transmittedPayload := jwsPayload(t, transmitted)
	deliveredStructure := normalizedJSON(t, delivered)
	delete(deliveredStructure, "firmaElectronica")
	for field, value := range deliveredStructure {
		assert.Equal(t, value, transmittedPayload[field], "field %s differs between the delivered and the transmitted document", field)
	}
	assert.Len(t, transmittedPayload, len(deliveredStructure))
}

// TestContingencyEventsAreGroupedByContingencyType checks that every event only reports documents of one issuer and
// one contingency type, and that documents stored before the JWS was kept are signed when transmitted.
func TestContingencyEventsAreGroupedByContingencyType(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeSvc, _, _ := storeDeferredFixture(t, ctrl, &fakeContingencySigner{})
	deferredFor := func(contingencyType int8) *contingency.DeferredDocument {
		deferred, err := storeSvc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
			constants.FacturaElectronica, contingencyType, "Motivo de la contingencia")
		require.NoError(t, err)
		return deferred
	}

	legacy := pendingFromDeferred("LEGACY-01", constants.CCFElectronico, constants.NoDisponibilidadMH, deferredFor(constants.NoDisponibilidadMH))
	legacy.Document.SignedDocument = ""
	pending := []coreDte.ContingencyDocument{
		pendingFromDeferred("TYPE1-FE", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferredFor(constants.NoDisponibilidadMH)),
		pendingFromDeferred("TYPE5-FE", constants.FacturaElectronica, constants.OtroMotivo, deferredFor(constants.OtroMotivo)),
		legacy,
	}

	retransmitSigner := &fakeContingencySigner{}
	batch := &fakeBatchTransmitter{}
	var events [][]coreDte.ContingencyDocument
	svc := retransmissionFixture(t, ctrl, retransmitSigner, batch, pending, &events)

	require.NoError(t, svc.RetransmitPendingDocuments(context.Background()))

	require.Len(t, events, 2, "one event per contingency type")
	for _, group := range events {
		for _, doc := range group {
			assert.Equal(t, group[0].ContingencyType, doc.ContingencyType, "an event mixes contingency types")
		}
	}
	assert.Len(t, events[0], 2, "type 1 event reports the FE and the CCF")
	assert.Len(t, events[1], 1, "type 5 event reports only its FE")
	assert.Equal(t, int8(constants.OtroMotivo), events[1][0].ContingencyType)
	assert.Equal(t, 1, retransmitSigner.callCount(), "only the legacy document without JWS is signed when transmitted")
}

// TestContingencyEventsAreGroupedByBranch checks that documents of the same issuer and contingency type but different
// branches are reported in separate events, since each event carries the establishment and point of sale of one branch.
func TestContingencyEventsAreGroupedByBranch(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeSvc, _, _ := storeDeferredFixture(t, ctrl, &fakeContingencySigner{})
	deferred, err := storeSvc.StoreDocumentInContingency(buildContingencyCtx(1), json.RawMessage(minimalDTEJSON()),
		constants.FacturaElectronica, constants.NoDisponibilidadMH, "Servicio no disponible")
	require.NoError(t, err)

	mainBranch := pendingFromDeferred("BRANCH1-FE", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred)
	secondBranch := pendingFromDeferred("BRANCH2-FE", constants.FacturaElectronica, constants.NoDisponibilidadMH, deferred)
	secondBranch.BranchID = 2
	secondBranch.Branch = &user.BranchOffice{ID: 2, User: &user.User{ID: 7, NIT: "06140101010000"}}
	sameBranch := pendingFromDeferred("BRANCH1-CCF", constants.CCFElectronico, constants.NoDisponibilidadMH, deferred)

	var events [][]coreDte.ContingencyDocument
	svc := retransmissionFixture(t, ctrl, &fakeContingencySigner{}, &fakeBatchTransmitter{},
		[]coreDte.ContingencyDocument{mainBranch, secondBranch, sameBranch}, &events)

	require.NoError(t, svc.RetransmitPendingDocuments(context.Background()))

	require.Len(t, events, 2, "one event per branch")
	for _, group := range events {
		for _, doc := range group {
			assert.Equal(t, group[0].BranchID, doc.BranchID, "an event mixes branches")
		}
	}
	assert.Len(t, events[0], 2, "branch 1 reports its FE and CCF")
	assert.Len(t, events[1], 1, "branch 2 reports only its FE")
}

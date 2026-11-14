package handlers_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const (
	observationsHTTPCode  = "AD54E9BB-79A3-42AE-B432-EC522C97EFB7"
	observationsHTTPStamp = "2026STAMPFORHTTPOBSERVATIONS"
)

type eventRepositoryStub struct {
	events []coreDTE.DTEEvent
}

func (s *eventRepositoryStub) Create(context.Context, *coreDTE.DTEEvent) error {
	return nil
}

func (s *eventRepositoryStub) ListByRelatedDTE(context.Context, uint, string) ([]coreDTE.DTEEvent, error) {
	return s.events, nil
}

func observationsJWS() string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"identificacion":{"version":3}}`))
	return "header." + payload + ".signature"
}

func observationsStoredDocument(haciendaResponse string) *coreDTE.DTEDocument {
	stamp := observationsHTTPStamp
	return &coreDTE.DTEDocument{
		BranchID:   1,
		DocumentID: observationsHTTPCode,
		CreatedAt:  time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC),
		Details: &coreDTE.DTEDetails{
			ID:               observationsHTTPCode,
			DTEType:          "01",
			ControlNumber:    "DTE-01-00000001-000000000000001",
			ReceptionStamp:   &stamp,
			Transmission:     constants.TransmissionNormal,
			Status:           constants.DocumentReceived,
			SignedDocument:   observationsJWS(),
			ProcessingDate:   "14/11/2026 09:00:00",
			HaciendaResponse: haciendaResponse,
			JSONData:         `{"identificacion":{"version":1}}`,
		},
	}
}

func newObservationsRouter(t *testing.T, repo *mocks.MockDTERepositoryPort, events *eventRepositoryStub) http.Handler {
	t.Helper()
	service := dte_documents.NewDTEService(repo, events)
	handler := handlers.NewDTEHandler(dte.NewDTEConsultUseCase(service), nil, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/dte", handler.GetAll).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/dte/{id}", handler.GetByGenerationCode).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/dte/{id}/archivo", handler.GetArchive).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/dte/{id}/eventos", handler.GetEvents).Methods(http.MethodGet)
	return router
}

func observationsGet(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(context.WithValue(req.Context(), "claims", &models.AuthClaims{BranchID: 1, NIT: "11111111111111"}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeObservationsBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	return body
}

// TestConsultByGenerationCode_ObservationsShape verifies GET /dte/{id} returns data.observations as an array.
func TestConsultByGenerationCode_ObservationsShape(t *testing.T) {
	test.TestMain(t)

	cases := map[string]struct {
		haciendaResponse string
		expected         string
	}{
		"with observations":    {`{"estado":"PROCESADO","observaciones":["Observación uno","second"]}`, `["Observación uno","second"]`},
		"without observations": {`{"estado":"PROCESADO"}`, `[]`},
		"empty response":       {``, `[]`},
		"malformed response":   {`{"observaciones":[`, `[]`},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockDTERepositoryPort(ctrl)
			repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), observationsHTTPCode).Return(observationsStoredDocument(tc.haciendaResponse), nil)

			recorder := observationsGet(t, newObservationsRouter(t, repo, &eventRepositoryStub{}), "/api/v1/dte/"+observationsHTTPCode)

			body := decodeObservationsBody(t, recorder)
			assert.NotContains(t, body, "observations", "observations live inside data for consult responses")
			var data map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(body["data"], &data))
			require.Contains(t, data, "observations")
			assert.JSONEq(t, tc.expected, string(data["observations"]))
			assert.JSONEq(t, `{"identificacion":{"version":1}}`, string(data["json_data"]))
			assert.NotContains(t, string(body["data"]), "hacienda_response")
		})
	}
}

// TestConsultList_ObservationsShape verifies GET /dte returns an observations array in every document.
func TestConsultList_ObservationsShape(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetSummaryStats(gomock.Any(), gomock.Any()).Return(&coreDTE.ListSummary{Total: 3, Received: 3}, nil)
	repo.EXPECT().GetPagedDocuments(gomock.Any(), gomock.Any()).Return([]coreDTE.DTEModelResponse{
		{Status: "RECEIVED", TransmissionType: "NORMAL", Observations: coreDTE.ObservationsFromHaciendaResponse(`{"observaciones":["Observación uno","second"]}`), Document: json.RawMessage(`{"n":1}`)},
		{Status: "RECEIVED", TransmissionType: "NORMAL", Observations: coreDTE.ObservationsFromHaciendaResponse(`{"estado":"PROCESADO"}`), Document: json.RawMessage(`{"n":2}`)},
		{Status: "REJECTED", TransmissionType: "NORMAL", Observations: coreDTE.ObservationsFromHaciendaResponse(``), Document: json.RawMessage(`{"n":3}`)},
	}, nil)

	recorder := observationsGet(t, newObservationsRouter(t, repo, &eventRepositoryStub{}), "/api/v1/dte")

	body := decodeObservationsBody(t, recorder)
	var data struct {
		Documents []map[string]json.RawMessage `json:"documents"`
	}
	require.NoError(t, json.Unmarshal(body["data"], &data))
	require.Len(t, data.Documents, 3)

	expected := []string{`["Observación uno","second"]`, `[]`, `[]`}
	for i, document := range data.Documents {
		require.Contains(t, document, "observations", "document %d", i)
		assert.JSONEq(t, expected[i], string(document["observations"]), "document %d", i)
		assert.Contains(t, document, "document")
	}
}

// TestConsultList_EmptyPageKeepsDocumentsArray verifies an empty result is an empty documents array, not null.
func TestConsultList_EmptyPageKeepsDocumentsArray(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetSummaryStats(gomock.Any(), gomock.Any()).Return(&coreDTE.ListSummary{}, nil)

	recorder := observationsGet(t, newObservationsRouter(t, repo, &eventRepositoryStub{}), "/api/v1/dte")

	body := decodeObservationsBody(t, recorder)
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["data"], &data))
	assert.JSONEq(t, `[]`, string(data["documents"]))
}

// TestConsultEvents_ObservationsShape verifies GET /dte/{id}/eventos returns an observations array per event and that
// a rejected event has no archive.
func TestConsultEvents_ObservationsShape(t *testing.T) {
	test.TestMain(t)

	acceptedStamp := "2026EVENTSTAMP"
	events := &eventRepositoryStub{events: []coreDTE.DTEEvent{
		{
			GenerationCode: "EVENT-ACCEPTED", BranchID: 1, EventType: coreDTE.EventInvalidation,
			SignedDocument: observationsJWS(), ReceptionStamp: &acceptedStamp, Status: constants.DocumentReceived,
			HaciendaResponse: `{"estado":"PROCESADO","observaciones":["Observación del evento"]}`,
		},
		{
			GenerationCode: "EVENT-ACCEPTED-CLEAN", BranchID: 1, EventType: coreDTE.EventInvalidation,
			SignedDocument: observationsJWS(), ReceptionStamp: &acceptedStamp, Status: constants.DocumentReceived,
			HaciendaResponse: `{"estado":"PROCESADO"}`,
		},
		{
			GenerationCode: "EVENT-REJECTED", BranchID: 1, EventType: coreDTE.EventInvalidation,
			SignedDocument: observationsJWS(), Status: constants.DocumentRejected,
			HaciendaResponse: `{"estado":"RECHAZADO","observaciones":["campo invalido","segundo"]}`,
		},
		{
			GenerationCode: "EVENT-LEGACY", BranchID: 1, EventType: coreDTE.EventInvalidation, Status: constants.DocumentReceived,
		},
	}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), observationsHTTPCode).Return(observationsStoredDocument(``), nil)

	recorder := observationsGet(t, newObservationsRouter(t, repo, events), "/api/v1/dte/"+observationsHTTPCode+"/eventos")

	body := decodeObservationsBody(t, recorder)
	var items []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["data"], &items))
	require.Len(t, items, 4)

	expected := []string{`["Observación del evento"]`, `[]`, `["campo invalido","segundo"]`, `[]`}
	for i, item := range items {
		require.Contains(t, item, "observations", "event %d", i)
		assert.JSONEq(t, expected[i], string(item["observations"]), "event %d", i)
	}

	assert.Contains(t, items[0], "archivo_dte")
	assert.Contains(t, items[1], "archivo_dte")
	assert.NotContains(t, items[2], "archivo_dte", "a rejected event has no archive")
	assert.NotContains(t, items[3], "archivo_dte")
	assert.Contains(t, items[2], "hacienda_response")
}

// TestConsultEvents_EmptyListIsArray verifies a document without events returns an empty array.
func TestConsultEvents_EmptyListIsArray(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), observationsHTTPCode).Return(observationsStoredDocument(``), nil)

	recorder := observationsGet(t, newObservationsRouter(t, repo, &eventRepositoryStub{}), "/api/v1/dte/"+observationsHTTPCode+"/eventos")

	body := decodeObservationsBody(t, recorder)
	assert.JSONEq(t, `[]`, string(body["data"]))
}

// TestConsultArchive_BodyHasNoObservations verifies GET /dte/{id}/archivo returns the raw archive without wrapping it
// or adding observations, even when Hacienda sent some.
func TestConsultArchive_BodyHasNoObservations(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), observationsHTTPCode).
		Return(observationsStoredDocument(`{"estado":"PROCESADO","observaciones":["Observación uno"]}`), nil)

	recorder := observationsGet(t, newObservationsRouter(t, repo, &eventRepositoryStub{}), "/api/v1/dte/"+observationsHTTPCode+"/archivo")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Contains(t, recorder.Header().Get("Content-Disposition"), observationsHTTPCode+".json")

	var archive map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &archive))
	assert.NotContains(t, archive, "observations")
	assert.NotContains(t, archive, "observaciones")
	assert.NotContains(t, archive, "success")
	assert.NotContains(t, archive, "data")
	assert.Contains(t, archive, "identificacion")
	assert.Contains(t, archive, "firmaElectronica")
	assert.JSONEq(t, `"`+observationsHTTPStamp+`"`, string(archive["selloRecibido"]))
	assert.NotContains(t, recorder.Body.String(), "Observación uno")
}

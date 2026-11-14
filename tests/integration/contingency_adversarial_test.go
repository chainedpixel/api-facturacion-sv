package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// contingencyFakeSigner builds a compact JWS whose payload is the signed document and records its calls.
type contingencyFakeSigner struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (s *contingencyFakeSigner) SignDTE(_ context.Context, document json.RawMessage, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString(document) + ".c2lnbmF0dXJl", nil
}

func (s *contingencyFakeSigner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// contingencyDocType describes a document type exercised by the adversarial contingency tests.
type contingencyDocType struct {
	name         string
	endpoint     string
	documentType string
	requestType  interface{}
	request      func() interface{}
	dte          func() (interfaces.DTEDocument, error)
	mapper       func() (mapper.DTEMapper, mapper.ResponseMapperFunc)
}

func contingencyDocTypes() []contingencyDocType {
	factory := mapper.NewMapperFactory()
	return []contingencyDocType{
		{
			name:         "Invoice",
			endpoint:     "/invoice",
			documentType: constants.FacturaElectronica,
			requestType:  &structs.CreateInvoiceRequest{},
			request:      func() interface{} { return fixtures.CreateDefaultInvoiceRequest() },
			dte:          func() (interfaces.DTEDocument, error) { return fixtures.NewDTEBuilder().BuildElectronicInvoice() },
			mapper: func() (mapper.DTEMapper, mapper.ResponseMapperFunc) {
				return factory.CreateInvoiceMapperAdapter(), factory.GetInvoiceResponseMapper()
			},
		},
		{
			name:         "CCF",
			endpoint:     "/ccf",
			documentType: constants.CCFElectronico,
			requestType:  &structs.CreateCreditFiscalRequest{},
			request:      func() interface{} { return fixtures.CreateDefaultCreditFiscalRequest() },
			dte:          func() (interfaces.DTEDocument, error) { return fixtures.NewDTEBuilder().BuildCreditFiscalDocument() },
			mapper: func() (mapper.DTEMapper, mapper.ResponseMapperFunc) {
				return factory.CreateCCFMapperAdapter(), factory.GetCCFResponseMapper()
			},
		},
	}
}

// contingencyOutcome holds everything observed after one request through the real handler and contingency manager.
type contingencyOutcome struct {
	recorder        *httptest.ResponseRecorder
	storedDocument  interface{}
	evidence        *coreDTE.TransmissionEvidence
	dteCreateCalls  int
	repoCreateCalls int
	releaseCalls    int
	signer          *contingencyFakeSigner
}

// runContingencyRequest sends one create request whose transmission fails with transmitErr and whose contingency
// signature fails with signerErr when it is not nil.
func runContingencyRequest(t *testing.T, docType contingencyDocType, transmitErr error, signerErr error) *contingencyOutcome {
	t.Helper()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	outcome := &contingencyOutcome{signer: &contingencyFakeSigner{err: signerErr}}

	mockAuth := mocks.NewMockAuthManager(ctrl)
	mockAuth.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)

	mockDTE, err := docType.dte()
	require.NoError(t, err)
	mockDTEService := mocks.NewMockDTEService(ctrl)
	mockDTEService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(mockDTE, nil)

	mockTransmitter := mocks.NewMockBaseTransmitter(ctrl)
	mockTransmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "test-token", "11111111111111").
		Return(nil, transmitErr)

	mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeq.EXPECT().ReleaseReservation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, string, string, uint) error {
			outcome.releaseCalls++
			return nil
		}).AnyTimes()
	mockSeq.EXPECT().MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	mockDTEManager := mocks.NewMockDTEManager(ctrl)
	mockDTEManager.EXPECT().
		Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, gomock.Any()).
		DoAndReturn(func(_ context.Context, document interface{}, _ string, _ string, given *coreDTE.TransmissionEvidence) error {
			outcome.dteCreateCalls++
			outcome.storedDocument = document
			outcome.evidence = given
			return nil
		}).AnyTimes()

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *coreDTE.ContingencyDocument) error {
			outcome.repoCreateCalls++
			return nil
		}).AnyTimes()

	manager := contingency.NewContingencyManager(nil, mockDTEManager, mockRepo, nil, nil, nil, outcome.signer, nil,
		mocks.NewMockContingencyEventSender(ctrl), mockSeq, nil, nil)

	requestMapper, responseMapper := docType.mapper()
	useCase := dte.NewGenericDTEUseCase(mockAuth, mockDTEManager, mockTransmitter, mockDTEService, mockSeq,
		requestMapper, responseMapper, nil, nil)

	handler := handlers.NewGenericDTEHandler(helpers.NewContingencyHandler(manager))
	handler.RegisterDocument(docType.endpoint, helpers.DocumentConfig{
		DocumentType:    docType.documentType,
		UseCase:         useCase,
		RequestType:     docType.requestType,
		UsesContingency: true,
	})

	body, err := json.Marshal(docType.request())
	require.NoError(t, err)
	req, err := http.NewRequest("POST", "/api/v1"+docType.endpoint, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "claims", &models.AuthClaims{BranchID: 1, NIT: "11111111111111"})
	ctx = context.WithValue(ctx, "token", "test-token")
	req = req.WithContext(ctx)

	outcome.recorder = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1"+docType.endpoint, handler.HandleCreate).Methods("POST")
	router.ServeHTTP(outcome.recorder, req)

	return outcome
}

type contingencyEnvelope struct {
	Success        bool                   `json:"success"`
	Data           map[string]interface{} `json:"data"`
	ReceptionStamp *string                `json:"reception_stamp"`
	QRLink         *string                `json:"qr_link"`
	Error          map[string]interface{} `json:"error"`
}

func decodeContingencyEnvelope(t *testing.T, outcome *contingencyOutcome) contingencyEnvelope {
	t.Helper()
	var envelope contingencyEnvelope
	require.NoError(t, json.Unmarshal(outcome.recorder.Body.Bytes(), &envelope), outcome.recorder.Body.String())
	return envelope
}

func contingencyIdentification(t *testing.T, data map[string]interface{}) map[string]interface{} {
	t.Helper()
	identification, ok := data["identificacion"].(map[string]interface{})
	require.True(t, ok, "identificacion must be present")
	return identification
}

func contingencyJWSPayload(t *testing.T, jws string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(jws, ".")
	require.Len(t, parts, 3, "firmaElectronica must be a compact JWS")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func contingencyNormalized(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &normalized))
	return normalized
}

func contingencyWithoutSignature(data map[string]interface{}) map[string]interface{} {
	trimmed := make(map[string]interface{}, len(data))
	for key, value := range data {
		if key != "firmaElectronica" {
			trimmed[key] = value
		}
	}
	return trimmed
}

func assertFieldByField(t *testing.T, expected map[string]interface{}, actual map[string]interface{}) {
	t.Helper()
	assert.Len(t, actual, len(expected), "both documents must carry the same number of top-level fields")
	for key, value := range expected {
		assert.Equal(t, value, actual[key], "field %s differs", key)
	}
}

func assertNothingStored(t *testing.T, outcome *contingencyOutcome) {
	t.Helper()
	assert.Zero(t, outcome.dteCreateCalls, "DTEManager.Create must not be called")
	assert.Zero(t, outcome.repoCreateCalls, "the contingency repository must not be called")
}

// TestContingencyAdversarial_ForcedServiceUnavailable checks the full deferred response and what is stored.
func TestContingencyAdversarial_ForcedServiceUnavailable(t *testing.T) {
	test.TestMain(t)

	for _, docType := range contingencyDocTypes() {
		t.Run(docType.name, func(t *testing.T) {
			outcome := runContingencyRequest(t, docType, &hacienda_error.HTTPResponseError{
				StatusCode: http.StatusServiceUnavailable, Body: []byte("down"), URL: "http://mh", Method: "POST",
			}, nil)

			require.Equal(t, http.StatusCreated, outcome.recorder.Code, outcome.recorder.Body.String())
			envelope := decodeContingencyEnvelope(t, outcome)
			assert.True(t, envelope.Success)
			assert.Nil(t, envelope.ReceptionStamp, "no reception stamp exists in contingency")
			require.NotNil(t, envelope.QRLink)
			assert.NotEmpty(t, *envelope.QRLink)

			identification := contingencyIdentification(t, envelope.Data)
			assert.EqualValues(t, constants.ModeloFacturacionDiferido, identification["tipoModelo"])
			assert.EqualValues(t, constants.TransmisionContingencia, identification["tipoOperacion"])
			assert.EqualValues(t, constants.NoDisponibilidadMH, identification["tipoContingencia"])
			assert.Nil(t, identification["motivoContin"], "motivoContin is null for contingency types 1 to 4")
			assert.NotContains(t, envelope.Data, "selloRecibido")

			signature, ok := envelope.Data["firmaElectronica"].(string)
			require.True(t, ok, "firmaElectronica must be present")
			require.NotEmpty(t, signature)

			delivered := contingencyWithoutSignature(envelope.Data)
			assertFieldByField(t, delivered, contingencyJWSPayload(t, signature))

			require.Equal(t, 1, outcome.dteCreateCalls)
			require.Equal(t, 1, outcome.repoCreateCalls)
			assertFieldByField(t, delivered, contingencyNormalized(t, outcome.storedDocument))
			require.NotNil(t, outcome.evidence)
			assert.Equal(t, signature, outcome.evidence.SignedDocument)
			assert.Nil(t, outcome.evidence.ReceptionStamp)
			assert.Equal(t, 1, outcome.signer.callCount(), "the deferred document is signed exactly once")
			assert.Zero(t, outcome.releaseCalls, "the reservation is kept when the document goes to contingency")
		})
	}
}

// TestContingencyAdversarial_ErrorsOutsideContingency checks that errors which do not call for contingency are
// answered as errors and nothing is signed or stored.
func TestContingencyAdversarial_ErrorsOutsideContingency(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation error", dte_errors.NewValidationError("RequiredField", "Request->Receiver"), http.StatusBadRequest},
		{"dte error", dte_errors.NewDTEErrorSimple("InvalidItemSaleAmount", 1, 2.0, 3.0), http.StatusBadRequest},
		{"service error", shared_error.NewGeneralServiceError("Repo", "Create", "duplicate", nil), http.StatusBadRequest},
		{"hacienda rejected", &hacienda_error.HaciendaResponseError{Status: "RECHAZADO", StatusCode: http.StatusBadRequest, Code: "801", Description: "rejected"}, http.StatusBadRequest},
		{"hacienda rejected with unavailable status code", &hacienda_error.HaciendaResponseError{Status: "RECHAZADO", StatusCode: http.StatusServiceUnavailable, Code: "802", Description: "rejected"}, http.StatusBadRequest},
		{"wrapped validation error", fmt.Errorf("wrapped: %w", dte_errors.NewValidationError("RequiredField", "Request->Receiver")), http.StatusInternalServerError},
		{"wrapped service error", fmt.Errorf("wrapped: %w", shared_error.NewGeneralServiceError("Repo", "Create", "duplicate", nil)), http.StatusInternalServerError},
	}

	for _, docType := range contingencyDocTypes() {
		for _, tc := range cases {
			t.Run(docType.name+"/"+tc.name, func(t *testing.T) {
				outcome := runContingencyRequest(t, docType, tc.err, nil)

				assert.NotEqual(t, http.StatusCreated, outcome.recorder.Code, outcome.recorder.Body.String())
				assert.Equal(t, tc.wantStatus, outcome.recorder.Code, outcome.recorder.Body.String())
				envelope := decodeContingencyEnvelope(t, outcome)
				assert.False(t, envelope.Success)
				assert.NotNil(t, envelope.Error)
				assert.NotContains(t, envelope.Data, "firmaElectronica")
				assertNothingStored(t, outcome)
				assert.Zero(t, outcome.signer.callCount(), "nothing is signed when the error does not call for contingency")
			})
		}
	}
}

// TestContingencyAdversarial_SignerFailure checks that a signing failure answers the original transmission error,
// never a created response, and stores nothing.
func TestContingencyAdversarial_SignerFailure(t *testing.T) {
	test.TestMain(t)

	for _, docType := range contingencyDocTypes() {
		t.Run(docType.name, func(t *testing.T) {
			outcome := runContingencyRequest(t, docType, &hacienda_error.HTTPResponseError{
				StatusCode: http.StatusServiceUnavailable, Body: []byte("down"), URL: "http://mh", Method: "POST",
			}, errors.New("signer unreachable"))

			assert.NotEqual(t, http.StatusCreated, outcome.recorder.Code)
			assert.Equal(t, http.StatusServiceUnavailable, outcome.recorder.Code, "the original transmission error is answered")
			envelope := decodeContingencyEnvelope(t, outcome)
			assert.False(t, envelope.Success)
			require.NotNil(t, envelope.Error)
			assert.Equal(t, "HACIENDA_503", envelope.Error["code"])
			assert.NotContains(t, envelope.Data, "firmaElectronica")
			assert.Equal(t, 1, outcome.signer.callCount())
			assertNothingStored(t, outcome)
		})
	}
}

// TestContingencyAdversarial_ErrorClassification checks the contingency type and reason produced for each kind of
// transmission failure that calls for contingency.
func TestContingencyAdversarial_ErrorClassification(t *testing.T) {
	test.TestMain(t)

	dialError := func(message string) error {
		return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New(message)}
	}

	cases := []struct {
		name     string
		err      error
		wantType int
	}{
		{"http 503", &hacienda_error.HTTPResponseError{StatusCode: http.StatusServiceUnavailable}, constants.NoDisponibilidadMH},
		{"http 502", &hacienda_error.HTTPResponseError{StatusCode: http.StatusBadGateway}, constants.NoDisponibilidadMH},
		{"http 504", &hacienda_error.HTTPResponseError{StatusCode: http.StatusGatewayTimeout}, constants.NoDisponibilidadMH},
		{"http 500 is unclassified", &hacienda_error.HTTPResponseError{StatusCode: http.StatusInternalServerError}, constants.OtroMotivo},
		{"http 429 is unclassified", &hacienda_error.HTTPResponseError{StatusCode: http.StatusTooManyRequests}, constants.OtroMotivo},
		{"connection refused", dialError("connection refused"), constants.FallaConexionSistema},
		{"connection reset", dialError("connection reset by peer"), constants.FallaConexionSistema},
		{"wrapped connection refused", fmt.Errorf("transmit: %w", dialError("connection refused")), constants.FallaConexionSistema},
		{"no such host", dialError("no such host"), constants.FallaServicioInternet},
		{"network unreachable", dialError("network unreachable"), constants.FallaServicioInternet},
		{"unknown network failure", dialError("something odd"), constants.FallaConexionSistema},
		{"network failure from the signer service", fmt.Errorf("signer service: %w", dialError("connection refused")), constants.OtroMotivo},
		{"network failure from redis", fmt.Errorf("redis: %w", dialError("connection refused")), constants.OtroMotivo},
		{"deadline exceeded", fmt.Errorf("transmit: %w", context.DeadlineExceeded), constants.NoDisponibilidadMH},
		{"canceled", context.Canceled, constants.NoDisponibilidadMH},
		{"hacienda unavailable", &hacienda_error.HaciendaResponseError{Status: "ERROR", StatusCode: http.StatusServiceUnavailable, Code: "503", Description: "unavailable"}, constants.NoDisponibilidadMH},
		{"hacienda maintenance", &hacienda_error.HaciendaResponseError{Status: "ERROR", StatusCode: http.StatusInternalServerError, Code: "99", Description: "Sistema en mantenimiento"}, constants.NoDisponibilidadMH},
		{"unclassified plain error", errors.New("boom"), constants.OtroMotivo},
	}

	docType := contingencyDocTypes()[0]
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome := runContingencyRequest(t, docType, tc.err, nil)

			require.Equal(t, http.StatusCreated, outcome.recorder.Code, outcome.recorder.Body.String())
			envelope := decodeContingencyEnvelope(t, outcome)
			identification := contingencyIdentification(t, envelope.Data)
			assert.EqualValues(t, tc.wantType, identification["tipoContingencia"])
			assert.EqualValues(t, constants.ModeloFacturacionDiferido, identification["tipoModelo"])
			assert.EqualValues(t, constants.TransmisionContingencia, identification["tipoOperacion"])

			if tc.wantType == constants.OtroMotivo {
				reason, ok := identification["motivoContin"].(string)
				require.True(t, ok, "type 5 must carry a reason")
				assert.NotEmpty(t, strings.TrimSpace(reason))
			} else {
				assert.Nil(t, identification["motivoContin"])
			}

			signature, _ := envelope.Data["firmaElectronica"].(string)
			require.NotEmpty(t, signature)
			assertFieldByField(t, contingencyWithoutSignature(envelope.Data), contingencyJWSPayload(t, signature))
			assert.Equal(t, 1, outcome.dteCreateCalls)
			assert.Equal(t, signature, outcome.evidence.SignedDocument)
		})
	}
}

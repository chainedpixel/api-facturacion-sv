package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/config/drivers"
	appdte "github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/user"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/contingency"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/batch"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const evidenceSignedJWS = "header.payload.signature"

type evidenceSigner struct {
	err error
}

func (s *evidenceSigner) SignDTE(_ context.Context, _ json.RawMessage, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return evidenceSignedJWS, nil
}

type evidenceTransmitter struct {
	result *transmitterModels.TransmitResult
	err    error
}

func (t *evidenceTransmitter) Transmit(context.Context, interface{}, string, string) (*transmitterModels.TransmitResult, error) {
	return t.result, t.err
}

func (t *evidenceTransmitter) CheckDocumentStatus(context.Context, interface{}, string, string) (*transmitterModels.TransmitResult, error) {
	return nil, errors.New("not used")
}

func (t *evidenceTransmitter) SendToHacienda(context.Context, *transmitterModels.HaciendaRequest, string) (*transmitterModels.HaciendaResponse, error) {
	return nil, errors.New("not used")
}

// TestTransmissionError_UnwrapsAndKeepsMessage verifies the wrapper keeps the message and the inner error reachable.
func TestTransmissionError_UnwrapsAndKeepsMessage(t *testing.T) {
	inner := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{Status: "RECHAZADO", MessageCode: "095"}, 400)
	wrapped := fmt.Errorf("outer: %w", &ports.TransmissionError{Err: inner, SignedDocument: "jws"})

	var transmissionErr *ports.TransmissionError
	require.True(t, errors.As(wrapped, &transmissionErr))
	assert.Equal(t, "jws", transmissionErr.SignedDocument)
	assert.Equal(t, inner.Error(), transmissionErr.Error())

	var haciendaErr *hacienda_error.HaciendaResponseError
	require.True(t, errors.As(wrapped, &haciendaErr))
	assert.Same(t, inner, haciendaErr)
	assert.True(t, errors.Is(wrapped, inner))
}

// TestRetryTransmission_ErrorsAfterSigningCarryTheJWS verifies transmission errors are wrapped with the transmitted JWS.
func TestRetryTransmission_ErrorsAfterSigningCarryTheJWS(t *testing.T) {
	test.TestMain(t)

	rejection := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{
		Status: hacienda_error.StatusRejected, MessageCode: "095", Observations: []string{"bad"},
	}, 400)
	httpErr := &hacienda_error.HTTPResponseError{StatusCode: 503, Method: "POST", URL: "http://mh", Body: []byte("down")}
	network := &hacienda_error.NetworkError{Operation: "transmit", OriginalError: errors.New("refused")}
	plain := errors.New("plain failure")

	for name, transmitErr := range map[string]error{
		"hacienda rejection": rejection,
		"http error":         httpErr,
		"network error":      network,
		"plain error":        plain,
	} {
		t.Run(name, func(t *testing.T) {
			transmitter := appdte.NewBaseTransmitter(&evidenceTransmitter{err: transmitErr}, &evidenceSigner{})

			result, err := transmitter.RetryTransmission(context.Background(), map[string]string{"a": "b"}, "token", "NIT")
			require.Error(t, err)
			assert.Nil(t, result)

			var transmissionErr *ports.TransmissionError
			require.True(t, errors.As(err, &transmissionErr))
			assert.Equal(t, evidenceSignedJWS, transmissionErr.SignedDocument)
			assert.Same(t, transmitErr, errors.Unwrap(err), "the inner error is not altered")
			assert.True(t, errors.Is(err, transmitErr))
			assert.Equal(t, transmitErr.Error(), err.Error())
		})
	}

	t.Run("inner types remain reachable with errors.As", func(t *testing.T) {
		transmitter := appdte.NewBaseTransmitter(&evidenceTransmitter{err: rejection}, &evidenceSigner{})
		_, err := transmitter.RetryTransmission(context.Background(), map[string]string{}, "token", "NIT")

		var haciendaErr *hacienda_error.HaciendaResponseError
		require.True(t, errors.As(err, &haciendaErr))
		assert.Equal(t, []string{"bad"}, haciendaErr.Observations)

		var httpTarget *hacienda_error.HTTPResponseError
		assert.False(t, errors.As(err, &httpTarget))
	})
}

// TestRetryTransmission_SigningErrorsAreNotWrapped verifies a signing failure is returned as is, without a JWS.
func TestRetryTransmission_SigningErrorsAreNotWrapped(t *testing.T) {
	test.TestMain(t)

	signErr := fmt.Errorf("%w: boom", ports.ErrSigning)
	transmitter := appdte.NewBaseTransmitter(&evidenceTransmitter{}, &evidenceSigner{err: signErr})

	_, err := transmitter.RetryTransmission(context.Background(), map[string]string{}, "token", "NIT")
	require.Error(t, err)
	assert.Same(t, signErr, err)

	var transmissionErr *ports.TransmissionError
	assert.False(t, errors.As(err, &transmissionErr))
	assert.True(t, errors.Is(err, ports.ErrSigning))
}

// TestRetryTransmission_MarshalErrorIsNotWrapped verifies a document that cannot be serialized is not wrapped.
func TestRetryTransmission_MarshalErrorIsNotWrapped(t *testing.T) {
	test.TestMain(t)

	transmitter := appdte.NewBaseTransmitter(&evidenceTransmitter{}, &evidenceSigner{})
	_, err := transmitter.RetryTransmission(context.Background(), make(chan int), "token", "NIT")
	require.Error(t, err)

	var transmissionErr *ports.TransmissionError
	assert.False(t, errors.As(err, &transmissionErr))
}

// TestRetryTransmission_SuccessReturnsNoErrorAndTheJWS verifies a received document returns no error and keeps the JWS.
func TestRetryTransmission_SuccessReturnsNoErrorAndTheJWS(t *testing.T) {
	test.TestMain(t)

	stamp := "STAMP"
	transmitter := appdte.NewBaseTransmitter(&evidenceTransmitter{result: &transmitterModels.TransmitResult{
		Status: appdte.ReceivedStatus, ReceptionStamp: &stamp,
	}}, &evidenceSigner{})

	result, err := transmitter.RetryTransmission(context.Background(), map[string]string{}, "token", "NIT")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, evidenceSignedJWS, result.SignedDocument)
}

type evidenceBatchHacienda struct {
	mu       sync.Mutex
	response string
}

func (h *evidenceBatchHacienda) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(h.response))
}

func evidenceDBConnection(t *testing.T) *drivers.DbConnection {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectPing()
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)
	return &drivers.DbConnection{Db: gormDB}
}

// TestBatchTransmitter_VerifyStatus_StoresPerDocumentResponses verifies processed and rejected documents reach
// UpdateBatch with their Hacienda response, including the observations.
func TestBatchTransmitter_VerifyStatus_StoresPerDocumentResponses(t *testing.T) {
	test.TestMain(t)

	server := httptest.NewServer(&evidenceBatchHacienda{response: `{
		"procesados":[{"codigoGeneracion":"GEN-OK","estado":"PROCESADO","selloRecibido":"STAMP-OK","fhProcesamiento":"14/11/2026 10:00:00",
			"descripcionMsg":"RECIBIDO","observaciones":["warning one","advertencia dos"]},
			{"codigoGeneracion":"GEN-UNKNOWN","estado":"PROCESADO","selloRecibido":"STAMP-X"}],
		"rechazados":[{"codigoGeneracion":"GEN-BAD","estado":"RECHAZADO","clasificaMsg":"10","codigoMsg":"095",
			"descripcionMsg":"RECHAZADO","observaciones":["error uno","error dos"],"fhProcesamiento":"14/11/2026 10:00:01"},
			{"codigoGeneracion":"GEN-BAD-EMPTY","estado":"RECHAZADO","descripcionMsg":"RECHAZADO"}]}`})
	defer server.Close()

	previous := config.MHPaths.LoteReceptionConsultURL
	config.MHPaths.LoteReceptionConsultURL = server.URL
	defer func() { config.MHPaths.LoteReceptionConsultURL = previous }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockContingencyRepositoryPort(ctrl)
	seq := mocks.NewMockSequentialNumberManager(ctrl)

	type update struct {
		ids          []string
		observations []string
		results      map[string]dte.ReceptionResult
		status       string
	}
	var updates []update
	repo.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "batch-1", "mh-1", gomock.Any()).
		DoAndReturn(func(_ context.Context, ids, observations []string, results map[string]dte.ReceptionResult, _, _, status string) error {
			updates = append(updates, update{ids, observations, results, status})
			return nil
		}).Times(2)
	seq.EXPECT().ConfirmReservationByDocumentID(gomock.Any(), "doc-ok").Return(nil)
	seq.EXPECT().ReleaseReservationByDocumentID(gomock.Any(), "doc-bad", gomock.Any(), gomock.Any()).Return(nil)
	seq.EXPECT().ReleaseReservationByDocumentID(gomock.Any(), "doc-bad-empty", gomock.Any(), gomock.Any()).Return(nil)

	service := batch.NewBatchTransmitterService(&stubHaciendaAuth{}, nil, repo, seq, minimalTransmissionConfig(),
		&noopTimeProvider{}, evidenceDBConnection(t))

	docs := map[string]dte.ContingencyDocument{
		"GEN-OK":        {ID: "c-ok", DocumentID: "doc-ok"},
		"GEN-BAD":       {ID: "c-bad", DocumentID: "doc-bad"},
		"GEN-BAD-EMPTY": {ID: "c-bad-empty", DocumentID: "doc-bad-empty"},
	}

	started := time.Now()
	err := service.VerifyContingencyBatchStatus(context.Background(), "batch-1", "mh-1", "token", 1, docs)
	require.NoError(t, err)
	assert.Less(t, time.Since(started), 30*time.Second)
	require.Len(t, updates, 2)

	processed := updates[0]
	assert.Equal(t, constants.DocumentReceived, processed.status)
	assert.Equal(t, []string{"c-ok"}, processed.ids)
	require.Contains(t, processed.results, "c-ok")
	assert.NotContains(t, processed.results, "c-unknown", "documents outside the batch map are ignored")
	okResult := processed.results["c-ok"]
	assert.Equal(t, "STAMP-OK", okResult.ReceptionStamp)
	assert.Equal(t, "14/11/2026 10:00:00", okResult.ProcessingDate)
	assert.Equal(t, []string{"warning one", "advertencia dos"}, dte.ObservationsFromHaciendaResponse(okResult.HaciendaResponse))

	rejected := updates[1]
	assert.Equal(t, constants.DocumentRejected, rejected.status)
	assert.ElementsMatch(t, []string{"c-bad", "c-bad-empty"}, rejected.ids)
	badResult := rejected.results["c-bad"]
	assert.Empty(t, badResult.ReceptionStamp, "a rejected document has no stamp")
	assert.Empty(t, badResult.ProcessingDate)
	assert.Equal(t, []string{"error uno", "error dos"}, dte.ObservationsFromHaciendaResponse(badResult.HaciendaResponse))
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(badResult.HaciendaResponse), &raw))
	assert.Equal(t, "RECHAZADO", raw["estado"])
	assert.Equal(t, "095", raw["codigoMsg"])

	emptyResult := rejected.results["c-bad-empty"]
	assert.NotEmpty(t, emptyResult.HaciendaResponse)
	assert.Equal(t, []string{}, dte.ObservationsFromHaciendaResponse(emptyResult.HaciendaResponse))
}

type evidenceTokenManager struct{}

func (evidenceTokenManager) GenerateToken(*models.AuthClaims, time.Duration) (string, error) {
	return "", nil
}
func (evidenceTokenManager) ValidateToken(string) (*models.AuthClaims, error) { return nil, nil }
func (evidenceTokenManager) RevokeToken(string) error                         { return nil }
func (evidenceTokenManager) SaveTimestampsForContingency(time.Time, time.Time, time.Duration, *models.AuthClaims) error {
	return nil
}
func (evidenceTokenManager) GetSecretKey() string { return "secret" }

type contingencyEventHarness struct {
	service  *contingency.ContingencyEventService
	dteMgr   *mocks.MockDTEManager
	repo     *mocks.MockContingencyRepositoryPort
	received []*dte.DTEEvent
}

func newContingencyEventHarness(t *testing.T, ctrl *gomock.Controller, signer *evidenceSigner, registerErr error) *contingencyEventHarness {
	t.Helper()

	harness := &contingencyEventHarness{
		dteMgr: mocks.NewMockDTEManager(ctrl),
		repo:   mocks.NewMockContingencyRepositoryPort(ctrl),
	}
	authManager := mocks.NewMockAuthManager(ctrl)
	cache := mocks.NewMockCacheManager(ctrl)

	started := time.Now().Add(-time.Hour)
	harness.repo.EXPECT().GetFirstContingencyTimestamp(gomock.Any(), uint(1)).Return(&started, nil).AnyTimes()
	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil).AnyTimes()
	authManager.EXPECT().GetByNIT(gomock.Any(), gomock.Any()).Return(&user.User{ID: 7, NIT: "11111111111111"}, nil).AnyTimes()
	cache.EXPECT().Get("token:timestamps:7").Return(
		fmt.Sprintf(`{"IssuedAt":%d,"ExpiresAt":%d}`, time.Now().Unix(), time.Now().Add(time.Hour).Unix()), nil).AnyTimes()
	cache.EXPECT().GetCredentials(gomock.Any()).Return(&models.HaciendaCredentials{Username: "u", Password: "p"}, nil).AnyTimes()
	harness.dteMgr.EXPECT().RegisterEvent(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event *dte.DTEEvent) error {
			harness.received = append(harness.received, event)
			return registerErr
		}).AnyTimes()

	harness.service = contingency.NewContingencyEventService(authManager, &stubHaciendaAuth{}, cache, evidenceTokenManager{},
		signer, harness.repo, &noopTimeProvider{}, evidenceDBConnection(t), harness.dteMgr)
	return harness
}

func contingencyEvidenceDocs() []dte.ContingencyDocument {
	return []dte.ContingencyDocument{{
		ID: "c-1", DocumentID: "doc-1", BranchID: 1, ContingencyType: 1, Reason: "reason",
		Document: &dte.DTEDetails{ID: "doc-1", DTEType: "01"},
	}}
}

func serveContingency(t *testing.T, status int, body string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	previous := config.MHPaths.ContingencyURL
	config.MHPaths.ContingencyURL = server.URL
	t.Cleanup(func() {
		config.MHPaths.ContingencyURL = previous
		server.Close()
	})
}

// TestContingencyEventService_RejectedResponseRegistersRejectedEvent verifies a response without stamp is stored as REJECTED.
func TestContingencyEventService_RejectedResponseRegistersRejectedEvent(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"http 200 with rechazado state", http.StatusOK, `{"estado":"RECHAZADO","mensaje":"validation failed","observaciones":["obs one","obs dos"]}`},
		{"http 400 with validation message", http.StatusBadRequest, `{"estado":"RECHAZADO","mensaje":"Validaciones no superadas","observaciones":["obs one","obs dos"]}`},
		{"http 400 recognized only by the message in upper case", http.StatusBadRequest, `{"mensaje":"VALIDACIONES NO SUPERADAS","observaciones":["obs one","obs dos"]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, nil)
			harness.repo.EXPECT().MarkEventStamped(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			serveContingency(t, tc.status, tc.body)

			err := harness.service.PrepareAndSendContingencyEvent(context.Background(), contingencyEvidenceDocs())
			require.Error(t, err)

			require.Len(t, harness.received, 1)
			event := harness.received[0]
			assert.Equal(t, constants.DocumentRejected, event.Status)
			assert.Equal(t, dte.EventContingency, event.EventType)
			assert.Equal(t, evidenceSignedJWS, event.SignedDocument)
			assert.Nil(t, event.ReceptionStamp)
			assert.Equal(t, uint(1), event.BranchID)
			assert.Equal(t, []string{"obs one", "obs dos"}, dte.ObservationsFromHaciendaResponse(event.HaciendaResponse))
		})
	}
}

// TestContingencyEventService_StampedResponseRegistersReceivedEvent verifies a stamped event is stored as RECEIVED.
func TestContingencyEventService_StampedResponseRegistersReceivedEvent(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, nil)
	harness.repo.EXPECT().MarkEventStamped(gomock.Any(), []string{"c-1"}, gomock.Any()).Return(nil)
	serveContingency(t, http.StatusOK, `{"estado":"RECIBIDO","selloRecibido":"EVENT-STAMP","fechaHora":"14/11/2026 10:00:00","observaciones":["note"]}`)

	require.NoError(t, harness.service.PrepareAndSendContingencyEvent(context.Background(), contingencyEvidenceDocs()))

	require.Len(t, harness.received, 1)
	event := harness.received[0]
	assert.Equal(t, constants.DocumentReceived, event.Status)
	require.NotNil(t, event.ReceptionStamp)
	assert.Equal(t, "EVENT-STAMP", *event.ReceptionStamp)
	assert.Equal(t, []string{"note"}, dte.ObservationsFromHaciendaResponse(event.HaciendaResponse))
}

// TestContingencyEventService_NoEvidenceRegistersNothing verifies failures without a Hacienda response are not stored.
func TestContingencyEventService_NoEvidenceRegistersNothing(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name   string
		status int
		body   string
		signer *evidenceSigner
	}{
		{"non json body", http.StatusOK, "<html>gateway</html>", &evidenceSigner{}},
		{"empty non-200 body", http.StatusBadGateway, "", &evidenceSigner{}},
		{"transient non-200 status with another message", http.StatusServiceUnavailable, `{"mensaje":"Servicio temporalmente no disponible","observaciones":["retry later"]}`, &evidenceSigner{}},
		{"non-200 status with message in upper case but not a rejection", http.StatusInternalServerError, `{"mensaje":"ERROR INTERNO"}`, &evidenceSigner{}},
		{"signing failure", http.StatusOK, `{"estado":"RECIBIDO"}`, &evidenceSigner{err: errors.New("sign down")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			harness := newContingencyEventHarness(t, ctrl, tc.signer, nil)
			harness.repo.EXPECT().MarkEventStamped(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			serveContingency(t, tc.status, tc.body)

			err := harness.service.PrepareAndSendContingencyEvent(context.Background(), contingencyEvidenceDocs())
			require.Error(t, err)
			assert.Empty(t, harness.received)
		})
	}
}

// TestContingencyEventService_RegisterFailureDoesNotHideTheSendError verifies the send error is returned even when
// storing the rejected event fails.
func TestContingencyEventService_RegisterFailureDoesNotHideTheSendError(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, errors.New("database down"))
	serveContingency(t, http.StatusOK, `{"estado":"RECHAZADO","mensaje":"nope","observaciones":["x"]}`)

	err := harness.service.PrepareAndSendContingencyEvent(context.Background(), contingencyEvidenceDocs())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sign and send contingency event")
	assert.NotContains(t, err.Error(), "database down")
	assert.Len(t, harness.received, 1)
}

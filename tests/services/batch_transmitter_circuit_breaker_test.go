package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	transmitterPort "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/batch"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubHaciendaAuth always returns "test-token" for any token request.
type stubHaciendaAuth struct{}

func (s *stubHaciendaAuth) GetOrCreateHaciendaToken(_ context.Context, _ string) (string, error) {
	return "test-token", nil
}

func (s *stubHaciendaAuth) GetOrCreateHaciendaTokenWithCreds(_ context.Context, _ string, _ authModels.HaciendaCredentials) (string, error) {
	return "test-token", nil
}

// noopTimeProvider never sleeps, so retry back-off does not slow tests.
type noopTimeProvider struct{}

func (n *noopTimeProvider) Now() time.Time        { return time.Now() }
func (n *noopTimeProvider) Sleep(_ time.Duration) {}

// minimalTransmissionConfig returns a config with zero retry intervals so no
// actual sleeping occurs during the retry loop.
func minimalTransmissionConfig() *transmitterModels.TransmissionConfig {
	return &transmitterModels.TransmissionConfig{
		Ambient:       "00",
		BatchSize:     50,
		RetryInterval: 0,
		MaxInterval:   0,
		BackoffFactor: 0,
	}
}

// newBatchTransmitterUnderTest constructs a BatchTransmitterService pointing at
// the given test-server URL with a no-op time provider so retries are instant.
func newBatchTransmitterUnderTest(serverURL string) transmitterPort.BatchTransmitterPort {
	config.MHPaths.LoteReceptionURL = serverURL
	return batch.NewBatchTransmitterService(
		&stubHaciendaAuth{},
		nil, nil, nil,
		minimalTransmissionConfig(),
		&noopTimeProvider{},
		nil,
	)
}

// TestBatchTransmitter_CircuitBreakerOpensAfterThreeHTTPFailures verifies that
// after 3 consecutive HTTP 500 responses (= one full retry cycle, MaxAttempts=3),
// the circuit breaker opens and the NEXT call is blocked locally without reaching
// the HTTP server at all, returning "service temporarily unavailable".
func TestBatchTransmitter_CircuitBreakerOpensAfterThreeHTTPFailures(t *testing.T) {
	test.TestMain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newBatchTransmitterUnderTest(server.URL)

	docs := []string{`{"test":"doc"}`}
	creds := authModels.HaciendaCredentials{}

	// First call: MaxAttempts=3, all return HTTP 500 → RecordFailure ×3 → StateOpen.
	_, _, err1 := svc.TransmitBatch(context.Background(), "0614-010101-000-0", "01", docs, "token", creds)
	require.Error(t, err1, "first batch should fail: all 3 attempts return HTTP 500")

	// Second call: circuit is Open → AllowRequest returns false → no HTTP request is made.
	// The error must mention "service temporarily unavailable".
	_, _, err2 := svc.TransmitBatch(context.Background(), "0614-010101-000-0", "01", docs, "token", creds)
	require.Error(t, err2, "second batch should fail: circuit is open")
	assert.Contains(t, err2.Error(), "service temporarily unavailable",
		"circuit-open error must describe why the request was blocked")
}

// TestBatchTransmitter_CircuitBreakerDoesNotBlockOnSuccess verifies that a healthy
// server produces a successful response and does not open the circuit breaker.
func TestBatchTransmitter_CircuitBreakerDoesNotBlockOnSuccess(t *testing.T) {
	test.TestMain(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := transmitterModels.BatchResponse{
			Status:    "PROCESADO",
			BatchCode: "LOTE-0001",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newBatchTransmitterUnderTest(server.URL)

	docs := []string{`{"test":"doc"}`}
	resp, _, err := svc.TransmitBatch(
		context.Background(), "0614-010101-000-0", "01", docs, "token", authModels.HaciendaCredentials{},
	)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "PROCESADO", resp.Status)

	// A second call must also succeed: circuit remains closed.
	resp2, _, err2 := svc.TransmitBatch(
		context.Background(), "0614-010101-000-0", "01", docs, "token", authModels.HaciendaCredentials{},
	)
	require.NoError(t, err2)
	assert.NotNil(t, resp2)
}

// TestBatchTransmitter_CircuitBreakerBlocksWithoutHittingServer verifies that when
// the circuit is open the test HTTP server receives zero additional requests.
func TestBatchTransmitter_CircuitBreakerBlocksWithoutHittingServer(t *testing.T) {
	test.TestMain(t)

	var hitCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newBatchTransmitterUnderTest(server.URL)

	docs := []string{`{"test":"doc"}`}
	creds := authModels.HaciendaCredentials{}

	// First call exhausts all 3 retry attempts → 3 HTTP hits → circuit opens.
	_, _, _ = svc.TransmitBatch(context.Background(), "0614-010101-000-0", "01", docs, "token", creds)
	hitsAfterOpen := atomic.LoadInt32(&hitCount)
	assert.EqualValues(t, 3, hitsAfterOpen, "first call should make exactly 3 HTTP requests (MaxAttempts=3)")

	// Second call: circuit open → AllowRequest=false → no HTTP request at all.
	_, _, _ = svc.TransmitBatch(context.Background(), "0614-010101-000-0", "01", docs, "token", creds)
	hitsAfterBlocked := atomic.LoadInt32(&hitCount)
	assert.EqualValues(t, 3, hitsAfterBlocked, "circuit-open call must not add any new HTTP hits")
}

// TestBatchTransmitter_CircuitBreakerPartialFailuresBelowThreshold verifies that
// two failures (below threshold=3) do not open the circuit; the third attempt in
// the same call succeeds.
func TestBatchTransmitter_CircuitBreakerPartialFailuresBelowThreshold(t *testing.T) {
	test.TestMain(t)

	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		// First two requests fail; third succeeds.
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := transmitterModels.BatchResponse{Status: "PROCESADO", BatchCode: "LOTE-0001"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newBatchTransmitterUnderTest(server.URL)

	docs := []string{`{"test":"doc"}`}
	resp, _, err := svc.TransmitBatch(
		context.Background(), "0614-010101-000-0", "01", docs, "token", authModels.HaciendaCredentials{},
	)
	require.NoError(t, err, "should succeed on the 3rd attempt with circuit still closed")
	assert.NotNil(t, resp)
	// Circuit closed (RecordSuccess called on 3rd attempt).
	// A subsequent call should also succeed (circuit not open).
	atomic.StoreInt32(&callCount, 0) // reset so the server succeeds from attempt 0.
	_, _, err2 := svc.TransmitBatch(
		context.Background(), "0614-010101-000-0", "01", docs, "token", authModels.HaciendaCredentials{},
	)
	require.NoError(t, err2)
}

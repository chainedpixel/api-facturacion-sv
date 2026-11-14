package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	authModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

type advTokenAuth struct {
	failFor string
	calls   atomic.Int64
}

func (a *advTokenAuth) GetOrCreateHaciendaToken(_ context.Context, systemToken string) (string, error) {
	a.calls.Add(1)
	if systemToken == a.failFor {
		return "", errors.New("auth unavailable")
	}
	time.Sleep(time.Duration(len(systemToken)%5) * time.Millisecond)
	return "hacienda-for-" + systemToken, nil
}

func (a *advTokenAuth) GetOrCreateHaciendaTokenWithCreds(_ context.Context, systemToken string, _ authModels.HaciendaCredentials) (string, error) {
	return "hacienda-for-" + systemToken, nil
}

type advSeen struct {
	mu     sync.Mutex
	byCode map[string]string
	hits   atomic.Int64
}

func newAdvServer(t *testing.T) (*httptest.Server, *advSeen) {
	t.Helper()
	seen := &advSeen{byCode: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.hits.Add(1)
		var body struct {
			GenerationCode string `json:"codigoGeneracion"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		seen.mu.Lock()
		seen.byCode[body.GenerationCode] = r.Header.Get("Authorization")
		seen.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transmitterModels.HaciendaResponse{
			Status: "PROCESADO", GenerationCode: body.GenerationCode, ReceptionStamp: "STAMP-" + body.GenerationCode,
			ProcessingDate: "2026-11-13 10:00:00", DescriptionMessage: "RECIBIDO", MessageCode: "001",
		})
	}))
	t.Cleanup(server.Close)
	return server, seen
}

func advDocument(code string) map[string]interface{} {
	return map[string]interface{}{
		"identificacion": map[string]interface{}{
			"version": float64(2), "tipoDte": "01",
			"numeroControl":    "DTE-01-00000000-000000000000001",
			"codigoGeneracion": code,
		},
	}
}

// TestAdversarialMHTransmitterHoldsNoSharedToken verifies the transmitter has no per-instance token state left.
func TestAdversarialMHTransmitterHoldsNoSharedToken(t *testing.T) {
	typeOf := reflect.TypeOf(transmitter.MHTransmitter{})
	for index := 0; index < typeOf.NumField(); index++ {
		assert.NotEqual(t, "HaciendaToken", typeOf.Field(index).Name)
	}
}

// TestAdversarialMHTransmitterConcurrentTokens verifies each concurrent call sends the token of its own system token.
func TestAdversarialMHTransmitterConcurrentTokens(t *testing.T) {
	test.TestMain(t)
	server, seen := newAdvServer(t)

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&advTokenAuth{}, nil)

	const workers = 60
	var group sync.WaitGroup
	start := make(chan struct{})
	expected := map[string]string{}
	var expectedMu sync.Mutex

	for index := 0; index < workers; index++ {
		group.Add(1)
		systemToken := fmt.Sprintf("system-token-%d-%s", index%2, string(rune('a'+index%7)))
		code := fmt.Sprintf("CODE-%03d", index)
		expectedMu.Lock()
		expected[code] = "hacienda-for-" + systemToken
		expectedMu.Unlock()

		go func(useSend bool) {
			defer group.Done()
			<-start
			if useSend {
				_, err := trans.SendToHacienda(context.Background(), &transmitterModels.HaciendaRequest{
					Ambient: "00", Version: 2, Document: advDocument(code), DTEType: "01", GenerationCode: code, URL: server.URL,
				}, systemToken)
				assert.NoError(t, err)
				return
			}
			_, err := trans.CheckDocumentStatus(context.Background(), advDocument(code), "06140101010000", systemToken)
			assert.NoError(t, err)
		}(index%2 == 0)
	}
	close(start)
	group.Wait()

	seen.mu.Lock()
	defer seen.mu.Unlock()
	assert.Len(t, seen.byCode, workers)
	for code, token := range expected {
		assert.Equal(t, token, seen.byCode[code], "request %s must carry its own token", code)
	}
}

// TestAdversarialMHTransmitterAuthFailure verifies no HTTP call is made when the token cannot be obtained.
func TestAdversarialMHTransmitterAuthFailure(t *testing.T) {
	test.TestMain(t)
	server, seen := newAdvServer(t)

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	trans := transmitter.NewMHTransmitter(&advTokenAuth{failFor: "bad"}, nil)

	_, err := trans.CheckDocumentStatus(context.Background(), advDocument("X"), "nit", "bad")
	require.Error(t, err)
	_, err = trans.SendToHacienda(context.Background(), &transmitterModels.HaciendaRequest{Document: advDocument("X"), URL: server.URL}, "bad")
	require.Error(t, err)
	assert.EqualValues(t, 0, seen.hits.Load())

	_, err = trans.CheckDocumentStatus(context.Background(), advDocument("OK"), "nit", "good")
	require.NoError(t, err)
	assert.Equal(t, "hacienda-for-good", seen.byCode["OK"])

	_, err = trans.CheckDocumentStatus(context.Background(), advDocument("AFTER"), "nit", "bad")
	require.Error(t, err)
	assert.NotContains(t, seen.byCode, "AFTER", "a failed token request must not reuse the previous token")
}

// TestAdversarialMHTransmitterCancelledContext verifies a cancelled context is honoured by the status check.
func TestAdversarialMHTransmitterCancelledContext(t *testing.T) {
	test.TestMain(t)
	server, seen := newAdvServer(t)

	originalURL := config.MHPaths.ReceptionConsultURL
	config.MHPaths.ReceptionConsultURL = server.URL
	defer func() { config.MHPaths.ReceptionConsultURL = originalURL }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	trans := transmitter.NewMHTransmitter(&advTokenAuth{}, nil)
	_, err := trans.CheckDocumentStatus(ctx, advDocument("CANCELLED"), "nit", "token")
	assert.Error(t, err)
	assert.EqualValues(t, 0, seen.hits.Load())
}

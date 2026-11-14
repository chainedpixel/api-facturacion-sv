package services

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/service/strategies"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/cache"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/signing"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/tokens"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/middleware"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

const (
	secretPassword  = "TOP-SECRET-HACIENDA-PASSWORD"
	secretAPISecret = "TOP-SECRET-API-SECRET"
	secretToken     = "SECRET.JWT.TOKENVALUE"
	secretHacienda  = "Bearer-HACIENDA-SECRET-TOKEN"
	secretRedisPass = "REDIS-SECRET-PASSWORD"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	test.TestMain(t)
	buffer := &syncBuffer{}
	previous := logs.Logger.Out
	logs.Logger.SetOutput(buffer)
	t.Cleanup(func() { logs.Logger.SetOutput(previous) })
	return buffer
}

func assertNoSecrets(t *testing.T, output string, secrets ...string) {
	t.Helper()
	require.NotEmpty(t, output, "the flow under test must have logged something")
	for _, secret := range secrets {
		assert.NotContains(t, output, secret, "secret value leaked into the logs")
	}
}

type fakeCrypt struct{ fail bool }

func (f fakeCrypt) GenerateAPIKey() (string, error)    { return "", nil }
func (f fakeCrypt) GenerateAPISecret() (string, error) { return "", nil }
func (f fakeCrypt) EncryptStruct(string, models.HaciendaCredentials) (string, error) {
	if f.fail {
		return "", errors.New("encrypt failed")
	}
	return "cipher", nil
}
func (f fakeCrypt) DecryptStruct(string, string) (models.HaciendaCredentials, error) {
	return models.HaciendaCredentials{}, errors.New("decrypt failed")
}
func (f fakeCrypt) GenerateBulkAPIKeys(int) ([]string, []string, error) { return nil, nil, nil }

// TestAdversarialNoSecretsInRedisCacheLogs verifies keys that embed tokens and Redis URLs with passwords never reach the logs.
func TestAdversarialNoSecretsInRedisCacheLogs(t *testing.T) {
	t.Run("operations on token keys", func(t *testing.T) {
		output := captureLogs(t)
		server := miniredis.RunT(t)
		store, err := cache.NewRedisTokenCache(&config.RedisConfig{Host: server.Host(), Port: server.Port()}, fakeCrypt{})
		require.NoError(t, err)

		require.NoError(t, store.Set("token:"+secretToken, []byte("{}"), time.Minute))
		_, _ = store.Get("token:" + secretToken)
		_, _ = store.Get("token:missing-" + secretToken)
		_, _ = store.Get("hacienda:token:" + secretHacienda)
		require.NoError(t, store.Set("hacienda:token:"+secretHacienda, []byte("x"), time.Minute))
		require.NoError(t, store.SetCredentials(secretToken, &models.HaciendaCredentials{Username: "u", Password: secretPassword}, time.Minute))
		_, _ = store.GetCredentials(secretToken)
		_, _ = store.GetCredentials("absent-" + secretToken)
		require.NoError(t, store.Delete(secretToken))

		assertNoSecrets(t, output.String(), secretToken, secretHacienda, secretPassword)
	})

	t.Run("encryption failure", func(t *testing.T) {
		output := captureLogs(t)
		server := miniredis.RunT(t)
		store, err := cache.NewRedisTokenCache(&config.RedisConfig{Host: server.Host(), Port: server.Port()}, fakeCrypt{fail: true})
		require.NoError(t, err)

		assert.Error(t, store.SetCredentials(secretToken, &models.HaciendaCredentials{Password: secretPassword}, time.Minute))
		assertNoSecrets(t, output.String(), secretToken, secretPassword)
	})

	t.Run("redis connection failures never print the password", func(t *testing.T) {
		output := captureLogs(t)

		_, parseErr := cache.NewRedisTokenCache(&config.RedisConfig{Host: "host", Port: "notaport", Password: secretRedisPass}, fakeCrypt{})
		assert.Error(t, parseErr)
		_, pingErr := cache.NewRedisTokenCache(&config.RedisConfig{Host: "127.0.0.1", Port: "1", Password: secretRedisPass}, fakeCrypt{})
		assert.Error(t, pingErr)

		assertNoSecrets(t, output.String(), secretRedisPass)
	})
}

// TestAdversarialNoSecretsInHaciendaAuthLogs verifies credentials and tokens are not logged during the Hacienda login.
func TestAdversarialNoSecretsInHaciendaAuthLogs(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		cacheSetOK bool
	}{
		{"success", http.StatusOK, `{"status":"OK","body":{"token":"` + secretHacienda + `","tokenType":"Bearer"}}`, true},
		{"cache set failure", http.StatusOK, `{"status":"OK","body":{"token":"` + secretHacienda + `","tokenType":"Bearer"}}`, false},
		{"rejected login", http.StatusUnauthorized, `{"status":"ERROR","body":{"token":"` + secretHacienda + `"}}`, true},
		{"malformed body", http.StatusOK, `not-json ` + secretHacienda, true},
		{"non OK status field", http.StatusOK, `{"status":"FAIL","body":{"token":"` + secretHacienda + `"}}`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := captureLogs(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			originalURL := config.MHPaths.AuthURL
			config.MHPaths.AuthURL = server.URL
			defer func() { config.MHPaths.AuthURL = originalURL }()

			ctrl := gomock.NewController(t)
			cacheManager := mocks.NewMockCacheManager(ctrl)
			authManager := mocks.NewMockAuthManager(ctrl)
			cacheManager.EXPECT().Get(gomock.Any()).Return("", errors.New("miss"))
			authManager.EXPECT().GetHaciendaCredentials(gomock.Any(), "NIT-1", secretToken).
				Return(&models.HaciendaCredentials{Username: "user", Password: secretPassword}, nil)
			if tc.status == http.StatusOK && tc.name != "malformed body" && tc.name != "non OK status field" {
				var setErr error
				if !tc.cacheSetOK {
					setErr = errors.New("redis down")
				}
				cacheManager.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Return(setErr)
			}

			ctx := context.WithValue(context.Background(), "claims", &models.AuthClaims{NIT: "NIT-1", BranchID: 1})
			_, _ = signing.NewHaciendaAuthService(cacheManager, authManager).GetOrCreateHaciendaToken(ctx, secretToken)

			assertNoSecrets(t, output.String(), secretPassword, secretHacienda, secretToken)
		})
	}

	t.Run("missing claims and credentials failure", func(t *testing.T) {
		output := captureLogs(t)
		ctrl := gomock.NewController(t)
		cacheManager := mocks.NewMockCacheManager(ctrl)
		authManager := mocks.NewMockAuthManager(ctrl)
		cacheManager.EXPECT().Get(gomock.Any()).Return("", errors.New("miss")).Times(2)
		authManager.EXPECT().GetHaciendaCredentials(gomock.Any(), "NIT-1", secretToken).Return(nil, errors.New("not found"))

		service := signing.NewHaciendaAuthService(cacheManager, authManager)
		_, _ = service.GetOrCreateHaciendaToken(context.Background(), secretToken)
		ctx := context.WithValue(context.Background(), "claims", &models.AuthClaims{NIT: "NIT-1"})
		_, _ = service.GetOrCreateHaciendaToken(ctx, secretToken)

		assertNoSecrets(t, output.String(), secretToken)
	})
}

// TestAdversarialNoSecretsInStandardAuthLogs verifies API secrets and credential structs are never logged.
func TestAdversarialNoSecretsInStandardAuthLogs(t *testing.T) {
	t.Run("validation failures", func(t *testing.T) {
		output := captureLogs(t)
		ctrl := gomock.NewController(t)
		strategy := strategies.NewStandardAuthStrategy(mocks.NewMockAuthRepositoryPort(ctrl), mocks.NewMockCacheManager(ctrl))

		assert.Error(t, strategy.ValidateCredentials(&models.AuthCredentials{APISecret: secretAPISecret, MHCredentials: &models.HaciendaCredentials{Username: "u", Password: secretPassword}}))
		assert.Error(t, strategy.ValidateCredentials(&models.AuthCredentials{APIKey: "key", MHCredentials: &models.HaciendaCredentials{Username: "u", Password: secretPassword}}))

		assertNoSecrets(t, output.String(), secretAPISecret, secretPassword)
	})

	t.Run("hacienda credentials lookup failure", func(t *testing.T) {
		output := captureLogs(t)
		ctrl := gomock.NewController(t)
		cacheManager := mocks.NewMockCacheManager(ctrl)
		cacheManager.EXPECT().GetCredentials(secretToken).Return(nil, errors.New("missing")).Times(2)
		strategy := strategies.NewStandardAuthStrategy(mocks.NewMockAuthRepositoryPort(ctrl), cacheManager)

		_, err := strategy.GetHaciendaCredentials(secretToken)
		assert.Error(t, err)
		_, _ = strategy.GetHaciendaCredentials(secretToken)

		assertNoSecrets(t, output.String(), secretToken)
	})
}

// TestAdversarialNoSecretsInJWTAndMiddlewareLogs verifies bearer tokens are absent from the token service and middleware logs.
func TestAdversarialNoSecretsInJWTAndMiddlewareLogs(t *testing.T) {
	output := captureLogs(t)
	server := miniredis.RunT(t)
	store, err := cache.NewRedisTokenCache(&config.RedisConfig{Host: server.Host(), Port: server.Port()}, fakeCrypt{})
	require.NoError(t, err)
	service := tokens.NewJWTService("jwt-signing-secret-value", store)

	signed, err := service.GenerateToken(&models.AuthClaims{ClientID: 1, BranchID: 1, NIT: "NIT-1"}, time.Hour)
	require.NoError(t, err)
	_, err = service.ValidateToken(signed)
	require.NoError(t, err)
	_, _ = service.ValidateToken(signed + "tampered")
	_, _ = service.ValidateToken(secretToken)
	require.NoError(t, service.RevokeToken(signed))
	_, _ = service.ValidateToken(signed)

	handler := middleware.NewAuthMiddleware(service).Handle(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, header := range []string{"Bearer " + signed, "Token " + secretToken, "Bearer " + secretToken + " extra", secretToken, "Bearer " + secretToken} {
		request := httptest.NewRequest(http.MethodGet, "/dte", nil)
		request.Header.Set("Authorization", header)
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}

	assertNoSecrets(t, output.String(), signed, secretToken, "jwt-signing-secret-value")
	assertNoSecrets(t, output.String(), signed[:40])
}

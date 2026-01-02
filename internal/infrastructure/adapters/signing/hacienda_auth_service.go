package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MarlonG1/api-facturacion-sv/config"
	ports2 "github.com/MarlonG1/api-facturacion-sv/internal/application/ports"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/auth"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MarlonG1/api-facturacion-sv/internal/domain/auth/models"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/ports"
	errPackage "github.com/MarlonG1/api-facturacion-sv/internal/infrastructure/error"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/logs"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/shared_error"
)

type HaciendaAuthService struct {
	client      *http.Client
	cache       ports.CacheManager
	authService auth.AuthManager
}

type haciendaAuthRequest struct {
	User string `json:"user"`
	Pwd  string `json:"pwd"`
}

type haciendaAuthResponse struct {
	Status string `json:"status"`
	Body   struct {
		Token     string `json:"token"`
		TokenType string `json:"tokenType"`
	} `json:"body"`
}

// NewHaciendaAuthService crea una instancia de HaciendaAuthService. Recibe un cache de tokens de Hacienda.
func NewHaciendaAuthService(cache ports.CacheManager, authService auth.AuthManager) ports2.HaciendaAuthManager {
	return &HaciendaAuthService{
		authService: authService,
		client:      &http.Client{},
		cache:       cache,
	}
}

// GetOrCreateHaciendaToken obtiene un token de Hacienda, primero verificando la caché.
func (s *HaciendaAuthService) GetOrCreateHaciendaToken(ctx context.Context, systemToken string) (string, error) {
	haciendaToken, err := s.getFromCache(systemToken)
	if err == nil {
		logs.Info("Token found in cache, skipping authentication")
		return haciendaToken, nil
	}

	logs.Info("Token not found in cache, starting process to get it")
	//Obtener del contexto los claims
	claims, ok := ctx.Value("claims").(*models.AuthClaims)
	if !ok {
		logs.Error("Claims not found in context")
		return "", shared_error.NewGeneralServiceError("AuthHacienda", "GetOrCreateHaciendaToken", "claims not found in context", nil)
	}
	logs.Info("Claims found in context", map[string]interface{}{"claims": claims})

	//Obtener las credenciales de hacienda
	haciendaCreds, err := s.authService.GetHaciendaCredentials(ctx, claims.NIT, systemToken)
	if err != nil {
		logs.Error("Error getting hacienda credentials", map[string]interface{}{"error": err.Error()})
		return "", err
	}
	logs.Info("Hacienda credentials retrieved", map[string]interface{}{"haciendaCreds": haciendaCreds})

	return s.createAndCacheToken(ctx, systemToken, *haciendaCreds)
}

func (s *HaciendaAuthService) GetOrCreateHaciendaTokenWithCreds(ctx context.Context, systemToken string, creds models.HaciendaCredentials) (string, error) {
	haciendaToken, err := s.getFromCache(systemToken)
	if err == nil {
		return haciendaToken, nil
	}

	return s.createAndCacheToken(ctx, systemToken, creds)
}

func (s *HaciendaAuthService) getFromCache(systemToken string) (string, error) {
	key := fmt.Sprintf("hacienda:token:%s", systemToken)
	haciendaToken, err := s.cache.Get(key)
	if err != nil {
		logs.Warn("Token not found in cache or expired", map[string]interface{}{
			"systemToken": systemToken,
			"error":       err.Error(),
		})
		return "", shared_error.NewGeneralServiceError("AuthHacienda", "GetOrCreateHaciendaToken", "token not found in cache", errPackage.ErrExpiredToken)
	}

	logs.Info("Token found in cache, returned successfully")
	return haciendaToken, nil
}

func (s *HaciendaAuthService) createAndCacheToken(ctx context.Context, systemToken string, creds models.HaciendaCredentials) (string, error) {
	token, err := s.authenticateWithHacienda(ctx, creds)
	key := fmt.Sprintf("hacienda:token:%s", systemToken)
	if err != nil {
		return "", err
	}

	if err := s.cache.Set(key, []byte(token), 24*time.Hour); err != nil {
		return "", s.cacheError(err, systemToken)
	}

	logs.Info("Token authenticated and stored in cache", map[string]interface{}{
		"key":   key,
		"token": token,
	})
	return token, nil
}

func (s *HaciendaAuthService) cacheError(err error, systemToken string) error {
	logs.Error("Error storing token in cache", map[string]interface{}{
		"systemToken": systemToken,
		"error":       err.Error(),
	})
	return shared_error.NewGeneralServiceError("AuthHacienda", "GetOrCreateHaciendaToken", "error caching Hacienda token", err)
}

func (s *HaciendaAuthService) authenticateWithHacienda(ctx context.Context, creds models.HaciendaCredentials) (string, error) {
	reqBody := haciendaAuthRequest{
		User: creds.Username,
		Pwd:  creds.Password,
	}

	logs.Info("Authenticating with Hacienda", map[string]interface{}{
		"username": creds.Username,
		"password": creds.Password,
	})

	req, err := s.createHTTPRequest(ctx, reqBody)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", s.httpRequestError(err)
	}
	defer resp.Body.Close()

	if err := s.verifyAuthResponse(resp); err != nil {
		return "", err
	}

	return s.decodeAuthResponse(resp.Body)
}

func (s *HaciendaAuthService) encodeError(err error) error {
	logs.Error("Error encoding authentication request", map[string]interface{}{"error": err.Error()})
	return shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", "failed to encode authentication request", err)
}

func (s *HaciendaAuthService) createHTTPRequest(ctx context.Context, creds haciendaAuthRequest) (*http.Request, error) {
	formData := url.Values{}
	formData.Set("user", creds.User)
	formData.Set("pwd", creds.Pwd)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		config.MHPaths.AuthURL,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		logs.Error("Error creating HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", "failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "HaciendaApp/1.0")

	logs.Info("HTTP request created", map[string]interface{}{})
	return req, nil
}

func (s *HaciendaAuthService) httpRequestError(err error) error {
	logs.Error("Error performing HTTP request", map[string]interface{}{"error": err.Error()})
	return shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", "HTTP request failed", err)
}

func (s *HaciendaAuthService) verifyAuthResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		logs.Error("Error authenticating with Hacienda", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   resp.Body,
		})
		return shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", fmt.Sprintf("Hacienda signing failed with status: %d", resp.StatusCode), nil)
	}
	return nil
}

func (s *HaciendaAuthService) decodeAuthResponse(body io.Reader) (string, error) {
	var authResp haciendaAuthResponse
	if err := json.NewDecoder(body).Decode(&authResp); err != nil {
		logs.Error("Error decoding authentication response", map[string]interface{}{"error": err.Error()})
		return "", shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", "failed to decode Hacienda signing response", err)
	}

	if authResp.Status != "OK" {
		logs.Error("Error authenticating with Hacienda", map[string]interface{}{"status": authResp.Status})
		return "", shared_error.NewGeneralServiceError("AuthHacienda", "authenticateWithHacienda", "Hacienda signing failed: "+authResp.Status, nil)
	}

	logs.Info("Authentication successful", map[string]interface{}{
		"tokenType": authResp.Body.TokenType,
		"status":    authResp.Status,
	})
	return fmt.Sprintf("%s", authResp.Body.Token), nil
}

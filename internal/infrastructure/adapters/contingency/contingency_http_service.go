package contingency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	haciendaPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
	authPorts "github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// contingencyHTTPService handles signing and transmission of contingency events to Hacienda.
type contingencyHTTPService struct {
	haciendaAuth haciendaPorts.HaciendaAuthManager
	signer       haciendaPorts.SignerManager
	cache        authPorts.CacheManager
	httpClient   *http.Client
}

// HaciendaContingencyRequest is the payload structure sent to Hacienda's contingency endpoint.
type HaciendaContingencyRequest struct {
	NIT      string `json:"nit"`
	Document string `json:"documento"`
}

func newContingencyHTTPService(
	haciendaAuth haciendaPorts.HaciendaAuthManager,
	signer haciendaPorts.SignerManager,
	cache authPorts.CacheManager,
) *contingencyHTTPService {
	return &contingencyHTTPService{
		haciendaAuth: haciendaAuth,
		signer:       signer,
		cache:        cache,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
			},
		},
	}
}

// SignAndSend signs the contingency event and transmits it to Hacienda.
// It returns isDuplicate=true when Hacienda reports the event already exists, which
// allows the caller to handle that case with a domain-specific error type.
func (s *contingencyHTTPService) SignAndSend(
	ctx context.Context,
	event *models.ContingencyEvent,
	nit string,
	token string,
) (isDuplicate bool, err error) {
	haciendaToken, err := s.getHaciendaToken(ctx, token)
	if err != nil {
		return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "SignAndSend", "failed to get hacienda token", err)
	}

	signedDoc, err := s.signEvent(ctx, event, nit)
	if err != nil {
		return false, err
	}

	resp, err := s.sendRequest(ctx, nit, signedDoc, haciendaToken)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return s.handleResponse(resp)
}

func (s *contingencyHTTPService) getHaciendaToken(ctx context.Context, token string) (string, error) {
	encryptedCreds, err := s.cache.GetCredentials(token)
	if err != nil {
		return "", shared_error.NewGeneralServiceError("contingencyHTTPService", "getHaciendaToken", "failed to get credentials from cache", err)
	}

	return s.haciendaAuth.GetOrCreateHaciendaTokenWithCreds(ctx, token, *encryptedCreds)
}

func (s *contingencyHTTPService) signEvent(ctx context.Context, event *models.ContingencyEvent, nit string) (string, error) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return "", shared_error.NewGeneralServiceError("contingencyHTTPService", "signEvent", "failed to marshal contingency event", err)
	}

	signedDoc, err := s.signer.SignDTE(ctx, jsonData, nit)
	if err != nil {
		return "", shared_error.NewGeneralServiceError("contingencyHTTPService", "signEvent", "failed to sign contingency event", err)
	}

	return signedDoc, nil
}

func (s *contingencyHTTPService) sendRequest(ctx context.Context, nit, signedDoc, haciendaToken string) (*http.Response, error) {
	reqBody := &HaciendaContingencyRequest{
		NIT:      nit,
		Document: signedDoc,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("contingencyHTTPService", "sendRequest", "failed to marshal contingency request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.MHPaths.ContingencyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("contingencyHTTPService", "sendRequest", "failed to create request", err)
	}

	req.Header.Set("Authorization", haciendaToken)
	req.Header.Set("Content-Type", "application/json")

	logs.Info("Sending contingency event request", map[string]interface{}{
		"url":          config.MHPaths.ContingencyURL,
		"method":       "POST",
		"content-type": req.Header.Get("Content-Type"),
	})

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("contingencyHTTPService", "sendRequest", "failed to send request", err)
	}

	return resp, nil
}

func (s *contingencyHTTPService) handleResponse(resp *http.Response) (isDuplicate bool, err error) {
	var responseBody map[string]interface{}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&responseBody); decodeErr != nil {
		if decodeErr == io.EOF {
			logs.Warn("Contingency event response is empty")
		} else {
			return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleResponse", "failed to decode response body", decodeErr)
		}
	}

	logs.Info("Contingency event response", map[string]interface{}{
		"statusCode": resp.StatusCode,
		"body":       responseBody,
	})

	if resp.StatusCode != http.StatusOK {
		return s.handleNonOKResponse(responseBody)
	}

	return s.handleOKResponseBody(responseBody)
}

func (s *contingencyHTTPService) handleNonOKResponse(responseBody map[string]interface{}) (bool, error) {
	if responseBody == nil {
		return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleNonOKResponse", "unexpected non-200 status with empty body", nil)
	}

	mensaje, ok := responseBody["mensaje"].(string)
	if !ok {
		return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleNonOKResponse", "unexpected non-200 status", nil)
	}

	mensajeLower := strings.ToLower(mensaje)
	if containsDuplicateKeywords(mensajeLower) {
		logs.Warn("Contingency event already exists in Hacienda", map[string]interface{}{
			"message":      mensaje,
			"observations": responseBody["observaciones"],
		})
		return true, nil
	}

	if strings.Contains(mensajeLower, "no superadas") {
		logs.Error("Contingency event validation failed", map[string]interface{}{
			"message":      mensaje,
			"observations": responseBody["observaciones"],
		})
		return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleNonOKResponse", fmt.Sprintf("validation failed: %s", mensaje), nil)
	}

	return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleNonOKResponse", fmt.Sprintf("unexpected status: %s", mensaje), nil)
}

func (s *contingencyHTTPService) handleOKResponseBody(responseBody map[string]interface{}) (bool, error) {
	if responseBody == nil {
		return false, nil
	}

	estado, estadoOk := responseBody["estado"].(string)
	if !estadoOk || estado != "RECHAZADO" {
		return false, nil
	}

	mensaje, _ := responseBody["mensaje"].(string)
	observaciones := responseBody["observaciones"]

	if isDuplicate := detectDuplicateInObservations(observaciones) || containsDuplicateKeywords(strings.ToLower(mensaje)); isDuplicate {
		logs.Warn("Contingency event already exists in Hacienda (RECHAZADO body)", map[string]interface{}{
			"message":      mensaje,
			"observations": observaciones,
		})
		return true, nil
	}

	logs.Error("Contingency event rejected by Hacienda", map[string]interface{}{
		"message":      mensaje,
		"observations": observaciones,
	})
	return false, shared_error.NewGeneralServiceError("contingencyHTTPService", "handleOKResponseBody", fmt.Sprintf("event rejected: %s", mensaje), nil)
}

func containsDuplicateKeywords(text string) bool {
	return strings.Contains(text, "ya existe evento") || strings.Contains(text, "ya existe envento")
}

func detectDuplicateInObservations(observaciones interface{}) bool {
	obsArray, ok := observaciones.([]interface{})
	if !ok || len(obsArray) == 0 {
		return false
	}
	return containsDuplicateKeywords(strings.ToLower(fmt.Sprintf("%v", obsArray[0])))
}

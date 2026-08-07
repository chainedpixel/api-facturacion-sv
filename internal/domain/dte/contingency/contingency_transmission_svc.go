package contingency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/chainedpixel/ordo-factus/config"
	appPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	batch "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// contingencyTransmissionSvc handles token generation, document signing, and batch transmission to Hacienda.
type contingencyTransmissionSvc struct {
	authManager      auth.AuthManager
	haciendaAuth     appPorts.HaciendaAuthManager
	cache            ports.CacheManager
	tokenService     ports.TokenManager
	signer           appPorts.SignerManager
	batchTransmitter batch.BatchTransmitterPort
	config           *transmitterModels.TransmissionConfig
}

func newContingencyTransmissionSvc(
	authManager auth.AuthManager,
	haciendaAuth appPorts.HaciendaAuthManager,
	cache ports.CacheManager,
	tokenService ports.TokenManager,
	signer appPorts.SignerManager,
	batchTransmitter batch.BatchTransmitterPort,
	config *transmitterModels.TransmissionConfig,
) *contingencyTransmissionSvc {
	return &contingencyTransmissionSvc{
		authManager:      authManager,
		haciendaAuth:     haciendaAuth,
		cache:            cache,
		tokenService:     tokenService,
		signer:           signer,
		batchTransmitter: batchTransmitter,
		config:           config,
	}
}

// getTokenAndCreds generates a matching JWT for the branch and retrieves its Hacienda credentials.
func (s *contingencyTransmissionSvc) getTokenAndCreds(ctx context.Context, branchID uint) (string, *authModels.HaciendaCredentials, error) {
	client, err := s.authManager.GetBranchByBranchID(ctx, branchID)
	if err != nil {
		return "", nil, shared_error.NewGeneralServiceError("ContingencyService", "processSystemDocumentsByType", "failed to get branch by ID", err)
	}

	token, err := s.generateMatchingToken(client)
	if err != nil {
		return "", nil, shared_error.NewGeneralServiceError("ContingencyService", "processSystemDocumentsByType", "failed to generate matching token", err)
	}

	encryptedCreds, err := s.cache.GetCredentials(token)
	if err != nil {
		return "", nil, shared_error.NewGeneralServiceError("ContingencyService", "processSystemDocumentsByType", "failed to get credentials", err)
	}

	return token, encryptedCreds, nil
}

// processBatch signs a slice of documents and transmits them to Hacienda as a batch, then verifies.
func (s *contingencyTransmissionSvc) processBatch(
	ctx context.Context,
	systemNIT string,
	dteType string,
	docs []dte.ContingencyDocument,
	token string,
	creds authModels.HaciendaCredentials,
	branchID uint,
) error {
	signedDocs, docsMap := s.signDocuments(ctx, docs, systemNIT)
	if len(signedDocs) == 0 {
		logs.Warn("No documents signed")
		return fmt.Errorf("no documents could be signed for NIT %s, dteType %s", systemNIT, dteType)
	}

	response, haciendaToken, err := s.batchTransmitter.TransmitBatch(ctx, systemNIT, dteType, signedDocs, token, creds)
	if err != nil {
		logs.Error("Failed to transmit batch", map[string]interface{}{
			"error":    err.Error(),
			"dteType":  dteType,
			"docCount": len(signedDocs),
		})
		return fmt.Errorf("batch transmission failed for NIT %s, dteType %s: %w", systemNIT, dteType, err)
	}

	if err = s.batchTransmitter.VerifyContingencyBatchStatus(ctx, response.SendID, response.BatchCode, haciendaToken, branchID, docsMap); err != nil {
		logs.Error("Failed to verify batch status", map[string]interface{}{
			"error":   err.Error(),
			"dteType": dteType,
		})
		return fmt.Errorf("batch verification failed for NIT %s, dteType %s: %w", systemNIT, dteType, err)
	}

	return nil
}

// checkDocumentStatus queries Hacienda for the current processing status of a previously submitted document.
func (s *contingencyTransmissionSvc) checkDocumentStatus(ctx context.Context, generationCode, nit, dteType string) (*transmitterModels.TransmitResult, error) {
	userClient, err := s.authManager.GetByNIT(ctx, nit)
	if err != nil {
		return nil, fmt.Errorf("failed to get client by NIT: %w", err)
	}

	if len(userClient.BranchOffices) == 0 {
		return nil, fmt.Errorf("user has no branch offices")
	}

	token, err := s.generateMatchingToken(&userClient.BranchOffices[0])
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	haciendaToken, err := s.haciendaAuth.GetOrCreateHaciendaToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get Hacienda token: %w", err)
	}

	return s.queryHaciendaStatus(ctx, generationCode, nit, dteType, haciendaToken)
}

func (s *contingencyTransmissionSvc) signDocuments(ctx context.Context, docs []dte.ContingencyDocument, nit string) ([]string, map[string]dte.ContingencyDocument) {
	signedDocs := make([]string, 0, len(docs))
	docsMap := make(map[string]dte.ContingencyDocument, len(docs))

	for _, doc := range docs {
		signedDoc, err := s.signer.SignDTE(ctx, []byte(doc.Document.JSONData), nit)
		if err != nil {
			logs.Error("Failed to sign document", map[string]interface{}{
				"error": err.Error(),
				"nit":   nit,
				"id":    doc.ID,
			})
			continue
		}
		docsMap[doc.Document.ID] = doc
		signedDocs = append(signedDocs, signedDoc)
	}

	return signedDocs, docsMap
}

func (s *contingencyTransmissionSvc) generateMatchingToken(client *user.BranchOffice) (string, error) {
	key := fmt.Sprintf("token:timestamps:%d", client.User.ID)

	jsonTimestamps, err := s.cache.Get(key)
	if err != nil {
		return "", err
	}

	var timestamps struct {
		IssuedAt  int64 `json:"IssuedAt"`
		ExpiresAt int64 `json:"ExpiresAt"`
	}
	if err = json.Unmarshal([]byte(jsonTimestamps), &timestamps); err != nil {
		return "", err
	}

	claims := &authModels.AuthClaims{
		ClientID: client.User.ID,
		BranchID: client.ID,
		AuthType: client.User.AuthType,
		NIT:      client.User.NIT,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        claims.ClientID,
		"branch_sub": claims.BranchID,
		"auth_type":  claims.AuthType,
		"nit":        claims.NIT,
		"exp":        timestamps.ExpiresAt,
		"iat":        timestamps.IssuedAt,
	})

	return token.SignedString([]byte(s.tokenService.GetSecretKey()))
}

func (s *contingencyTransmissionSvc) queryHaciendaStatus(ctx context.Context, generationCode, nit, dteType, haciendaToken string) (*transmitterModels.TransmitResult, error) {
	consultReq := map[string]interface{}{
		"nitEmisor":        nit,
		"tdte":             dteType,
		"codigoGeneracion": generationCode,
	}

	jsonData, err := json.Marshal(consultReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal consult request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.MHPaths.ReceptionConsultURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", haciendaToken)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var haciendaResp struct {
		Status         string   `json:"estado"`
		ReceptionStamp *string  `json:"selloRecibido"`
		ProcessingDate string   `json:"fhProcesamiento"`
		MessageCode    string   `json:"codigoMsg"`
		MessageDesc    string   `json:"descripcionMsg"`
		Observations   []string `json:"observaciones,omitempty"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&haciendaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &transmitterModels.TransmitResult{
		Status:         haciendaResp.Status,
		ReceptionStamp: haciendaResp.ReceptionStamp,
		ProcessingDate: haciendaResp.ProcessingDate,
		MessageCode:    haciendaResp.MessageCode,
		MessageDesc:    haciendaResp.MessageDesc,
		Observations:   haciendaResp.Observations,
	}, nil
}

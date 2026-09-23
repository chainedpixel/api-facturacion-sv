package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	batchPorts "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter"
	ports2 "github.com/chainedpixel/ordo-factus/internal/domain/ports"

	authPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/circuit"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/google/uuid"
)

// BatchTransmitterService implements the batch transmission logic to Hacienda
type BatchTransmitterService struct {
	haciendaAuth      authPorts.HaciendaAuthManager
	signer            authPorts.SignerManager
	contingencyRepo   contingency.ContingencyRepositoryPort
	sequentialManager dte_documents.SequentialNumberManager
	timeProvider      ports2.TimeProvider
	config            *models.TransmissionConfig
	httpClient        *http.Client
	circuitBreaker    *circuit.CircuitBreaker
	connection        *drivers.DbConnection
	bus               event.Bus
}

func (s *BatchTransmitterService) SetEventBus(bus event.Bus) {
	s.bus = bus
}

// NewBatchTransmitterService constructor for BatchTransmitterService
func NewBatchTransmitterService(
	haciendaAuth authPorts.HaciendaAuthManager,
	signer authPorts.SignerManager,
	contingencyRepo contingency.ContingencyRepositoryPort,
	sequentialManager dte_documents.SequentialNumberManager,
	config *models.TransmissionConfig,
	timeProvider ports2.TimeProvider,
	connection *drivers.DbConnection,
) batchPorts.BatchTransmitterPort {
	return &BatchTransmitterService{
		haciendaAuth:      haciendaAuth,
		signer:            signer,
		contingencyRepo:   contingencyRepo,
		sequentialManager: sequentialManager,
		config:            config,
		timeProvider:      timeProvider,
		connection:        connection,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
			},
		},
		circuitBreaker: circuit.NewCircuitBreaker(
			3,
			5*time.Minute,
		),
	}
}

// GetDTEVersion determines the version based on the DTE type
func (s *BatchTransmitterService) GetDTEVersion(dteType string) int {
	switch dteType {
	case constants.FacturaElectronica, constants.ComprobanteRetencionElectronico, constants.FacturaSujetoExcluidoElectronica:
		return 2
	case constants.FacturaExportacionElectronica:
		return 3
	case constants.CCFElectronico, constants.NotaRemisionElectronica, constants.NotaCreditoElectronica, constants.NotaDebitoElectronica:
		return 4
	default:
		return 2
	}
}

// TransmitBatch transmits a batch of documents to Hacienda
func (s *BatchTransmitterService) TransmitBatch(
	ctx context.Context,
	systemNIT string,
	dteType string,
	signedDocs []string,
	token string,
	creds authModels.HaciendaCredentials,
) (*models.BatchResponse, string, error) {
	if len(signedDocs) == 0 {
		return nil, "", errors.New("no documents to transmit")
	}

	batchID := strings.ToUpper(uuid.New().String())
	batch := &models.BatchRequest{
		Ambient:   s.config.GetAmbient(),
		SendID:    batchID,
		Version:   s.GetDTEVersion(dteType),
		NIT:       systemNIT,
		Documents: signedDocs,
	}

	haciendaToken, err := s.getHaciendaTokenWithRetry(ctx, token, creds)
	if err != nil {
		return nil, "", shared_error.NewGeneralServiceError("BatchTransmitterService", "TransmitBatch", "failed to get hacienda token", err)
	}

	response, err := s.sendBatchWithRetry(ctx, batch, haciendaToken)
	if err != nil {
		logs.Error("Failed to send batch", map[string]interface{}{
			"error":   err.Error(),
			"batchId": batchID,
		})
		return nil, "", err
	}

	logs.Info("Batch sent successfully", map[string]interface{}{
		"batchId": response.BatchCode,
		"status":  response.Status,
		"msg":     response.Description,
		"idEnvio": response.SendID,
	})

	return response, haciendaToken, nil
}

// getHaciendaTokenWithRetry retrieves a Hacienda authentication token with retries
func (s *BatchTransmitterService) getHaciendaTokenWithRetry(
	ctx context.Context,
	token string,
	creds authModels.HaciendaCredentials,
) (string, error) {
	var haciendaToken string
	var err error
	retryPolicy := s.config.GetRetryPolicy()

	for attempt := 0; attempt < retryPolicy.MaxAttempts; attempt++ {
		haciendaToken, err = s.haciendaAuth.GetOrCreateHaciendaTokenWithCreds(ctx, token, creds)
		if err == nil {
			return haciendaToken, nil
		}

		if !s.shouldRetry(err) {
			return "", err
		}

		s.sleep(attempt)
	}

	return "", shared_error.NewGeneralServiceError("BatchTransmitterService", "getHaciendaTokenWithRetry", "max retry attempts reached", err)
}

// sendBatchWithRetry sends a batch with retries
func (s *BatchTransmitterService) sendBatchWithRetry(
	ctx context.Context,
	batch *models.BatchRequest,
	token string,
) (*models.BatchResponse, error) {
	var response *models.BatchResponse
	var err error
	retryPolicy := s.config.GetRetryPolicy()

	for attempt := 0; attempt < retryPolicy.MaxAttempts; attempt++ {
		response, err = s.transmitToHacienda(ctx, batch, token)
		if err == nil {
			return response, nil
		}

		if !s.shouldRetry(err) {
			return nil, err
		}

		s.sleep(attempt)
	}

	return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "sendBatchWithRetry", "max retry attempts reached", err)
}

// transmitToHacienda sends the batch to Hacienda
func (s *BatchTransmitterService) transmitToHacienda(
	ctx context.Context,
	batch *models.BatchRequest,
	token string,
) (*models.BatchResponse, error) {
	if !s.circuitBreaker.AllowRequest() {
		logs.Warn("Circuit breaker preventing request to Hacienda", map[string]interface{}{
			"state": s.circuitBreaker.GetState(),
		})
		return nil, shared_error.NewGeneralServiceError(
			"BatchTransmitterService",
			"transmitToHacienda",
			"service temporarily unavailable due to consecutive failures",
			nil,
		)
	}

	reqBody, err := json.Marshal(batch)
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "transmitToHacienda", "failed to marshal request", err)
	}

	logs.Info("Sending batch to Hacienda", map[string]interface{}{
		"batchId": batch.SendID,
		"ambient": batch.Ambient,
		"docs":    len(batch.Documents),
		"body":    string(reqBody),
	})

	req, err := http.NewRequestWithContext(ctx, "POST", config.MHPaths.LoteReceptionURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "transmitToHacienda", "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		logs.Error("Request to Hacienda failed", map[string]interface{}{
			"error":        err.Error(),
			"failureCount": s.circuitBreaker.GetFailureCount(),
		})
		return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "transmitToHacienda", "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.circuitBreaker.RecordFailure()
		return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "transmitToHacienda", "unexpected status code", nil)
	}

	var batchResp models.BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, shared_error.NewGeneralServiceError("BatchTransmitterService", "transmitToHacienda", "failed to decode response", err)
	}

	s.circuitBreaker.RecordSuccess()
	return &batchResp, nil
}

// VerifyContingencyBatchStatus verifies the status of a batch
func (s *BatchTransmitterService) VerifyContingencyBatchStatus(
	ctx context.Context,
	batchID string,
	mhBatchID string,
	token string,
	branchID uint,
	docsMap map[string]dte.ContingencyDocument,
) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	deadline := utils.TimeNow().Add(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			status, isProcessed, err := s.checkBatchStatus(ctx, mhBatchID, token)
			if err != nil {
				logs.Error("Failed to check batch status, verify", map[string]interface{}{
					"error":   err.Error(),
					"batchID": mhBatchID,
				})
				return shared_error.NewGeneralServiceError("BatchTransmitterService", "VerifyContingencyBatchStatus", "failed to check batch status", err)
			}

			if !isProcessed {
				logs.Info("Batch still processing", map[string]interface{}{
					"batchID": mhBatchID,
				})

				if utils.TimeNow().After(deadline) {
					return shared_error.NewGeneralServiceError("BatchTransmitterService", "VerifyContingencyBatchStatus", "batch processing timeout", nil)
				}
				continue
			}

			sqlDb, err := s.connection.Db.DB()
			if err != nil {
				return shared_error.NewGeneralServiceError("ContingencyEventService", "PrepareAndSendContingencyEvent", "failed to get sql db", err)
			}
			sqlDb.Ping()

			if len(status.Processed) > 0 {
				var processedContingencyIDs []string
				var processedDocumentIDs []string
				var processedStamps map[string]string
				var proccesedObservations []string
				processedStamps = make(map[string]string)
				for _, processed := range status.Processed {
					if doc, exists := docsMap[processed.GenerationCode]; exists {
						logs.Info("Document processed", map[string]interface{}{
							"code":            processed.MessageCode,
							"message":         processed.DescriptionMessage,
							"observations":    processed.Observations,
							"processedAt":     processed.ProcessingDate,
							"reception_stamp": processed.ReceptionStamp,
							"contingencyID":   doc.ID,
							"documentID":      doc.DocumentID,
						})
						processedStamps[doc.ID] = processed.ReceptionStamp
						proccesedObservations = append(proccesedObservations, processed.DescriptionMessage)
						processedContingencyIDs = append(processedContingencyIDs, doc.ID)
						processedDocumentIDs = append(processedDocumentIDs, doc.DocumentID)
					}
				}

				if err := s.contingencyRepo.UpdateBatch(ctx, processedContingencyIDs, proccesedObservations, processedStamps, batchID, mhBatchID, constants.DocumentReceived); err != nil {
					logs.Error("Failed to update processed documents", map[string]interface{}{
						"error":   err.Error(),
						"batchID": batchID,
					})
					return shared_error.NewGeneralServiceError("BatchTransmitterService", "VerifyContingencyBatchStatus", "failed to update processed documents", err)
				}

				for _, docID := range processedDocumentIDs {
					if confirmErr := s.sequentialManager.ConfirmReservationByDocumentID(ctx, docID); confirmErr != nil {
						logs.Warn("Failed to confirm reservation for processed document", map[string]interface{}{
							"error":      confirmErr.Error(),
							"documentID": docID,
						})
					} else {
						logs.Info("Reservation confirmed for processed contingency document", map[string]interface{}{
							"documentID": docID,
						})
					}
				}

				logs.Info("Processed documents updated", map[string]interface{}{
					"batchID": batchID,
				})
			}

			if len(status.Rejected) > 0 {
				type rejectedEntry struct {
					contingencyID string
					documentID    string
					controlNumber string
					code          string
					description   string
					observations  []string
				}
				var entries []rejectedEntry

				for _, rejected := range status.Rejected {
					if doc, exists := docsMap[rejected.GenerationCode]; exists {
						code := rejected.ClassifyMessage
						if code == "" && rejected.MessageCode != "" {
							code = rejected.MessageCode
						}
						controlNum := ""
						if doc.Document != nil {
							controlNum = doc.Document.ControlNumber
						}
						logs.Info("Document rejected", map[string]interface{}{
							"classifyMsg":   rejected.ClassifyMessage,
							"messageCode":   rejected.MessageCode,
							"codeUsed":      code,
							"message":       rejected.DescriptionMessage,
							"observations":  rejected.Observations,
							"processedAt":   rejected.ProcessingDate,
							"contingencyID": doc.ID,
							"documentID":    doc.DocumentID,
						})
						entries = append(entries, rejectedEntry{
							contingencyID: doc.ID,
							documentID:    doc.DocumentID,
							controlNumber: controlNum,
							code:          code,
							description:   rejected.DescriptionMessage,
							observations:  rejected.Observations,
						})
					}
				}

				contingencyIDs := make([]string, len(entries))
				descriptions := make([]string, len(entries))
				for i, e := range entries {
					contingencyIDs[i] = e.contingencyID
					descriptions[i] = e.description
				}

				if err := s.contingencyRepo.UpdateBatch(ctx, contingencyIDs, descriptions, nil, batchID, mhBatchID, constants.DocumentRejected); err != nil {
					logs.Error("Failed to update rejected documents", map[string]interface{}{
						"error":   err.Error(),
						"batchID": batchID,
					})
				}

				if s.bus != nil {
					var failedDocs []event.RejectedDocSummary
					for _, e := range entries {
						s.bus.Publish(ctx, event.EmissionFailureEvent{
							ControlNumber:        e.controlNumber,
							GenerationCode:       e.documentID,
							ErrorCode:            e.code,
							LastError:            e.description,
							HaciendaObservations: e.observations,
							Attempts:             1,
							OccurredAtTime:       time.Now(),
						})
						failedDocs = append(failedDocs, event.RejectedDocSummary{
							ControlNumber: e.controlNumber,
							ErrorCode:     e.code,
							Description:   e.description,
							Observations:  e.observations,
						})
					}
					if len(failedDocs) > 0 {
						s.bus.Publish(ctx, event.RetransmissionJobFailedEvent{
							JobName:         "contingency_retransmission",
							FailedDocuments: failedDocs,
							OccurredAtTime:  time.Now(),
						})
					}
				}

				documentIDs := make([]string, len(entries))
				for i, e := range entries {
					documentIDs[i] = e.documentID
				}

				for _, e := range entries {
					if releaseErr := s.sequentialManager.ReleaseReservationByDocumentID(ctx, e.documentID, e.description, e.code); releaseErr != nil {
						logs.Warn("Failed to release reservation for rejected document", map[string]interface{}{
							"error":      releaseErr.Error(),
							"documentID": e.documentID,
						})
					}
				}

				logs.Info("Rejected documents updated", map[string]interface{}{
					"batchID": batchID,
				})
			}

			logs.Info("Batch status verified", map[string]interface{}{
				"batchID":        batchID,
				"totalProcessed": len(status.Processed),
				"totalRejected":  len(status.Rejected),
			})

			return nil
		}
	}
}

// checkBatchStatus verifies the status of a batch in Hacienda
func (s *BatchTransmitterService) checkBatchStatus(ctx context.Context, batchID string, haciendaToken string) (*models.ConsultBatchResponse, bool, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/%s", config.MHPaths.LoteReceptionConsultURL, batchID),
		nil,
	)
	if err != nil {
		logs.Error("Failed to create batch status request", map[string]interface{}{
			"error":   err.Error(),
			"batchID": batchID,
		})
		return nil, false, err
	}

	req.Header.Set("Authorization", haciendaToken)
	req.Header.Set("Content-Type", "application/json")

	logs.Info("Checking batch status", map[string]interface{}{
		"url":     req.URL.String(),
		"method":  req.Method,
		"batchID": batchID,
	})

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logs.Error("Failed to check batch status inner", map[string]interface{}{
			"error":   err.Error(),
			"batchID": batchID,
		})
		return nil, false, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Error("Failed to read batch response body", map[string]interface{}{
			"error":   err.Error(),
			"batchID": batchID,
		})
		return nil, false, err
	}

	if len(bodyBytes) == 0 {
		return nil, false, nil
	}

	logs.Debug("Raw batch status response from Hacienda", map[string]interface{}{
		"batchID":  batchID,
		"response": string(bodyBytes),
	})

	var batchResp models.ConsultBatchResponse
	if err := json.Unmarshal(bodyBytes, &batchResp); err != nil {
		logs.Error("Failed to decode batch response", map[string]interface{}{
			"error":   err.Error(),
			"batchID": batchID,
			"rawBody": string(bodyBytes),
		})
		return nil, false, err
	}

	logs.Info("Batch status response", map[string]interface{}{
		"Processed": len(batchResp.Processed),
		"Rejected":  len(batchResp.Rejected),
	})
	return &batchResp, true, nil
}

// shouldRetry determines whether an operation should be retried
func (s *BatchTransmitterService) shouldRetry(err error) bool {
	if s.circuitBreaker.GetState() == constants.StateOpen {
		return false
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		logs.Info("Network error detected, will retry", map[string]interface{}{
			"error": err.Error(),
		})
		return true
	}

	var httpErr *hacienda_error.HTTPResponseError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode >= 500 && httpErr.StatusCode <= 599 {
			logs.Info("Server error detected, will retry", map[string]interface{}{
				"statusCode": httpErr.StatusCode,
			})
			return true
		}

		switch httpErr.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusRequestTimeout,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			logs.Info("Retryable HTTP error detected", map[string]interface{}{
				"statusCode": httpErr.StatusCode,
			})
			return true
		}

		logs.Info("Non-retryable HTTP error", map[string]interface{}{
			"statusCode": httpErr.StatusCode,
		})
		return false
	}

	var haciendaErr *hacienda_error.HaciendaResponseError
	if errors.As(err, &haciendaErr) {
		if strings.Contains(strings.ToLower(haciendaErr.Description), "validaci") ||
			strings.Contains(strings.ToLower(haciendaErr.Description), "autoriza") {
			logs.Info("Non-retryable Hacienda error", map[string]interface{}{
				"code":    haciendaErr.Code,
				"message": haciendaErr.Description,
			})
			return false
		}

		logs.Info("Retryable Hacienda error", map[string]interface{}{
			"code":    haciendaErr.Code,
			"message": haciendaErr.Description,
		})
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		logs.Info("Context error detected, will retry", map[string]interface{}{
			"error": err.Error(),
		})
		return true
	}

	logs.Warn("Unclassified error, defaulting to retry", map[string]interface{}{
		"error": err.Error(),
	})
	return true
}

// sleep implements exponential backoff for retries
func (s *BatchTransmitterService) sleep(attempt int) {
	retryPolicy := s.config.GetRetryPolicy()
	backoff := retryPolicy.InitialInterval * time.Duration(float64(attempt)*retryPolicy.BackoffFactor)
	if backoff > retryPolicy.MaxInterval {
		backoff = retryPolicy.MaxInterval
	}
	s.timeProvider.Sleep(backoff)
}

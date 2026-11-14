package contingency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainedpixel/api-facturacion-sv/config"
	appPorts "github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth"
	authModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/event"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	batch "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/ports"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

// ContingencyService orchestrates contingency document storage and retransmission.
type ContingencyService struct {
	contingencyEvents ContingencyEventSender
	timeProvider      ports.TimeProvider
	docSvc            *contingencyDocumentSvc
	txSvc             *contingencyTransmissionSvc
	bus               event.Bus
}

// NewContingencyManager creates a new ContingencyService with all required dependencies.
func NewContingencyManager(
	authManager auth.AuthManager,
	dteManager dte_documents.DTEManager,
	repo ContingencyRepositoryPort,
	haciendaAuth appPorts.HaciendaAuthManager,
	cache ports.CacheManager,
	tokenService ports.TokenManager,
	signer appPorts.SignerManager,
	batchTransmitter batch.BatchTransmitterPort,
	contingencyEvents ContingencyEventSender,
	sequentialManager dte_documents.SequentialNumberManager,
	timeProvider ports.TimeProvider,
	cfg *transmitterModels.TransmissionConfig,
) ContingencyManager {
	return &ContingencyService{
		contingencyEvents: contingencyEvents,
		timeProvider:      timeProvider,
		docSvc:            newContingencyDocumentSvc(dteManager, repo, sequentialManager),
		txSvc:             newContingencyTransmissionSvc(authManager, haciendaAuth, cache, tokenService, signer, batchTransmitter, dteManager, cfg),
	}
}

func (s *ContingencyService) SetEventBus(bus event.Bus) {
	s.bus = bus
}

// StoreDocumentInContingency builds the deferred document (deferred model, contingency transmission type, contingency
// type and reason), signs it once, persists the structure with its JWS and registers it for retransmission.
func (s *ContingencyService) StoreDocumentInContingency(ctx context.Context, document interface{}, dteType string, contingencyType int8, reason string) (*DeferredDocument, error) {
	claims := ctx.Value("claims").(*authModels.AuthClaims)

	contingencyDoc := &dte.ContingencyDocument{
		BranchID:        claims.BranchID,
		ContingencyType: contingencyType,
		Reason:          reason,
	}

	deferredDocument, err := utils.UpdateContingencyIdentification(document, &contingencyType, &reason)
	if err != nil {
		return nil, shared_error.NewGeneralServiceError("ContingencyService", "StoreDocumentInContingency", "failed to set contingency identification", err)
	}

	signedDocument, err := s.txSvc.signDeferredDocument(ctx, deferredDocument, claims.NIT)
	if err != nil {
		return nil, err
	}

	if err := s.docSvc.store(ctx, deferredDocument, contingencyDoc, signedDocument); err != nil {
		return nil, err
	}

	logs.Info("Document stored in contingency", map[string]interface{}{
		"id":              contingencyDoc.ID,
		"type":            dteType,
		"contingencyType": contingencyType,
	})

	if s.bus != nil {
		s.bus.Publish(ctx, event.ContingencyActivatedEvent{
			BranchID:        claims.BranchID,
			NIT:             claims.NIT,
			ContingencyType: fmt.Sprintf("%d", contingencyType),
			Reason:          reason,
			AffectedDocs:    1,
			OccurredAtTime:  time.Now(),
		})
	}

	return &DeferredDocument{Document: deferredDocument, SignedDocument: signedDocument}, nil
}

// RetransmitPendingDocuments fetches all pending contingency documents, sends one contingency event per issuer, branch
// and contingency type, so every DTE carries the same contingency type as its event, and retransmits each batch by
// DTE type. Documents that Hacienda already reports as processed or rejected are not transmitted again.
func (s *ContingencyService) RetransmitPendingDocuments(ctx context.Context) error {
	pendingDocs, err := s.docSvc.getPending(ctx, config.Server.MaxBatchSize)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyService", "RetransmitPendingDocuments", "failed to get pending documents", err)
	}

	pendingDocs = s.expireOverdueDocuments(ctx, pendingDocs)
	if len(pendingDocs) == 0 {
		logs.Info("No pending documents found")
		return nil
	}

	s.alertLongContingency(pendingDocs)

	for _, group := range groupByEvent(pendingDocs) {
		systemNIT := group.systemNIT
		docsForNIT := group.documents
		typeGroups := groupByDTEType(group.documents)

		if err := s.contingencyEvents.PrepareAndSendContingencyEvent(ctx, docsForNIT); err != nil {
			if contingencyErr, ok := err.(*ContingencyEventExistsError); ok {
				logs.Warn("Contingency event already exists, checking document status in Hacienda", map[string]interface{}{
					"error":     err.Error(),
					"systemNIT": systemNIT,
					"docCount":  len(contingencyErr.Documents),
				})
				settled := s.verifyAndUpdateExistingDocuments(ctx, contingencyErr.Documents)
				typeGroups = groupByDTEType(withoutSettled(group.documents, settled))
				logs.Info("Continuing with document transmission after checking existing events")
			} else {
				logs.Error("Failed to send contingency event", map[string]interface{}{
					"error":     err.Error(),
					"systemNIT": systemNIT,
				})
				continue
			}
		}

		for dteType, docs := range typeGroups {
			if err := s.processSystemDocumentsByType(ctx, systemNIT, dteType, docs); err != nil {
				logs.Error("Failed to process system documents", map[string]interface{}{
					"error":     err.Error(),
					"systemNIT": systemNIT,
					"dteType":   dteType,
				})
				continue
			}
		}
	}

	return nil
}

func (s *ContingencyService) expireOverdueDocuments(ctx context.Context, docs []dte.ContingencyDocument) []dte.ContingencyDocument {
	now := s.now()
	valid := make([]dte.ContingencyDocument, 0, len(docs))
	var expiredIDs, observations []string
	var expiredSummary []event.RejectedDocSummary

	for _, doc := range docs {
		if !IsContingencyWindowExpired(doc.EventStampedAt, now) {
			valid = append(valid, doc)
			continue
		}
		observation := fmt.Sprintf("Not transmitted within 72 hours after the contingency event stamp (%s)",
			doc.EventStampedAt.Format("2006-01-02 15:04:05"))
		expiredIDs = append(expiredIDs, doc.ID)
		observations = append(observations, observation)
		expiredSummary = append(expiredSummary, event.RejectedDocSummary{
			ControlNumber: controlNumberOf(doc),
			ErrorCode:     ContingencyExpiredCode,
			Description:   observation,
		})
	}

	if len(expiredIDs) == 0 {
		return valid
	}

	s.docSvc.markExpired(ctx, expiredIDs, observations)
	logs.Error("Contingency documents expired after 72 hours", map[string]interface{}{"count": len(expiredIDs)})
	s.publishAlert(ctx, "contingency_72h_expired", expiredSummary,
		"Contingency documents were not transmitted within 72 hours after the contingency event stamp and are considered not issued")

	return valid
}

func (s *ContingencyService) alertLongContingency(docs []dte.ContingencyDocument) {
	oldest := s.now()
	for _, doc := range docs {
		if !doc.CreatedAt.IsZero() && doc.CreatedAt.Before(oldest) {
			oldest = doc.CreatedAt
		}
	}

	if s.now().Sub(oldest) <= TechnicalReportThreshold {
		return
	}

	logs.Warn("Contingency longer than 3 days: technical report required", map[string]interface{}{
		"since": oldest.Format(time.RFC3339),
	})
	s.publishAlert(context.Background(), "contingency_technical_report", nil,
		fmt.Sprintf("Contingency started on %s and exceeds 3 days: a technical contingency report is required before transmitting the event",
			oldest.Format("2006-01-02 15:04:05")))
}

func (s *ContingencyService) publishAlert(ctx context.Context, jobName string, docs []event.RejectedDocSummary, message string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, event.RetransmissionJobFailedEvent{
		JobName:         jobName,
		FailedDocuments: docs,
		LastError:       message,
		OccurredAtTime:  s.now(),
	})
}

func (s *ContingencyService) now() time.Time {
	if s.timeProvider != nil {
		return s.timeProvider.Now()
	}
	return time.Now()
}

func controlNumberOf(doc dte.ContingencyDocument) string {
	if doc.Document != nil {
		return doc.Document.ControlNumber
	}
	return doc.DocumentID
}

func (s *ContingencyService) processSystemDocumentsByType(ctx context.Context, systemNIT string, dteType string, docs []dte.ContingencyDocument) error {
	if len(docs) == 0 {
		logs.Warn("No documents to process")
		return nil
	}

	branchID := docs[0].BranchID
	token, creds, err := s.txSvc.getTokenAndCreds(ctx, branchID)
	if err != nil {
		return err
	}

	batchSize := s.txSvc.config.GetBatchSize()
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := s.txSvc.processBatch(ctx, systemNIT, dteType, docs[i:end], token, *creds, branchID); err != nil {
			logs.Error("Batch failed, continuing with next batch", map[string]interface{}{
				"error":     err.Error(),
				"batchFrom": i,
				"batchTo":   end,
			})
		}
	}

	return nil
}

// verifyAndUpdateExistingDocuments queries Hacienda for each document of an event that already exists, stores the
// reported status and returns the contingency ids whose status is final (processed or rejected).
func (s *ContingencyService) verifyAndUpdateExistingDocuments(ctx context.Context, docs []dte.ContingencyDocument) map[string]bool {
	type verifiedEntry struct {
		contingencyID  string
		stamp          string
		processingDate string
	}
	byStatus := make(map[string][]verifiedEntry)

	for _, doc := range docs {
		logs.Info("Checking document status in Hacienda", map[string]interface{}{
			"documentID": doc.Document.ID,
			"dteType":    doc.Document.DTEType,
		})

		nit, err := extractNITFromDocument(doc.Document.JSONData)
		if err != nil {
			logs.Error("Failed to extract NIT from document", map[string]interface{}{
				"error":      err.Error(),
				"documentID": doc.Document.ID,
			})
			continue
		}

		statusResult, err := s.txSvc.checkDocumentStatus(ctx, doc.Document.ID, nit, doc.Document.DTEType)
		if err != nil {
			logs.Error("Failed to check document status in Hacienda", map[string]interface{}{
				"error":      err.Error(),
				"documentID": doc.Document.ID,
			})
			continue
		}

		internalStatus := mapHaciendaStatusToInternal(statusResult.Status)

		updatedDoc := dte.DTEDetails{
			ID:               doc.Document.ID,
			DTEType:          doc.Document.DTEType,
			ControlNumber:    doc.Document.ControlNumber,
			ReceptionStamp:   statusResult.ReceptionStamp,
			Transmission:     constants.TransmissionContingency,
			Status:           internalStatus,
			JSONData:         doc.Document.JSONData,
			ProcessingDate:   statusResult.ProcessingDate,
			HaciendaResponse: statusResult.HaciendaResponse,
		}

		if err = s.docSvc.updateStatus(ctx, doc.BranchID, updatedDoc); err != nil {
			logs.Error("Failed to update document with Hacienda status", map[string]interface{}{
				"error":      err.Error(),
				"documentID": doc.Document.ID,
				"status":     statusResult.Status,
			})
			continue
		}

		stamp := ""
		if statusResult.ReceptionStamp != nil && *statusResult.ReceptionStamp != "" {
			stamp = *statusResult.ReceptionStamp
		}
		byStatus[internalStatus] = append(byStatus[internalStatus], verifiedEntry{
			contingencyID:  doc.ID,
			stamp:          stamp,
			processingDate: statusResult.ProcessingDate,
		})

		logs.Info("Document updated successfully with Hacienda status", map[string]interface{}{
			"documentID":     doc.Document.ID,
			"status":         statusResult.Status,
			"receptionStamp": statusResult.ReceptionStamp,
		})
	}

	settled := make(map[string]bool)
	for status, entries := range byStatus {
		ids := make([]string, len(entries))
		results := make(map[string]dte.ReceptionResult, len(entries))
		for i, e := range entries {
			ids[i] = e.contingencyID
			results[e.contingencyID] = dte.ReceptionResult{ReceptionStamp: e.stamp, ProcessingDate: e.processingDate}
			if status != constants.DocumentPending {
				settled[e.contingencyID] = true
			}
		}
		s.docSvc.markContingencyBatchVerified(ctx, ids, results, status)
	}
	return settled

}

// ContingencyEventExistsError is returned when Hacienda reports that the contingency event already exists.
type ContingencyEventExistsError struct {
	Message      string
	Documents    []dte.ContingencyDocument
	Observations interface{}
}

func (e *ContingencyEventExistsError) Error() string {
	return fmt.Sprintf("contingency event already exists: %s", e.Message)
}

// eventGroup holds the pending documents reported in the same contingency event: same issuer, branch (establishment
// and point of sale) and contingency type.
type eventGroup struct {
	systemNIT       string
	branchID        uint
	contingencyType int8
	documents       []dte.ContingencyDocument
}

// groupByEvent splits the pending documents by issuer NIT, branch and contingency type, keeping the order in which
// each group first appears.
func groupByEvent(docs []dte.ContingencyDocument) []eventGroup {
	var groups []eventGroup
	index := make(map[string]int)
	for _, doc := range docs {
		nit := doc.Branch.User.NIT
		key := fmt.Sprintf("%s|%d|%d", nit, doc.BranchID, doc.ContingencyType)
		position, exists := index[key]
		if !exists {
			position = len(groups)
			index[key] = position
			groups = append(groups, eventGroup{systemNIT: nit, branchID: doc.BranchID, contingencyType: doc.ContingencyType})
		}
		groups[position].documents = append(groups[position].documents, doc)
	}
	return groups
}

// withoutSettled returns the documents whose contingency id is not in the settled set.
func withoutSettled(docs []dte.ContingencyDocument, settled map[string]bool) []dte.ContingencyDocument {
	remaining := make([]dte.ContingencyDocument, 0, len(docs))
	for _, doc := range docs {
		if !settled[doc.ID] {
			remaining = append(remaining, doc)
		}
	}
	return remaining
}

// groupByDTEType splits the documents of one event by DTE type, since each batch carries a single DTE type.
func groupByDTEType(docs []dte.ContingencyDocument) map[string][]dte.ContingencyDocument {
	result := make(map[string][]dte.ContingencyDocument)
	for _, doc := range docs {
		result[doc.Document.DTEType] = append(result[doc.Document.DTEType], doc)
	}
	return result
}

func extractNITFromDocument(jsonData string) (string, error) {
	var docData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &docData); err != nil {
		return "", fmt.Errorf("failed to parse document JSON: %w", err)
	}

	emisor, ok := docData["emisor"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("failed to extract emisor from document")
	}

	nit, ok := emisor["nit"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract NIT from emisor")
	}

	return nit, nil
}

func mapHaciendaStatusToInternal(haciendaStatus string) string {
	switch haciendaStatus {
	case "PROCESADO":
		return constants.DocumentReceived
	case "RECHAZADO":
		return constants.DocumentRejected
	default:
		return constants.DocumentPending
	}
}

package contingency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	appPorts "github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	batch "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
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
		txSvc:             newContingencyTransmissionSvc(authManager, haciendaAuth, cache, tokenService, signer, batchTransmitter, cfg),
	}
}

func (s *ContingencyService) SetEventBus(bus event.Bus) {
	s.bus = bus
}

// StoreDocumentInContingency persists a DTE document under contingency mode and registers it for later retransmission.
func (s *ContingencyService) StoreDocumentInContingency(ctx context.Context, document interface{}, dteType string, contingencyType int8, reason string) error {
	claims := ctx.Value("claims").(*authModels.AuthClaims)

	contingencyDoc := &dte.ContingencyDocument{
		BranchID:        claims.BranchID,
		ContingencyType: contingencyType,
		Reason:          reason,
	}

	if err := s.docSvc.store(ctx, document, contingencyDoc); err != nil {
		return err
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

	return nil
}

// RetransmitPendingDocuments fetches all pending contingency documents, sends the contingency event
// to Hacienda, and retransmits each document batch by DTE type.
func (s *ContingencyService) RetransmitPendingDocuments(ctx context.Context) error {
	pendingDocs, err := s.docSvc.getPending(ctx, config.Server.MaxBatchSize)
	if err != nil {
		return shared_error.NewGeneralServiceError("ContingencyService", "RetransmitPendingDocuments", "failed to get pending documents", err)
	}

	if len(pendingDocs) == 0 {
		logs.Info("No pending documents found")
		return nil
	}

	docsBySystemAndType := groupBySystemAndType(pendingDocs)

	for systemNIT, typeGroups := range docsBySystemAndType {
		var docsForNIT []dte.ContingencyDocument
		for _, typeDocs := range typeGroups {
			docsForNIT = append(docsForNIT, typeDocs...)
		}

		if err := s.contingencyEvents.PrepareAndSendContingencyEvent(ctx, docsForNIT); err != nil {
			if contingencyErr, ok := err.(*ContingencyEventExistsError); ok {
				logs.Warn("Contingency event already exists, checking document status in Hacienda", map[string]interface{}{
					"error":     err.Error(),
					"systemNIT": systemNIT,
					"docCount":  len(contingencyErr.Documents),
				})
				s.verifyAndUpdateExistingDocuments(ctx, contingencyErr.Documents)
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

func (s *ContingencyService) verifyAndUpdateExistingDocuments(ctx context.Context, docs []dte.ContingencyDocument) {
	type verifiedEntry struct {
		contingencyID string
		stamp         string
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
			ID:             doc.Document.ID,
			DTEType:        doc.Document.DTEType,
			ControlNumber:  doc.Document.ControlNumber,
			ReceptionStamp: statusResult.ReceptionStamp,
			Transmission:   constants.TransmissionContingency,
			Status:         internalStatus,
			JSONData:       doc.Document.JSONData,
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
			contingencyID: doc.ID,
			stamp:         stamp,
		})

		logs.Info("Document updated successfully with Hacienda status", map[string]interface{}{
			"documentID":     doc.Document.ID,
			"status":         statusResult.Status,
			"receptionStamp": statusResult.ReceptionStamp,
		})
	}

	for status, entries := range byStatus {
		ids := make([]string, len(entries))
		stamps := make(map[string]string, len(entries))
		for i, e := range entries {
			ids[i] = e.contingencyID
			stamps[e.contingencyID] = e.stamp
		}
		s.docSvc.markContingencyBatchVerified(ctx, ids, stamps, status)
	}
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

func groupBySystemAndType(docs []dte.ContingencyDocument) map[string]map[string][]dte.ContingencyDocument {
	result := make(map[string]map[string][]dte.ContingencyDocument)
	for _, doc := range docs {
		nit := doc.Branch.User.NIT
		if result[nit] == nil {
			result[nit] = make(map[string][]dte.ContingencyDocument)
		}
		result[nit][doc.Document.DTEType] = append(result[nit][doc.Document.DTEType], doc)
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

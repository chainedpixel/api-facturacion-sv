package contingency

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// contingencyDocumentSvc manages DTE creation, contingency registration, and status updates.
type contingencyDocumentSvc struct {
	dteManager        dte_documents.DTEManager
	repo              ContingencyRepositoryPort
	sequentialManager dte_documents.SequentialNumberManager
}

func newContingencyDocumentSvc(
	dteManager dte_documents.DTEManager,
	repo ContingencyRepositoryPort,
	sequentialManager dte_documents.SequentialNumberManager,
) *contingencyDocumentSvc {
	return &contingencyDocumentSvc{
		dteManager:        dteManager,
		repo:              repo,
		sequentialManager: sequentialManager,
	}
}

// store creates the DTE record in contingency mode and registers the contingency document.
// It fills in the DocumentID on contingencyDoc from the extracted DTE identification.
func (s *contingencyDocumentSvc) store(ctx context.Context, document interface{}, contingencyDoc *dte.ContingencyDocument) error {
	dteInfo, err := utils.ExtractAuxiliarIdentification(document)
	if err != nil {
		logs.Error("Failed to extract general DTE info", map[string]interface{}{
			"error": err.Error(),
		})
		return shared_error.NewGeneralServiceError("ContingencyService", "StoreDocumentInContingency", "failed to extract general DTE info", err)
	}

	if dteInfo.Identification.GenerationCode == "" || dteInfo.Identification.ControlNumber == "" {
		return shared_error.NewFormattedGeneralServiceError("ContingencyService", "StoreDocumentInContingency", "MissingDTEIdentification")
	}

	if err = s.dteManager.Create(ctx, document, constants.TransmissionContingency, constants.DocumentPending, nil); err != nil {
		logs.Error("Failed to store DTE", map[string]interface{}{
			"error": err.Error(),
		})
		return shared_error.NewGeneralServiceError("ContingencyService", "StoreDocumentInContingency", "failed to store DTE", err)
	}

	contingencyDoc.DocumentID = dteInfo.Identification.GenerationCode

	if err = s.repo.Create(ctx, contingencyDoc); err != nil {
		logs.Error("Failed to store contingency document", map[string]interface{}{
			"error": err.Error(),
			"id":    contingencyDoc.ID,
		})
		return shared_error.NewGeneralServiceError("ContingencyService", "StoreDocumentInContingency", "failed to store contingency document", err)
	}

	controlNumber := dteInfo.Identification.ControlNumber
	if markErr := s.sequentialManager.MarkReservationAsContingencyByControlNumber(
		ctx, controlNumber, dteInfo.Identification.GenerationCode, contingencyDoc.ID, contingencyDoc.BranchID,
	); markErr != nil {
		logs.Warn("Failed to mark reservation as contingency", map[string]interface{}{
			"error":         markErr.Error(),
			"controlNumber": controlNumber,
			"documentID":    dteInfo.Identification.GenerationCode,
		})
	}

	logs.Info("Document stored in contingency", map[string]interface{}{
		"id":            contingencyDoc.ID,
		"controlNumber": controlNumber,
	})

	return nil
}

// getPending retrieves pending contingency documents up to maxSize.
func (s *contingencyDocumentSvc) getPending(ctx context.Context, maxSize int) ([]dte.ContingencyDocument, error) {
	return s.repo.GetPending(ctx, maxSize)
}

// updateStatus updates a DTE document's status with data received from Hacienda.
func (s *contingencyDocumentSvc) updateStatus(ctx context.Context, branchID uint, updatedDoc dte.DTEDetails) error {
	return s.dteManager.UpdateDTE(ctx, branchID, updatedDoc)
}

// markContingencyBatchVerified marks a batch of contingency records with the verified status
// and their reception stamps so that the records are not left without tracking information.
func (s *contingencyDocumentSvc) markContingencyBatchVerified(ctx context.Context, ids []string, stamps map[string]string, status string) {
	if len(ids) == 0 {
		return
	}
	if err := s.repo.UpdateBatch(ctx, ids, nil, stamps, "verified", "", status); err != nil {
		logs.Warn("Failed to mark contingency records as verified", map[string]interface{}{
			"error": err.Error(),
			"count": len(ids),
		})
	}
}

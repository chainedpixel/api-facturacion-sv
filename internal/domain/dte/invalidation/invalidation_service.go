package invalidation

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/validator"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type invalidationService struct {
	validator  *validator.InvalidationRulesValidator
	dteManager dte_documents.DTEManager
}

// NewInvalidationService creates a new instance of InvalidationManager
func NewInvalidationService(dteManager dte_documents.DTEManager) InvalidationManager {
	return &invalidationService{
		validator:  validator.NewInvalidationRulesValidator(nil),
		dteManager: dteManager,
	}
}

func (s *invalidationService) InvalidateDocument(ctx context.Context, branchID uint, originalCode string) error {
	err := s.dteManager.UpdateDTE(ctx, branchID, dte.DTEDetails{
		ID:     originalCode,
		Status: constants.DocumentInvalid,
	})
	if err != nil {
		return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "FailedToInvalidatedDTE")
	}

	doc, err := s.dteManager.GetByGenerationCode(ctx, branchID, originalCode)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "FailedToGetDTE")
	}

	if doc.Details.DTEType == constants.NotaCreditoElectronica || doc.Details.DTEType == constants.NotaDebitoElectronica {
		err = s.handleControlBalance(ctx, branchID, originalCode, doc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *invalidationService) Validate(ctx context.Context, branchID uint, document *invalidation_models.InvalidationDocument) error {
	s.validator = validator.NewInvalidationRulesValidator(document)
	if err := s.validator.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *invalidationService) ValidateStatus(ctx context.Context, branchID uint, req structs.CreateInvalidationRequest) error {
	if err := s.validateDTEStatus(ctx,
		branchID,
		req.GenerationCode,
		"document to invalidate",
	); err != nil {
		return err
	}

	if req.Reason.Type != 2 && req.ReplacementGenerationCode != nil {
		if err := s.validateDTEStatus(ctx,
			branchID,
			*req.ReplacementGenerationCode,
			"replacement document",
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *invalidationService) handleControlBalance(ctx context.Context, branchID uint, originalCode string, doc *dte.DTEDocument) error {
	extractor := utils.ExtractRelatedDocAndItemsFromStringJSON(doc.Details.JSONData)

	relatedDocsMap := make(map[string]struct {
		GenerationType int
		DocumentNumber string
	})
	for _, rd := range extractor.RelatedDocs {
		relatedDocsMap[rd.DocumentNumber] = struct {
			GenerationType int
			DocumentNumber string
		}(rd)
	}

	for _, item := range extractor.Items {
		relatedDoc, ok := relatedDocsMap[item.RelatedDoc]
		if ok && relatedDoc.GenerationType == constants.ElectronicDocument {
			logs.Info("Generating balance control transaction", map[string]interface{}{
				"branchID":         branchID,
				"originalCode":     originalCode,
				"relatedDocument":  relatedDoc.DocumentNumber,
				"taxedAmount":      item.TaxedAmount,
				"exemptAmount":     item.ExemptAmount,
				"notSubjectAmount": item.NotSubjectAmount,
			})

			if err := s.dteManager.GenerateBalanceTransactionWithAmounts(ctx,
				branchID,
				constants.DocumentInvalid,
				relatedDoc.DocumentNumber,
				originalCode,
				item.TaxedAmount,
				item.ExemptAmount,
				item.NotSubjectAmount); err != nil {
				logs.Error("Error generating balance transaction", map[string]interface{}{
					"error": err.Error(),
				})
				return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "FailedToRecoverInvalidatedAmounts")
			}
		}
	}

	logs.Info("Balance control transaction generated successfully, the amounts was added to the balance", map[string]interface{}{
		"branchID":     branchID,
		"originalCode": originalCode,
	})
	return nil
}

func (s *invalidationService) validateDTEStatus(ctx context.Context, branchID uint, originalCode, message string) error {
	status, err := s.dteManager.VerifyStatus(ctx, branchID, originalCode)
	if err != nil {
		return err
	}
	return s.handleError(status, message)
}

func (s *invalidationService) handleError(status, message string) error {
	switch status {
	case constants.DocumentInvalid:
		return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "DocumentAlreadyInvalid", message)
	case constants.DocumentRejected:
		return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "DocumentReject", message)
	case constants.DocumentPending:
		return shared_error.NewFormattedGeneralServiceError("InvalidationService", "InvalidateDocument", "DocumentPending", message)
	default:
		return nil
	}
}

package credit_note

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	buisnessValidator "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type creditNoteService struct {
	validator        *validator.CreditNoteRulesValidator
	seqNumberManager dte_documents.SequentialNumberManager
	dteManager       dte_documents.DTEManager
}

// NewCreditNoteService Creates a new Credit Note service.
func NewCreditNoteService(seqNumberManager dte_documents.SequentialNumberManager, dteManager dte_documents.DTEManager) ports.DTEService {
	return &creditNoteService{
		validator:        validator.NewCreditNoteRulesValidator(nil),
		seqNumberManager: seqNumberManager,
		dteManager:       dteManager,
	}
}

// Create Creates a new electronic Credit Note based on the provided data.
func (s *creditNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
	data := input.(*credit_note_models.CreditNoteInput)
	if err := s.validateRelatedDocs(ctx, data, branchID); err != nil {
		logs.Error("Failed to validate related documents", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	baseDoc := createBaseDocument(data)
	creditNote := &credit_note_models.CreditNoteModel{
		DTEDocument:   baseDoc,
		CreditItems:   data.Items,
		CreditSummary: *data.CreditSummary,
	}

	if err := s.validate(creditNote); err != nil {
		logs.Error("Failed to validate credit note document basic validation", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := buisnessValidator.ValidateDTEDocument(creditNote); err != nil {
		logs.Error("Failed to validate credit note document generic validations", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	for _, doc := range creditNote.RelatedDocuments {
		if doc.GetGenerationType() == constants.ElectronicDocument {
			if err := s.dteManager.ValidateForCreditNote(ctx, branchID, doc.GetDocumentNumber(), creditNote); err != nil {
				logs.Error("Failed to validate credit note document totals", map[string]interface{}{"error": err.Error()})
				return nil, err
			}
		}
	}

	if err := s.generateCodeAndIdentifiers(ctx, creditNote, branchID); err != nil {
		return nil, err
	}

	return creditNote, nil
}

// validateRelatedDocs verifies that the related documents exist in the database
func (s *creditNoteService) validateRelatedDocs(ctx context.Context, data *credit_note_models.CreditNoteInput, branchID uint) error {
	if data.RelatedDocs == nil || len(data.RelatedDocs) == 0 {
		return shared_error.NewFormattedGeneralServiceError(
			"CreditNoteService",
			"validateRelatedDocs",
			"NoRelatedDocs",
		)
	}

	for i, relatedDoc := range data.RelatedDocs {
		doc, err := s.dteManager.GetByGenerationCode(ctx, branchID, relatedDoc.GetDocumentNumber())
		if err != nil {
			return err
		}

		status, err := s.dteManager.VerifyStatus(ctx, branchID, relatedDoc.GetDocumentNumber())
		if err != nil {
			return err
		}

		if status != constants.DocumentReceived {
			return shared_error.NewFormattedGeneralServiceError(
				"CreditNoteService",
				"validateRelatedDocs",
				"DocumentNotReceived",
				relatedDoc.GetDocumentNumber(),
				status,
			)
		}

		data.RelatedDocs[i].EmissionDate = *temporal.NewValidatedEmissionDate(doc.CreatedAt)
		extractor, err := utils.ExtractDTEReceiverFromString(doc.Details.JSONData)
		if err != nil {
			return err
		}

		if data.Receiver.NIT.GetValue() != extractor.Receiver.NIT {
			return shared_error.NewFormattedGeneralServiceError(
				"CreditNoteService",
				"validateRelatedDocs",
				"NotMatchingReceiverNIT",
			)
		}
	}

	return nil
}

// Validate Validates an electronic Credit Note based on the business rules.
func (s *creditNoteService) validate(creditNote *credit_note_models.CreditNoteModel) error {
	s.validator = validator.NewCreditNoteRulesValidator(creditNote)
	err := s.validator.Validate()
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"CreditNoteService",
			"Validate",
			err,
			"ValidationFailed",
		)
	}
	return nil
}

// generateControlNumber Generates a unique control number for the Credit Note.
func (s *creditNoteService) generateControlNumber(ctx context.Context, creditNote *credit_note_models.CreditNoteModel, branchID uint) error {
	establishmentCode := creditNote.Issuer.GetEstablishmentCode()
	posCode := creditNote.Issuer.GetPOSCode()

	controlNumber, err := s.seqNumberManager.GetNextControlNumber(
		ctx,
		constants.NotaCreditoElectronica,
		branchID,
		posCode,
		establishmentCode,
	)
	if err != nil {
		return err
	}

	err = creditNote.Identification.SetControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"CreditNoteService",
			"GenerateControlNumber",
			err,
			"FailedToSetControlNumber",
		)
	}
	return nil
}

// generateCodeAndIdentifiers Generates the UUID code and control number of the Credit Note.
func (s *creditNoteService) generateCodeAndIdentifiers(ctx context.Context, creditNote *credit_note_models.CreditNoteModel, branchID uint) error {
	err := creditNote.Identification.GenerateCode()
	if err != nil {
		return err
	}

	return s.generateControlNumber(ctx, creditNote, branchID)
}

// createBaseDocument Creates a base document for the electronic Credit Note.
func createBaseDocument(data *credit_note_models.CreditNoteInput) *models.DTEDocument {
	var extInterface interfaces.Extension
	var thirdPartySale interfaces.ThirdPartySale
	var appendixes []interfaces.Appendix
	var otherDocuments []interfaces.OtherDocuments
	var relatedDocuments []interfaces.RelatedDocument

	baseItems := make([]interfaces.Item, len(data.Items))
	for i, item := range data.Items {
		baseItems[i] = &item
	}

	if data.Appendixes != nil {
		for _, appendix := range data.Appendixes {
			appendixes = append(appendixes, &appendix)
		}
	}

	if data.Extension != nil {
		extInterface = data.Extension
	}

	if data.RelatedDocs != nil {
		for _, relatedDoc := range data.RelatedDocs {
			relatedDocuments = append(relatedDocuments, &relatedDoc)
		}
	}

	if data.OtherDocs != nil {
		for _, otherDoc := range data.OtherDocs {
			otherDocuments = append(otherDocuments, &otherDoc)
		}
	}

	if data.ThirdPartySale != nil {
		thirdPartySale = data.ThirdPartySale
	}

	return &models.DTEDocument{
		Identification:   data.Identification,
		Issuer:           data.Issuer,
		Receiver:         data.Receiver,
		Items:            baseItems,
		RelatedDocuments: relatedDocuments,
		OtherDocuments:   otherDocuments,
		Summary:          data.CreditSummary.Summary,
		ThirdPartySale:   thirdPartySale,
		Extension:        extInterface,
		Appendix:         appendixes,
	}
}

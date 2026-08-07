package remission_note

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	buisnessValidator "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type remissionNoteService struct {
	validator        *validator.RemissionNoteRulesValidator
	seqNumberManager dte_documents.SequentialNumberManager
	dteManager       dte_documents.DTEManager
}

// NewRemissionNoteService creates a new Remission Note service.
func NewRemissionNoteService(seqNumberManager dte_documents.SequentialNumberManager, dteManager dte_documents.DTEManager) ports.DTEService {
	return &remissionNoteService{
		validator:        validator.NewRemissionNoteRulesValidator(nil),
		seqNumberManager: seqNumberManager,
		dteManager:       dteManager,
	}
}

// Create creates a new electronic Remission Note based on the provided data.
func (s *remissionNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
	data := input.(*remission_note_models.RemissionNoteInput)

	if data.RemissionSummary == nil {
		return nil, shared_error.NewFormattedGeneralServiceError(
			"RemissionNoteService", "Create", "SummaryIsRequired",
		)
	}

	baseDoc := createBaseDocument(data)
	remissionNote := &remission_note_models.RemissionNoteModel{
		DTEDocument:    baseDoc,
		RemissionItems: data.Items,
		Summary:        data.RemissionSummary,
	}

	if err := s.validate(remissionNote); err != nil {
		logs.Error("Failed to validate remission note document basic validation", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := buisnessValidator.ValidateDTEDocument(remissionNote); err != nil {
		logs.Error("Failed to validate remission note document generic validations", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := s.generateCodeAndIdentifiers(ctx, remissionNote, branchID); err != nil {
		return nil, err
	}

	return remissionNote, nil
}

// validate validates an electronic Remission Note based on business rules.
func (s *remissionNoteService) validate(remissionNote *remission_note_models.RemissionNoteModel) error {
	s.validator = validator.NewRemissionNoteRulesValidator(remissionNote)
	err := s.validator.Validate()
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"RemissionNoteService",
			"Validate",
			err,
			"ValidationFailed",
		)
	}
	return nil
}

// generateControlNumber generates a unique control number for the Remission Note.
func (s *remissionNoteService) generateControlNumber(ctx context.Context, remissionNote *remission_note_models.RemissionNoteModel, branchID uint) error {
	establishmentCode := remissionNote.Issuer.GetEstablishmentCode()
	posCode := remissionNote.Issuer.GetPOSCode()

	controlNumber, err := s.seqNumberManager.GetNextControlNumber(
		ctx,
		constants.NotaRemisionElectronica,
		branchID,
		posCode,
		establishmentCode,
	)
	if err != nil {
		return err
	}

	err = remissionNote.Identification.SetControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"RemissionNoteService",
			"GenerateControlNumber",
			err,
			"FailedToSetControlNumber",
		)
	}
	return nil
}

// generateCodeAndIdentifiers generates the UUID code and control number for the Remission Note.
func (s *remissionNoteService) generateCodeAndIdentifiers(ctx context.Context, remissionNote *remission_note_models.RemissionNoteModel, branchID uint) error {
	err := remissionNote.Identification.GenerateCode()
	if err != nil {
		return err
	}

	return s.generateControlNumber(ctx, remissionNote, branchID)
}

// createBaseDocument creates a base document for the electronic Remission Note.
func createBaseDocument(data *remission_note_models.RemissionNoteInput) *models.DTEDocument {
	var extInterface interfaces.Extension
	var thirdPartySale interfaces.ThirdPartySale
	var appendixes []interfaces.Appendix
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

	if data.ThirdPartySale != nil {
		thirdPartySale = data.ThirdPartySale
	}

	var receiver interfaces.Receiver
	if data.Receiver != nil {
		receiver = data.Receiver
	}

	return &models.DTEDocument{
		Identification:   data.Identification,
		Issuer:           data.Issuer,
		Receiver:         receiver,
		Items:            baseItems,
		RelatedDocuments: relatedDocuments,
		Summary:          data.RemissionSummary.Summary,
		ThirdPartySale:   thirdPartySale,
		Extension:        extInterface,
		Appendix:         appendixes,
	}
}

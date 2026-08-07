package debit_note

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	buisnessValidator "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/shopspring/decimal"
)

type debitNoteService struct {
	validator        *validator.DebitNoteRulesValidator
	seqNumberManager dte_documents.SequentialNumberManager
	dteManager       dte_documents.DTEManager
}

func NewDebitNoteService(seqNumberManager dte_documents.SequentialNumberManager, dteManager dte_documents.DTEManager) ports.DTEService {
	return &debitNoteService{
		validator:        validator.NewDebitNoteRulesValidator(nil),
		seqNumberManager: seqNumberManager,
		dteManager:       dteManager,
	}
}

func (s *debitNoteService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
	data := input.(*debit_note_models.DebitNoteInput)

	if err := s.validateRelatedDocs(ctx, data, branchID); err != nil {
		logs.Error("Failed to validate related documents", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	baseDoc := createBaseDocument(data)
	debitNote := &debit_note_models.DebitNoteModel{
		DTEDocument:  baseDoc,
		DebitItems:   data.Items,
		DebitSummary: *data.DebitSummary,
	}

	if err := s.validate(debitNote); err != nil {
		logs.Error("Failed to validate debit note document basic validation", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := buisnessValidator.ValidateDTEDocument(debitNote); err != nil {
		logs.Error("Failed to validate debit note document generic validations", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	for _, doc := range debitNote.RelatedDocuments {
		if doc.GetGenerationType() == constants.ElectronicDocument {
			if err := s.dteManager.ValidateForDebitNote(ctx, branchID, doc.GetDocumentNumber(), debitNote); err != nil {
				logs.Error("Failed to validate debit note document totals", map[string]interface{}{"error": err.Error()})
				return nil, err
			}
		}
	}

	if err := s.generateCodeAndIdentifiers(ctx, debitNote, branchID); err != nil {
		return nil, err
	}

	if err := s.calculateTotalToPay(debitNote); err != nil {
		return nil, err
	}

	return debitNote, nil
}

func (s *debitNoteService) validateRelatedDocs(ctx context.Context, data *debit_note_models.DebitNoteInput, branchID uint) error {
	if data.RelatedDocs == nil || len(data.RelatedDocs) == 0 {
		return shared_error.NewFormattedGeneralServiceError(
			"DebitNoteService",
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
				"DebitNoteService",
				"validateRelatedDocs",
				"RelatedDocumentNotReceived",
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
				"DebitNoteService",
				"validateRelatedDocs",
				"NotMatchingReceiverNIT",
			)
		}
	}

	return nil
}

func (s *debitNoteService) validate(debitNote *debit_note_models.DebitNoteModel) error {
	s.validator = validator.NewDebitNoteRulesValidator(debitNote)
	err := s.validator.Validate()
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"DebitNoteService",
			"Validate",
			err,
			"ValidationFailed",
		)
	}
	return nil
}

func (s *debitNoteService) generateControlNumber(ctx context.Context, debitNote *debit_note_models.DebitNoteModel, branchID uint) error {
	establishmentCode := debitNote.Issuer.GetEstablishmentCode()
	posCode := debitNote.Issuer.GetPOSCode()

	controlNumber, err := s.seqNumberManager.GetNextControlNumber(
		ctx,
		constants.NotaDebitoElectronica,
		branchID,
		posCode,
		establishmentCode,
	)
	if err != nil {
		return err
	}

	err = debitNote.Identification.SetControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"DebitNoteService",
			"GenerateControlNumber",
			err,
			"FailedToSetControlNumber",
		)
	}
	return nil
}

func (s *debitNoteService) generateCodeAndIdentifiers(ctx context.Context, debitNote *debit_note_models.DebitNoteModel, branchID uint) error {
	err := debitNote.Identification.GenerateCode()
	if err != nil {
		return err
	}

	return s.generateControlNumber(ctx, debitNote, branchID)
}

func (s *debitNoteService) calculateTotalToPay(debitNote *debit_note_models.DebitNoteModel) error {
	expectedSubTotal := decimal.NewFromFloat(debitNote.DebitSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(debitNote.DebitSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(debitNote.DebitSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(debitNote.DebitSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(debitNote.DebitSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(debitNote.DebitSummary.NonSubjectDiscount.GetValue()))

	expectedTotalOperation := expectedSubTotal

	taxedAmount := decimal.NewFromFloat(debitNote.DebitSummary.TotalTaxed.GetValue())
	if taxedAmount.GreaterThan(decimal.Zero) {
		perception := decimal.NewFromFloat(debitNote.DebitSummary.IVAPerception.GetValue())
		expectedTotalOperation = expectedTotalOperation.Add(perception)

		ivaRetention := decimal.NewFromFloat(debitNote.DebitSummary.IVARetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(ivaRetention)

		incomeRetention := decimal.NewFromFloat(debitNote.DebitSummary.IncomeRetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(incomeRetention)
	}

	for _, tax := range debitNote.DebitSummary.TotalTaxes {
		expectedTotalOperation = expectedTotalOperation.Add(decimal.NewFromFloat(tax.GetValue()))
	}

	totalNonTaxed := decimal.NewFromFloat(debitNote.DebitSummary.TotalNonTaxed.GetValue())
	if totalNonTaxed.GreaterThan(decimal.Zero) {
		expectedTotalOperation = expectedTotalOperation.Add(totalNonTaxed)
	}

	debitNote.DebitSummary.SetForceTotalToPay(expectedTotalOperation.InexactFloat64())
	return nil
}

func createBaseDocument(data *debit_note_models.DebitNoteInput) *models.DTEDocument {
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
		Summary:          data.DebitSummary.Summary,
		ThirdPartySale:   thirdPartySale,
		Extension:        extInterface,
		Appendix:         appendixes,
	}
}

package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/credit_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// CreditNoteMapper maps credit note creation requests to the credit note domain model.
type CreditNoteMapper struct{}

// NewCreditNoteMapper creates a new CreditNoteMapper instance.
func NewCreditNoteMapper() *CreditNoteMapper {
	return &CreditNoteMapper{}
}

// MapToCreditNoteData converts a CreateCreditNoteRequest to a CreditNoteInput domain model.
func (m *CreditNoteMapper) MapToCreditNoteData(req *structs.CreateCreditNoteRequest, client *dte.IssuerDTE) (*credit_note_models.CreditNoteInput, error) {
	if err := validateCreditNoteRequest(req); err != nil {
		return nil, err
	}

	items, err := credit_note.MapCreditNoteItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->Items")
	}

	receiver, err := credit_note.MapCreditNoteRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->Receiver")
	}

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 3, constants.NotaCreditoElectronica)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->Identification")
	}

	summary, err := credit_note.MapCreditNoteRequestSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->Issuer")
	}

	relatedDocs, err := common.MapCommonRequestRelatedDocuments(req.RelatedDocs)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->RelatedDocs")
	}

	result := &credit_note_models.CreditNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
			Receiver:       receiver,
			RelatedDocs:    relatedDocs,
		},
		Items:         items,
		CreditSummary: summary,
	}

	if err = mapCreditNoteOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateCreditNoteRequest validates that the credit note creation request is correct.
func validateCreditNoteRequest(req *structs.CreateCreditNoteRequest) error {
	if req == nil {
		return dte_errors.NewValidationError("RequiredField", "Request")
	}
	if req.Items == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Items")
	}
	if req.Summary == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Summary")
	}
	if req.Receiver == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver")
	}
	if len(req.RelatedDocs) == 0 {
		return dte_errors.NewValidationError("RequiredField", "Request->RelatedDocs")
	}
	return common.ValidateRelatedDocs(req.RelatedDocs)
}

// mapCreditNoteOptionalFields maps the optional fields of the credit note request into the result model.
func mapCreditNoteOptionalFields(req *structs.CreateCreditNoteRequest, result *credit_note_models.CreditNoteInput) error {
	if err := common.MapCommonOptionalToInputData(common.CommonOptionalFields{
		ThirdPartySale: req.ThirdPartySale,
		Extension:      req.Extension,
		OtherDocs:      req.OtherDocs,
		Appendixes:     req.Appendixes,
	}, result.InputDataCommon); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->OptionalFields")
	}

	if req.Payments != nil {
		payments, err := common.MapCommonRequestPaymentsType(req.Payments)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("CreditNoteMapper", "MapToCreditNoteData", err, "ErrorMapping", "CreditNote->PaymentTypes")
		}
		result.CreditSummary.PaymentTypes = payments
	}

	return nil
}

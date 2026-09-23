package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/debit_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// DebitNoteMapper maps debit note creation requests to the debit note domain model.
type DebitNoteMapper struct{}

// NewDebitNoteMapper creates a new DebitNoteMapper instance.
func NewDebitNoteMapper() *DebitNoteMapper {
	return &DebitNoteMapper{}
}

// MapToDebitNoteData converts a CreateDebitNoteRequest to a DebitNoteInput domain model.
func (m *DebitNoteMapper) MapToDebitNoteData(req *structs.CreateDebitNoteRequest, client *dte.IssuerDTE) (*debit_note_models.DebitNoteInput, error) {
	if err := validateDebitNoteRequest(req); err != nil {
		return nil, err
	}

	items, err := debit_note.MapDebitNoteItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->Items")
	}

	receiver, err := debit_note.MapDebitNoteRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->Receiver")
	}

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 4, constants.NotaDebitoElectronica)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->Identification")
	}

	summary, err := debit_note.MapDebitNoteRequestSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->Issuer")
	}

	relatedDocs, err := common.MapCommonRequestRelatedDocuments(req.RelatedDocs)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->RelatedDocs")
	}

	result := &debit_note_models.DebitNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
			Receiver:       receiver,
			RelatedDocs:    relatedDocs,
		},
		Items:        items,
		DebitSummary: summary,
	}

	if err = mapDebitNoteOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateDebitNoteRequest validates that the debit note creation request is correct.
func validateDebitNoteRequest(req *structs.CreateDebitNoteRequest) error {
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

// mapDebitNoteOptionalFields maps the optional fields of the debit note request into the result model.
func mapDebitNoteOptionalFields(req *structs.CreateDebitNoteRequest, result *debit_note_models.DebitNoteInput) error {
	if err := common.MapCommonOptionalToInputData(common.CommonOptionalFields{
		ThirdPartySale: req.ThirdPartySale,
		OtherDocs:      req.OtherDocs,
		Appendixes:     req.Appendixes,
	}, result.InputDataCommon); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->OptionalFields")
	}

	if req.Payments != nil {
		payments, err := common.MapCommonRequestPaymentsType(req.Payments)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("DebitNoteMapper", "MapToDebitNoteData", err, "ErrorMapping", "DebitNote->PaymentTypes")
		}
		result.DebitSummary.PaymentTypes = payments
	}

	return nil
}

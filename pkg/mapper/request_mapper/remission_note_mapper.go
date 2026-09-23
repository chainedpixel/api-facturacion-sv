package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/remission_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// RemissionNoteMapper maps remission note creation requests to the remission note domain model.
type RemissionNoteMapper struct{}

// NewRemissionNoteMapper creates a new RemissionNoteMapper instance.
func NewRemissionNoteMapper() *RemissionNoteMapper {
	return &RemissionNoteMapper{}
}

// MapToRemissionNoteData converts a CreateRemissionNoteRequest to a RemissionNoteInput domain model.
func (m *RemissionNoteMapper) MapToRemissionNoteData(req *structs.CreateRemissionNoteRequest, client *dte.IssuerDTE) (*remission_note_models.RemissionNoteInput, error) {
	if err := validateRemissionNoteRequest(req); err != nil {
		return nil, err
	}

	items, err := remission_note.MapRemissionNoteItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Items")
	}

	receiver, err := remission_note.MapRemissionNoteRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Receiver")
	}

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 4, constants.NotaRemisionElectronica)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Identification")
	}

	summary, err := remission_note.MapRemissionNoteSummaryFromRequest(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Issuer")
	}

	result := &remission_note_models.RemissionNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
		},
		Receiver:         receiver,
		Items:            items,
		RemissionSummary: summary,
	}

	if len(req.RelatedDocs) > 0 {
		relatedDocsValues := make([]structs.RelatedDocRequest, len(req.RelatedDocs))
		for i, doc := range req.RelatedDocs {
			relatedDocsValues[i] = *doc
		}
		relatedDocs, err := common.MapCommonRequestRelatedDocuments(relatedDocsValues)
		if err != nil {
			return nil, shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->RelatedDocs")
		}
		result.RelatedDocs = relatedDocs
	}

	if err = mapRemissionNoteOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateRemissionNoteRequest validates that the remission note creation request is correct.
func validateRemissionNoteRequest(req *structs.CreateRemissionNoteRequest) error {
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
	if req.Receiver.BienTitulo == nil || *req.Receiver.BienTitulo == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver->BienTitulo")
	}

	if req.RelatedDocs != nil {
		relatedDocsValues := make([]structs.RelatedDocRequest, len(req.RelatedDocs))
		for i, doc := range req.RelatedDocs {
			relatedDocsValues[i] = *doc
		}
		return common.ValidateRelatedDocs(relatedDocsValues)
	}

	return nil
}

// mapRemissionNoteOptionalFields maps the optional fields of the remission note request into the result model.
func mapRemissionNoteOptionalFields(req *structs.CreateRemissionNoteRequest, result *remission_note_models.RemissionNoteInput) error {
	if req.ThirdPartySale != nil {
		thirdPartySale, err := common.MapCommonRequestThirdPartySale(req.ThirdPartySale)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->ThirdPartySales")
		}
		result.ThirdPartySale = thirdPartySale
	}

	if req.Summary.Payments != nil {
		paymentsValues := make([]structs.PaymentRequest, len(req.Summary.Payments))
		for i, payment := range req.Summary.Payments {
			paymentsValues[i] = *payment
		}
		payments, err := common.MapCommonRequestPaymentsType(paymentsValues)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->PaymentTypes")
		}
		result.RemissionSummary.PaymentTypes = payments
	}

	if req.Appendixes != nil {
		appendixValues := make([]structs.AppendixRequest, len(req.Appendixes))
		for i, appendix := range req.Appendixes {
			appendixValues[i] = *appendix
		}
		appendixes, err := common.MapCommonRequestAppendix(appendixValues)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("RemissionNoteMapper", "MapToRemissionNoteData", err, "ErrorMapping", "RemissionNote->Appendixes")
		}
		result.Appendixes = appendixes
	}

	return nil
}

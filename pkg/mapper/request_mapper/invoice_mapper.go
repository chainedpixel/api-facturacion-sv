package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/invoice"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// InvoiceMapper maps invoice creation requests to the invoice domain model.
type InvoiceMapper struct{}

// NewInvoiceMapper creates a new InvoiceMapper instance.
func NewInvoiceMapper() *InvoiceMapper {
	return &InvoiceMapper{}
}

// MapToInvoiceData converts a CreateInvoiceRequest to an InvoiceData domain model.
func (m *InvoiceMapper) MapToInvoiceData(req *structs.CreateInvoiceRequest, client *dte.IssuerDTE) (*invoice_models.InvoiceData, error) {
	if err := validateInvoiceRequest(req); err != nil {
		return nil, err
	}

	items, err := invoice.MapInvoiceItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->Items")
	}

	receiver, err := common.MapCommonRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->Receiver")
	}

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 1, constants.FacturaElectronica)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->Identification")
	}

	summary, err := invoice.MapInvoiceRequestSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->Issuer")
	}

	result := &invoice_models.InvoiceData{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
			Receiver:       receiver,
		},
		Items:          items,
		InvoiceSummary: summary,
	}

	if err = mapInvoiceOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateInvoiceRequest validates the required fields in the invoice creation request.
func validateInvoiceRequest(req *structs.CreateInvoiceRequest) error {
	if req == nil {
		return dte_errors.NewValidationError("RequiredField", "Request")
	}
	if req.Items == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Items")
	}
	if req.Summary == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Summary")
	}
	if req.Receiver != nil {
		if req.Receiver.DocumentType != nil && req.Receiver.DocumentNumber == nil || req.Receiver.DocumentType == nil && req.Receiver.DocumentNumber != nil {
			return shared_error.NewFormattedGeneralServiceError("InvoiceMapper", "MapToInvoiceData", "InvalidDocumentTypeAndNumber")
		}
		if req.Receiver.NIT != nil {
			return dte_errors.NewValidationError("InvalidFieldValue", "Request->Receiver->NIT")
		}
	}
	return nil
}

// mapInvoiceOptionalFields maps optional fields (ThirdPartySale, Extension, Payments, etc.)
// from the request into the result domain model.
func mapInvoiceOptionalFields(req *structs.CreateInvoiceRequest, result *invoice_models.InvoiceData) error {
	if err := common.MapCommonOptionalToInputData(common.CommonOptionalFields{
		ThirdPartySale: req.ThirdPartySale,
		Extension:      req.Extension,
		OtherDocs:      req.OtherDocs,
		RelatedDocs:    req.RelatedDocs,
		Appendixes:     req.Appendixes,
	}, result.InputDataCommon); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->OptionalFields")
	}

	if req.Payments != nil {
		payments, err := common.MapCommonRequestPaymentsType(req.Payments)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("InvoiceMapper", "MapToInvoiceData", err, "ErrorMapping", "Invoice->PaymentTypes")
		}
		result.InvoiceSummary.PaymentTypes = payments
	}

	return nil
}

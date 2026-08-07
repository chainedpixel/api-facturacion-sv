package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/ccf"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// CCFMapper maps Comprobante de Crédito Fiscal creation requests to the CCF domain model.
type CCFMapper struct{}

// NewCCFMapper creates a new CCFMapper instance.
func NewCCFMapper() *CCFMapper {
	return &CCFMapper{}
}

// MapToCCFData converts a CreateCreditFiscalRequest to a CCFData domain model.
func (m *CCFMapper) MapToCCFData(req *structs.CreateCreditFiscalRequest, client *dte.IssuerDTE) (*ccf_models.CCFData, error) {
	if err := validateCCFRequest(req); err != nil {
		return nil, err
	}

	items, err := ccf.MapCCFItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->Items")
	}

	receiver, err := ccf.MapCCFRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->Receiver")
	}

	if req.Receiver.CommercialName == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Request->Receiver->CommercialName")
	}
	receiver.CommercialName = req.Receiver.CommercialName

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 3, constants.CCFElectronico)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->Identification")
	}

	summary, err := ccf.MapCCFRequestSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->Issuer")
	}

	result := &ccf_models.CCFData{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
			Receiver:       receiver,
		},
		Items:         items,
		CreditSummary: summary,
	}

	if err = mapCCFOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateCCFRequest validates that the CCF creation request is correct.
func validateCCFRequest(req *structs.CreateCreditFiscalRequest) error {
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
	if req.Receiver.CommercialName == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver->CommercialName")
	}
	if req.Summary.TotalIVA != 0 {
		return dte_errors.NewValidationError("InvalidField", "Request->Summary->TotalIVA")
	}
	if req.Receiver.DocumentType != nil {
		return dte_errors.NewValidationError("InvalidField", "Request->Receiver->DocumentType")
	}
	if req.Receiver.DocumentNumber != nil {
		return dte_errors.NewValidationError("InvalidField", "Request->Receiver->DocumentNumber")
	}
	return nil
}

// mapCCFOptionalFields maps the optional fields of the CCF creation request into the result model.
func mapCCFOptionalFields(req *structs.CreateCreditFiscalRequest, result *ccf_models.CCFData) error {
	if err := common.MapCommonOptionalToInputData(common.CommonOptionalFields{
		ThirdPartySale: req.ThirdPartySale,
		Extension:      req.Extension,
		OtherDocs:      req.OtherDocs,
		RelatedDocs:    req.RelatedDocs,
		Appendixes:     req.Appendixes,
	}, result.InputDataCommon); err != nil {
		return shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->OptionalFields")
	}

	if req.Payments != nil {
		payments, err := common.MapCommonRequestPaymentsType(req.Payments)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("CCFMapper", "MapToCCFData", err, "ErrorMapping", "CCF->PaymentTypes")
		}
		result.CreditSummary.PaymentTypes = payments
	}

	return nil
}

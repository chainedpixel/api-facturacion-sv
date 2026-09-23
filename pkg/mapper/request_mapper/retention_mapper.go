package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/retention"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// RetentionMapper maps retention creation requests to the retention domain model.
type RetentionMapper struct{}

// NewRetentionMapper creates a new RetentionMapper instance.
func NewRetentionMapper() *RetentionMapper {
	return &RetentionMapper{}
}

// MapToRetentionData converts a CreateRetentionRequest to an InputRetentionData domain model.
func (m *RetentionMapper) MapToRetentionData(req *structs.CreateRetentionRequest, client *dte.IssuerDTE) (*retention_models.InputRetentionData, error) {
	if req == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Request")
	}
	if req.Summary == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Request->Summary")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Issuer")
	}

	items, err := retention.MapRetentionItemList(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Items")
	}

	identification, err := common.MapCommonRequestIdentification(constants.ModeloFacturacionPrevio, 2, constants.ComprobanteRetencionElectronico)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Identification")
	}

	if err = validateRetentionReceiverRequest(req); err != nil {
		return nil, err
	}

	receiver, err := retention.MapRetentionRequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Receiver")
	}

	summary, err := retention.MapRetentionSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Summary")
	}

	result := &retention_models.InputRetentionData{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Receiver:       receiver,
			Identification: identification,
		},
		RetentionItems:   items,
		RetentionSummary: summary,
	}

	if err = mapRetentionOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateRetentionReceiverRequest validates the receiver section of a retention request.
func validateRetentionReceiverRequest(req *structs.CreateRetentionRequest) error {
	if req.Receiver == nil {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver")
	}
	if req.Receiver.DocumentType == nil {
		return dte_errors.NewValidationError("InvalidField", "Request->Receiver->DocumentType")
	}
	if req.Receiver.DocumentNumber == nil {
		return dte_errors.NewValidationError("InvalidField", "Request->Receiver->DocumentNumber")
	}
	return nil
}

// mapRetentionOptionalFields maps the optional fields of the retention request into the result model.
func mapRetentionOptionalFields(req *structs.CreateRetentionRequest, result *retention_models.InputRetentionData) error {
	if req.Appendixes != nil {
		appendixes, err := common.MapCommonRequestAppendix(req.Appendixes)
		if err != nil {
			return shared_error.NewFormattedGeneralServiceWithError("RetentionMapper", "MapToRetentionData", err, "ErrorMapping", "Retention->Appendixes")
		}
		result.Appendixes = appendixes
	}

	return nil
}

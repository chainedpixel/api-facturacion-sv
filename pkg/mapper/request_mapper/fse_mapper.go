package request_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/fse"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// FSEMapper maps Factura de Sujeto Excluido creation requests to the FSE domain model.
type FSEMapper struct{}

// NewFSEMapper creates a new FSEMapper instance.
func NewFSEMapper() *FSEMapper {
	return &FSEMapper{}
}

// MapToFSEData converts a CreateFSERequest to an FSEData domain model.
func (m *FSEMapper) MapToFSEData(req *structs.CreateFSERequest, client *dte.IssuerDTE) (*fse_models.FSEData, error) {
	if err := validateFSERequest(req); err != nil {
		return nil, err
	}

	items, err := fse.MapFSEItems(req.Items)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("FSEMapper", "MapToFSEData", err, "ErrorMapping", "FSE->Items")
	}

	receiver, err := fse.MapFSERequestReceiver(req.Receiver)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("FSEMapper", "MapToFSEData", err, "ErrorMapping", "FSE->Receiver")
	}

	summary, err := fse.MapFSERequestSummary(req.Summary)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("FSEMapper", "MapToFSEData", err, "ErrorMapping", "FSE->Summary")
	}

	identification, err := common.MapCommonRequestIdentification(1, 1, constants.FacturaSujetoExcluidoElectronica)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("FSEMapper", "MapToFSEData", err, "ErrorMapping", "FSE->Identification")
	}

	issuer, err := common.MapCommonIssuer(client)
	if err != nil {
		return nil, shared_error.NewFormattedGeneralServiceWithError("FSEMapper", "MapToFSEData", err, "ErrorMapping", "FSE->Issuer")
	}

	result := &fse_models.FSEData{
		InputDataCommon: &models.InputDataCommon{
			Issuer:         issuer,
			Identification: identification,
		},
		Items:       items,
		FSESummary:  summary,
		FSEReceiver: receiver,
	}

	if err = mapFSEOptionalFields(req, result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateFSERequest validates the required fields in the FSE creation request.
func validateFSERequest(req *structs.CreateFSERequest) error {
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
	if req.Receiver.DocumentType == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver->DocumentType")
	}
	if req.Receiver.DocumentNumber == "" {
		return dte_errors.NewValidationError("RequiredField", "Request->Receiver->DocumentNumber")
	}
	return nil
}

// mapFSEOptionalFields maps the optional fields of the FSE request into the result model.
func mapFSEOptionalFields(req *structs.CreateFSERequest, result *fse_models.FSEData) error {
	if req.Extension != nil {
		extension, err := common.MapCommonRequestExtension(req.Extension)
		if err != nil {
			return err
		}
		result.Extension = extension
	}

	if len(req.Appendixes) > 0 {
		appendixes, err := common.MapCommonRequestAppendix(req.Appendixes)
		if err != nil {
			return err
		}
		result.Appendixes = appendixes
	}

	return nil
}

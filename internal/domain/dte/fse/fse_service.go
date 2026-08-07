package fse

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	businessValidator "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/validator"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type fseService struct {
	validator        *validator.FSERulesValidator
	seqNumberManager dte_documents.SequentialNumberManager
}

// NewFSEService Creates a new electronic excluded-subject invoice service.
func NewFSEService(seqNumberManager dte_documents.SequentialNumberManager) ports.DTEService {
	return &fseService{
		validator:        validator.NewFSERulesValidator(nil),
		seqNumberManager: seqNumberManager,
	}
}

// Create Creates a new FSE based on the provided data.
func (s *fseService) Create(ctx context.Context, input interface{}, branchID uint) (interface{}, error) {
	data := input.(*fse_models.FSEData)

	if data.FSEReceiver == nil {
		return nil, shared_error.NewFormattedGeneralServiceError(
			"FSEService", "Create", "ReceiverIsRequired",
		)
	}

	baseDoc := createBaseFSEDocument(data)

	fse := &fse_models.FSEModel{
		DTEDocument: baseDoc,
		FSEItems:    data.Items,
		FSESummary:  *data.FSESummary,
		FSEReceiver: *data.FSEReceiver,
	}

	if err := s.validate(fse); err != nil {
		logs.Error("Failed to validate FSE document basic validation", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := businessValidator.ValidateDTEDocument(fse); err != nil {
		logs.Error("Failed to validate FSE document generic validations", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if err := s.generateCodeAndIdentifiers(ctx, fse, branchID); err != nil {
		return nil, err
	}

	return fse, nil
}

// validate performs the FSE-specific validations
func (s *fseService) validate(fse *fse_models.FSEModel) error {

	return nil
}

// generateCodeAndIdentifiers generates the unique codes and required identifiers
func (s *fseService) generateCodeAndIdentifiers(ctx context.Context, fse *fse_models.FSEModel, branchID uint) error {
	if err := s.generateControlNumber(ctx, fse, branchID); err != nil {
		return err
	}
	return fse.Identification.GenerateCode()
}

// generateControlNumber generates the sequential control number
func (s *fseService) generateControlNumber(ctx context.Context, fse *fse_models.FSEModel, branchID uint) error {
	establishmentCode := fse.Issuer.GetEstablishmentCode()
	posCode := fse.Issuer.GetPOSCode()

	controlNumber, err := s.seqNumberManager.GetNextControlNumber(
		ctx,
		constants.FacturaSujetoExcluidoElectronica,
		branchID,
		posCode,
		establishmentCode)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"FSEService",
			"GenerateControlNumber",
			err,
			"FailedToGenerateControlNumber",
		)
	}

	err = fse.Identification.SetControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewFormattedGeneralServiceWithError(
			"FSEService",
			"GenerateControlNumber",
			err,
			"FailedToSetControlNumber",
		)
	}
	return nil
}

// createBaseFSEDocument creates the base structure of the FSE document
func createBaseFSEDocument(data *fse_models.FSEData) *models.DTEDocument {
	baseDoc := &models.DTEDocument{
		Identification: data.Identification,
		Issuer:         data.Issuer,
		Receiver:       data.FSEReceiver.Receiver,
		Items:          convertFSEItemsToInterface(data.Items),
		Summary:        data.FSESummary.Summary,
	}

	if data.Extension != nil {
		baseDoc.Extension = data.Extension
	}

	if data.Appendixes != nil && len(data.Appendixes) > 0 {
		appendixes := make([]interfaces.Appendix, len(data.Appendixes))
		for i, app := range data.Appendixes {
			appendixes[i] = &app
		}
		baseDoc.Appendix = appendixes
	}

	return baseDoc
}

// convertFSEItemsToInterface converts FSE items to the generic item interface
func convertFSEItemsToInterface(fseItems []fse_models.FSEItem) []interfaces.Item {
	items := make([]interfaces.Item, len(fseItems))
	for i, fseItem := range fseItems {
		items[i] = fseItem.Item
	}
	return items
}

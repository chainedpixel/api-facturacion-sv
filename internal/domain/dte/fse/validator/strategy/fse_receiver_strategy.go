package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type FSEReceiverStrategy struct {
	Document *fse_models.FSEModel
}

func NewFSEReceiverStrategy(document *fse_models.FSEModel) *FSEReceiverStrategy {
	return &FSEReceiverStrategy{
		Document: document,
	}
}

func (v *FSEReceiverStrategy) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	if v.Document == nil {
		return dte_errors.NewDTEErrorSimple("FSEReceiverInvalidDocument")
	}

	receiver := v.Document.FSEReceiver

	if err := v.validateRequiredFields(receiver); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	if err := v.validateDocumentType(receiver); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := v.validateDocumentNumber(receiver); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := v.validateOptionalFields(receiver); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

func (v *FSEReceiverStrategy) validateRequiredFields(receiver fse_models.FSEReceiver) []*dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	if !receiver.DocumentType.IsForReception() {
		logs.Error("FSEReceiverStrategy: DocumentType is invalid or missing", map[string]interface{}{
			"documentType": receiver.DocumentType.GetValue(),
		})
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverRequiredDocumentType"))
	}

	if !receiver.DocumentNumber.IsValid() {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverRequiredDocumentNumber"))
	}

	if receiver.Name == nil || *receiver.Name == "" {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverRequiredName"))
	}

	if receiver.Address == nil {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverRequiredAddress"))
	}

	return errors
}

func (v *FSEReceiverStrategy) validateDocumentType(receiver fse_models.FSEReceiver) *dte_errors.DTEError {
	if !receiver.DocumentType.IsValid() {
		return nil
	}

	validTypes := map[string]bool{
		constants.NIT:             true,
		constants.DUI:             true,
		constants.CarnetResidente: true,
		constants.Pasaporte:       true,
		constants.OtroDocumento:   true,
	}

	documentTypeValue := receiver.DocumentType.GetValue()
	if !validTypes[documentTypeValue] {
		return dte_errors.NewDTEErrorSimple("FSEReceiverInvalidDocumentType")
	}

	return nil
}

func (v *FSEReceiverStrategy) validateDocumentNumber(receiver fse_models.FSEReceiver) *dte_errors.DTEError {
	if !receiver.DocumentNumber.IsValid() || !receiver.DocumentType.IsValid() {
		return nil
	}

	documentType := receiver.DocumentType.GetValue()
	documentNumber := receiver.DocumentNumber.GetValue()

	switch documentType {
	case constants.NIT:
		if len(documentNumber) != 9 && len(documentNumber) != 14 {
			return dte_errors.NewDTEErrorSimple("FSEReceiverInvalidNIT")
		}

	case constants.DUI:
		if len(documentNumber) != 9 {
			return dte_errors.NewDTEErrorSimple("FSEReceiverInvalidDUI")
		}

	default:
		if len(documentNumber) > 20 {
			return dte_errors.NewDTEErrorSimple("FSEReceiverInvalidDocumentLength")
		}
	}

	return nil
}

func (v *FSEReceiverStrategy) validateOptionalFields(receiver fse_models.FSEReceiver) []*dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	if receiver.ActivityCode != nil && !receiver.ActivityCode.IsValid() {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverInvalidActivityCode"))
	}

	if receiver.ActivityDescription != nil && len(*receiver.ActivityDescription) > 150 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSEReceiverInvalidActivityDescription"))
	}

	return errors
}

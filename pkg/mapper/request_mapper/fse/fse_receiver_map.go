package fse

import (
	"strings"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapFSERequestReceiver(receiverReq *structs.FSEReceiverRequest) (*fse_models.FSEReceiver, error) {
	baseReceiver, err := common.MapCommonRequestReceiver(&structs.ReceiverRequest{
		Name:           receiverReq.Name,
		CommercialName: receiverReq.CommercialName,
		Address:        receiverReq.Address,
		Phone:          receiverReq.Phone,
		Email:          receiverReq.Email,
		NIT:            receiverReq.NIT,
		NRC:            receiverReq.NRC,
		ActivityDesc:   receiverReq.ActivityDescription,
		DocumentType:   &receiverReq.DocumentType,
		DocumentNumber: &receiverReq.DocumentNumber,
	})
	if err != nil {
		return nil, err
	}

	documentType, err := document.NewDTETypeForReceiver(receiverReq.DocumentType)
	if err != nil {
		return nil, err
	}

	documentNumber, err := identification.NewDocumentNumber(receiverReq.DocumentNumber, receiverReq.DocumentType)
	if err != nil {
		return nil, err
	}

	var activityCode *identification.ActivityCode
	if receiverReq.ActivityCode != nil && *receiverReq.ActivityCode != "" {
		ac, err := identification.NewActivityCode(*receiverReq.ActivityCode)
		if err != nil {
			return nil, err
		}
		activityCode = ac
	}

	if err := validateFSEReceiver(receiverReq); err != nil {
		return nil, err
	}

	return fse_models.NewFSEReceiver(
		baseReceiver,
		*documentType,
		*documentNumber,
		activityCode,
		receiverReq.ActivityDescription,
	), nil
}

func validateFSEReceiver(receiverReq *structs.FSEReceiverRequest) error {
	if receiverReq.DocumentType == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentType")
	}

	if receiverReq.DocumentNumber == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentNumber")
	}

	if receiverReq.Name == nil || *receiverReq.Name == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Name")
	}

	validDocumentTypes := map[string]bool{
		constants.NIT:             true,
		constants.DUI:             true,
		constants.CarnetResidente: true,
		constants.Pasaporte:       true,
		constants.OtroDocumento:   true,
	}

	if !validDocumentTypes[receiverReq.DocumentType] {
		return dte_errors.NewValidationError("InvalidField", "Receiver->DocumentType")
	}

	cleanDocNumber := strings.ReplaceAll(receiverReq.DocumentNumber, "-", "")

	switch receiverReq.DocumentType {
	case constants.NIT:
		if len(cleanDocNumber) != 9 && len(cleanDocNumber) != 14 {
			return dte_errors.NewValidationError("InvalidField", "Receiver->DocumentNumber->NIT")
		}
	case constants.DUI:
		if len(cleanDocNumber) != 9 {
			return dte_errors.NewValidationError("InvalidField", "Receiver->DocumentNumber->DUI")
		}
	default:
		if len(receiverReq.DocumentNumber) > 20 {
			return dte_errors.NewValidationError("InvalidField", "Receiver->DocumentNumber->Length")
		}
	}

	if receiverReq.ActivityDescription != nil && len(*receiverReq.ActivityDescription) > 150 {
		return dte_errors.NewValidationError("InvalidField", "Receiver->ActivityDescription->Length")
	}

	return nil
}

package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapRetentionRequestReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	var err error
	if receiver == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Receiver")
	}

	if err = validateRequiredFields(receiver); err != nil {
		return nil, err
	}

	docType, err := document.NewDTETypeForReceiver(*receiver.DocumentType)
	if err != nil {
		return nil, err
	}

	docNumber, err := identification.NewDocumentNumber(*receiver.DocumentNumber, *receiver.DocumentType)
	if err != nil {
		return nil, err
	}

	activityCode, err := identification.NewActivityCode(*receiver.ActivityCode)
	if err != nil {
		return nil, err
	}

	address, err := common.MapCommonRequestAddress(*receiver.Address)
	if err != nil {
		return nil, err
	}

	email, err := base.NewEmail(*receiver.Email)
	if err != nil {
		return nil, err
	}

	phone := base.NewValidatedPhone("")
	if receiver.Phone != nil {
		phone, err = base.NewPhone(*receiver.Phone)
		if err != nil {
			return nil, err
		}
	}

	ncr := identification.NewValidatedNRC("")
	if receiver.NRC != nil {
		ncr, err = identification.NewNRC(*receiver.NRC)
		if err != nil {
			return nil, err
		}
	}

	return &models.Receiver{
		DocumentType:        docType,
		DocumentNumber:      docNumber,
		Name:                receiver.Name,
		Email:               email,
		NRC:                 ncr,
		Address:             address,
		Phone:               phone,
		ActivityCode:        activityCode,
		ActivityDescription: receiver.ActivityDesc,
		CommercialName:      receiver.CommercialName,
	}, nil
}

func validateRequiredFields(receiver *structs.ReceiverRequest) error {
	if receiver.Name == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Name")
	}

	if receiver.Email == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Email")
	}

	if receiver.Address == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Address")
	}

	if receiver.DocumentNumber == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentNumber")
	}

	if receiver.DocumentType == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentType")
	}

	if receiver.ActivityCode == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityCode")
	}

	if receiver.ActivityDesc == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityDesc")
	}

	return nil
}

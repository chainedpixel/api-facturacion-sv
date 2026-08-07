package ccf

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapCCFRequestReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	if receiver == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "receiver")
	}

	if err := validateRequiredFields(receiver); err != nil {
		return nil, err
	}

	var err error
	docType := document.NewValidatedDTEType("")
	if receiver.DocumentType != nil {
		docType, err = document.NewDTETypeForReceiver(*receiver.DocumentType)
		if err != nil {
			return nil, err
		}
	}

	docNumber := identification.NewValidatedDocumentNumber("")
	if receiver.DocumentNumber != nil {
		docNumber, err = identification.NewDocumentNumber(*receiver.DocumentNumber, *receiver.DocumentType)
		if err != nil {
			return nil, err
		}
	}

	phone := base.NewValidatedPhone("")
	if receiver.Phone != nil {
		phone, err = base.NewPhone(*receiver.Phone)
		if err != nil {
			return nil, err
		}
	}

	email := base.NewValidatedEmail("")
	if receiver.Email != nil {
		email, err = base.NewEmail(*receiver.Email)
		if err != nil {
			return nil, err
		}
	}

	activityCode, err := identification.NewActivityCode(*receiver.ActivityCode)
	if err != nil {
		return nil, err
	}

	nit, err := identification.NewNIT(*receiver.NIT)
	if err != nil {
		return nil, err
	}

	address, err := common.MapCommonRequestAddress(*receiver.Address)
	if err != nil {
		return nil, err
	}

	ncr, err := identification.NewNRC(*receiver.NRC)
	if err != nil {
		return nil, err
	}

	return &models.Receiver{
		DocumentType:        docType,
		DocumentNumber:      docNumber,
		Name:                receiver.Name,
		Email:               email,
		NRC:                 ncr,
		NIT:                 nit,
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

	if receiver.Address == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Address")
	}

	if receiver.NIT == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->NIT")
	}

	if receiver.NRC == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->NRC")
	}

	if receiver.ActivityCode == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityCode")
	}

	if receiver.ActivityDesc == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityDesc")
	}

	return nil
}

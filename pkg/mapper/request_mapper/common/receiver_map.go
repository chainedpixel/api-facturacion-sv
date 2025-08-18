package common

import (
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/value_objects/base"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/value_objects/document"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/value_objects/identification"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/value_objects/location"
	"github.com/MarlonG1/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// MapCommonRequestReceiver mapea un receptor común a un modelo de receptor -> Origen: Request
func MapCommonRequestReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	var nrc *identification.NRC
	var address *models.Address
	var activityDesc *string
	var err error

	if receiver == nil {
		return nil, nil
	}

	if receiver.NRC != nil {
		nrc, err = identification.NewNRC(*receiver.NRC)
		if err != nil {
			return nil, err
		}
	}

	if receiver.Address != nil {
		address, err = MapCommonRequestAddress(*receiver.Address)
		if err != nil {
			return nil, err
		}
	} else {
		address = &models.Address{
			Department:   location.Department{},
			Municipality: location.Municipality{},
			Complement:   location.Address{},
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

	activityCode := identification.NewValidatedActivityCode("")
	if receiver.ActivityCode != nil {
		activityCode, err = identification.NewActivityCode(*receiver.ActivityCode)
		if err != nil {
			return nil, err
		}
	}

	if receiver.ActivityDesc != nil {
		activityDesc = receiver.ActivityDesc
	}

	if receiver.DocumentType != nil && *receiver.DocumentType == constants.DUI && receiver.NRC != nil {
		return nil, dte_errors.NewValidationError("InvalidFieldValue", "Request->Receiver->NRC")
	}

	return &models.Receiver{
		DocumentType:        docType,
		DocumentNumber:      docNumber,
		Name:                receiver.Name,
		Email:               email,
		NRC:                 nrc,
		Address:             address,
		Phone:               phone,
		ActivityCode:        activityCode,
		ActivityDescription: activityDesc,
	}, nil
}

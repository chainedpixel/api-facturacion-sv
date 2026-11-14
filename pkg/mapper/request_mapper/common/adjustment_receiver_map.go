package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// MapAdjustmentRequestReceiver mapea el receptor de una NCE o NDE. El request identifica al receptor por NIT
// (se compara con el del CCF relacionado) y a Hacienda se envia como tipo y numero de documento:
// Anexo IV, campos 38.3 y 39.6, cuando el documento a ajustar es CCFE solo se permite el codigo 36-NIT
// y el numero debe ser el NIT del receptor del CCFE.
func MapAdjustmentRequestReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	var err error
	if receiver == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Receiver")
	}

	if err = validateAdjustmentReceiverFields(receiver); err != nil {
		return nil, err
	}

	nit, err := identification.NewNIT(*receiver.NIT)
	if err != nil {
		return nil, err
	}

	documentType, err := document.NewDTETypeForReceiver(constants.NIT)
	if err != nil {
		return nil, err
	}
	documentNumber, err := identification.NewDocumentNumber(nit.GetValue(), constants.NIT)
	if err != nil {
		return nil, err
	}

	activityCode, err := identification.NewActivityCode(*receiver.ActivityCode)
	if err != nil {
		return nil, err
	}

	address, err := MapCommonRequestAddress(*receiver.Address)
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
		DocumentType:        documentType,
		DocumentNumber:      documentNumber,
		NIT:                 nit,
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

func validateAdjustmentReceiverFields(receiver *structs.ReceiverRequest) error {
	if receiver.Name == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Name")
	}

	if receiver.Email == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Email")
	}

	if receiver.Address == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Address")
	}

	if receiver.NRC == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->NRC")
	}

	if receiver.NIT == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->NIT")
	}

	if receiver.ActivityCode == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityCode")
	}

	if receiver.ActivityDesc == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityDesc")
	}

	return nil
}

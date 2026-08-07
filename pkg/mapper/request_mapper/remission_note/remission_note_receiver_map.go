package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapRemissionNoteRequestReceiver(receiver *structs.RemissionNoteReceiverRequest) (*remission_note_models.RemissionNoteReceiver, error) {
	if receiver == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Receiver")
	}

	if err := validateRequiredFields(receiver); err != nil {
		return nil, err
	}

	baseReceiver, err := createBaseReceiver(receiver.ReceiverRequest)
	if err != nil {
		return nil, err
	}

	remissionReceiver := &remission_note_models.RemissionNoteReceiver{
		Receiver:   baseReceiver,
		BienTitulo: receiver.BienTitulo,
	}

	return remissionReceiver, nil
}

func createBaseReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	baseReceiver := &models.Receiver{}

	baseReceiver.SetName(receiver.Name)

	if receiver.DocumentType != nil {
		if err := baseReceiver.SetDocumentType(receiver.DocumentType); err != nil {
			return nil, err
		}
	}

	if receiver.DocumentNumber != nil {
		if err := baseReceiver.SetDocumentNumber(receiver.DocumentNumber); err != nil {
			return nil, err
		}
	}

	if receiver.Email != nil {
		if err := baseReceiver.SetEmail(receiver.Email); err != nil {
			return nil, err
		}
	}

	if receiver.Address != nil {
		address, err := common.MapCommonRequestAddress(*receiver.Address)
		if err != nil {
			return nil, err
		}
		baseReceiver.SetAddress(address)
	}

	if receiver.Phone != nil {
		if err := baseReceiver.SetPhone(receiver.Phone); err != nil {
			return nil, err
		}
	}

	if receiver.NRC != nil {
		if err := baseReceiver.SetNRC(receiver.NRC); err != nil {
			return nil, err
		}
	}

	if receiver.NIT != nil {
		if err := baseReceiver.SetNIT(receiver.NIT); err != nil {
			return nil, err
		}
	}

	if receiver.ActivityCode != nil {
		if err := baseReceiver.SetActivityCode(receiver.ActivityCode); err != nil {
			return nil, err
		}
	}

	baseReceiver.SetActivityDescription(receiver.ActivityDesc)
	baseReceiver.SetCommercialName(receiver.CommercialName)

	return baseReceiver, nil
}

func validateRequiredFields(receiver *structs.RemissionNoteReceiverRequest) error {
	if receiver.ReceiverRequest == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->BaseFields")
	}

	if receiver.BienTitulo == nil || *receiver.BienTitulo == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->BienTitulo")
	}

	if len(*receiver.BienTitulo) != 2 {
		return dte_errors.NewValidationError("InvalidLength", "Receiver->BienTitulo must be exactly 2 characters")
	}

	if receiver.ReceiverRequest.Name == nil || *receiver.ReceiverRequest.Name == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Name")
	}

	if receiver.ReceiverRequest.DocumentType == nil || *receiver.ReceiverRequest.DocumentType == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentType")
	}

	if receiver.ReceiverRequest.DocumentNumber == nil || *receiver.ReceiverRequest.DocumentNumber == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->DocumentNumber")
	}

	if receiver.ReceiverRequest.Email == nil || *receiver.ReceiverRequest.Email == "" {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Email")
	}

	if receiver.ReceiverRequest.Address == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver->Address")
	}

	if receiver.ReceiverRequest.DocumentType != nil && *receiver.ReceiverRequest.DocumentType == "36" {
		if receiver.ReceiverRequest.NIT == nil || *receiver.ReceiverRequest.NIT == "" {
			return dte_errors.NewValidationError("RequiredField", "Receiver->NIT is required for document type 36")
		}

		if receiver.ReceiverRequest.ActivityCode == nil || *receiver.ReceiverRequest.ActivityCode == "" {
			return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityCode is required for document type 36")
		}

		if receiver.ReceiverRequest.ActivityDesc == nil || *receiver.ReceiverRequest.ActivityDesc == "" {
			return dte_errors.NewValidationError("RequiredField", "Receiver->ActivityDesc is required for document type 36")
		}
	}

	return nil
}

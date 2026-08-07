package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// MapCommonRequestExtension maps a common extension to an extension model -> Source: Request
func MapCommonRequestExtension(extension *structs.ExtensionRequest) (*models.Extension, error) {
	var observation *document.Observation
	if extension == nil {
		return nil, nil
	}
	if err := validateExtensionRequest(extension); err != nil {
		return nil, err
	}

	deliveryName, err := document.NewDeliveryName(extension.DeliveryName)
	if err != nil {
		return nil, err
	}

	deliveryDocument, err := document.NewDeliveryDocument(extension.DeliveryDocument)
	if err != nil {
		return nil, err
	}

	receiverName, err := document.NewDeliveryName(extension.ReceiverName)
	if err != nil {
		return nil, err
	}

	receiverDocument, err := document.NewDeliveryDocument(extension.ReceiverDocument)
	if err != nil {
		return nil, err
	}

	if extension.Observation != nil {
		observation, err = document.NewObservation(*extension.Observation)
	}

	if extension.VehiculePlate != nil {
		value := *extension.VehiculePlate
		if len(value) < 1 || len(value) > 10 {
			return nil, dte_errors.NewValidationError("InvalidLength", "Extension->VehiculePlate", "1-10", value)
		}
	}

	return &models.Extension{
		DeliveryName:     *deliveryName,
		DeliveryDocument: *deliveryDocument,
		ReceiverName:     *receiverName,
		ReceiverDocument: *receiverDocument,
		Observation:      observation,
		VehiculePlate:    extension.VehiculePlate,
	}, nil
}

func validateExtensionRequest(extension *structs.ExtensionRequest) error {
	if extension == nil {
		return nil
	}

	if extension.DeliveryName == "" {
		return dte_errors.NewValidationError("RequiredField", "Extension->DeliveryName")
	}

	if extension.DeliveryDocument == "" {
		return dte_errors.NewValidationError("RequiredField", "Extension->DeliveryDocument")
	}

	if extension.ReceiverName == "" {
		return dte_errors.NewValidationError("RequiredField", "Extension->ReceiverName")
	}

	if extension.ReceiverDocument == "" {
		return dte_errors.NewValidationError("RequiredField", "Extension->ReceiverDocument")
	}

	return nil
}

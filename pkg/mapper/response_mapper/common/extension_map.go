package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseExtension maps an extension to an extension model -> Source: Response
func MapCommonResponseExtension(extension interfaces.Extension) *structs.DTEExtension {
	if extension == nil {
		return nil
	}

	return &structs.DTEExtension{
		NombreEntrega:    extension.GetDeliveryName(),
		DocumentoEntrega: extension.GetDeliveryDocument(),
		NombreRecibe:     extension.GetReceiverName(),
		DocumentoRecibe:  extension.GetReceiverDocument(),
		Observacion:      extension.GetObservation(),
		PlacaVehiculo:    extension.GetVehiculePlate(),
	}
}

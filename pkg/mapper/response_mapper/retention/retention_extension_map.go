package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRetentionResponseExtension maps an extension to an extension model -> Source: Response
func MapRetentionResponseExtension(extension interfaces.Extension) *structs.RetentionExtension {
	if extension == nil {
		return nil
	}

	return &structs.RetentionExtension{
		NombreEntrega:    extension.GetDeliveryName(),
		DocumentoEntrega: extension.GetDeliveryDocument(),
		NombreRecibe:     extension.GetReceiverName(),
		DocumentoRecibe:  extension.GetReceiverDocument(),
		Observacion:      extension.GetObservation(),
	}
}

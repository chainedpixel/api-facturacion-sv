package credit_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapCreditNoteResponseExtension(extension interfaces.Extension) *structs.CreditNoteDTEExtension {
	if extension == nil {
		return nil
	}

	return &structs.CreditNoteDTEExtension{
		NombreEntrega:    extension.GetDeliveryName(),
		DocumentoEntrega: extension.GetDeliveryDocument(),
		NombreRecibe:     extension.GetReceiverName(),
		DocumentoRecibe:  extension.GetReceiverDocument(),
		Observacion:      extension.GetObservation(),
	}
}

package invoice

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapInvoiceResponseReceiver maps a receiver to a receiver model -> Source: Response
func MapInvoiceResponseReceiver(receiver interfaces.Receiver) structs.InvoiceReceiver {
	result := structs.InvoiceReceiver{
		TipoDocumento: receiver.GetDocumentType(),
		NumDocumento:  receiver.GetDocumentNumber(),
		Nombre:        receiver.GetName(),
		Direccion:     addressToPointer(common.MapCommonResponseAddress(receiver.GetAddress())),
		NRC:           receiver.GetNRC(),
		Correo:        receiver.GetEmail(),
		Telefono:      receiver.GetPhone(),
		CodActividad:  receiver.GetActivityCode(),
		DescActividad: receiver.GetActivityDescription(),
	}
	return result
}

func addressToPointer(address structs.DTEAddress) *structs.DTEAddress {
	if address.Departamento == "" && address.Municipio == "" && address.Complemento == "" {
		return nil
	}
	return &address
}

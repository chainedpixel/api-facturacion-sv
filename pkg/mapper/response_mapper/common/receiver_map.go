package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseReceiver maps a receiver to a receiver model -> Source: Response
func MapCommonResponseReceiver(receiver interfaces.Receiver) structs.DTEReceiver {
	result := structs.DTEReceiver{
		TipoDocumento:   receiver.GetDocumentType(),
		NumDocumento:    receiver.GetDocumentNumber(),
		Nombre:          receiver.GetName(),
		NRC:             receiver.GetNRC(),
		NIT:             receiver.GetNIT(),
		Direccion:       addressToPointer(MapCommonResponseAddress(receiver.GetAddress())),
		Correo:          receiver.GetEmail(),
		Telefono:        receiver.GetPhone(),
		CodActividad:    receiver.GetActivityCode(),
		DescActividad:   receiver.GetActivityDescription(),
		NombreComercial: receiver.GetCommercialName(),
	}
	return result
}

func addressToPointer(address structs.DTEAddress) *structs.DTEAddress {
	if address.Departamento == "" && address.Municipio == "" && address.Complemento == "" {
		return nil
	}
	return &address
}

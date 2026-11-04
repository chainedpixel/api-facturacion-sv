package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseReceiver maps the receiver of a CCFE (identified by NIT) -> Source: Response
func MapCommonResponseReceiver(receiver interfaces.Receiver) structs.DTEReceiver {
	return structs.DTEReceiver{
		NIT:             receiver.GetNIT(),
		DTEReceiverBase: mapReceiverBase(receiver),
	}
}

// MapCommonResponseDocumentReceiver maps the receiver of NCE, NDE and CRE (identified by document type and number) -> Source: Response
func MapCommonResponseDocumentReceiver(receiver interfaces.Receiver) structs.DTEDocumentReceiver {
	return structs.DTEDocumentReceiver{
		TipoDocumento:   receiver.GetDocumentType(),
		NumDocumento:    receiver.GetDocumentNumber(),
		DTEReceiverBase: mapReceiverBase(receiver),
	}
}

// mapReceiverBase se encarga de los campos comunes a todos los receptores
func mapReceiverBase(receiver interfaces.Receiver) structs.DTEReceiverBase {
	return structs.DTEReceiverBase{
		Nombre:          receiver.GetName(),
		NRC:             receiver.GetNRC(),
		CodActividad:    receiver.GetActivityCode(),
		DescActividad:   receiver.GetActivityDescription(),
		NombreComercial: receiver.GetCommercialName(),
		Direccion:       addressToPointer(MapCommonResponseAddress(receiver.GetAddress())),
		Telefono:        receiver.GetPhone(),
		Correo:          receiver.GetEmail(),
	}
}

func addressToPointer(address structs.DTEAddress) *structs.DTEAddress {
	if address.Departamento == "" && address.Municipio == "" && address.Complemento == "" {
		return nil
	}
	return &address
}

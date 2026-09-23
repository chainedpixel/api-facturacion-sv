package debit_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapDebitNoteIssuer(issuer interfaces.Issuer) structs.DebitNoteDTEIssuer {
	result := structs.DebitNoteDTEIssuer{
		NIT:           issuer.GetNIT(),
		NRC:           issuer.GetNRC(),
		Nombre:        issuer.GetName(),
		CodActividad:  issuer.GetActivityCode(),
		DescActividad: issuer.GetActivityDescription(),
		Direccion:     common.MapCommonResponseAddress(issuer.GetAddress()),
		Telefono:      issuer.GetPhone(),
		Correo:        issuer.GetEmail(),
	}

	if name := issuer.GetCommercialName(); name != "" {
		result.NombreComercial = &name
	}

	return result
}

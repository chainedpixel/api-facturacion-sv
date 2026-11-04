package fse

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapFSEResponseIssuer maps the specific FSE issuer excluding non-allowed fields
func MapFSEResponseIssuer(issuer interfaces.Issuer) structs.FSEIssuer {
	var codActividad *string
	if actCode := issuer.GetActivityCode(); actCode != "" {
		codActividad = &actCode
	}

	var descActividad *string
	if actDesc := issuer.GetActivityDescription(); actDesc != "" {
		descActividad = &actDesc
	}

	result := structs.FSEIssuer{
		NIT:           issuer.GetNIT(),
		NRC:           issuer.GetNRC(),
		Nombre:        issuer.GetName(),
		CodActividad:  codActividad,
		DescActividad: descActividad,
		Direccion:     common.MapCommonResponseAddress(issuer.GetAddress()),
		Telefono:      issuer.GetPhone(),
		Correo:        issuer.GetEmail(),
		CodEstable:    issuer.GetEstablishmentCode(),
		CodPuntoVenta: issuer.GetPOSCode(),
	}

	return result
}

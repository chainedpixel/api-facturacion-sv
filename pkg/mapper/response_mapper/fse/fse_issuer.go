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
	}

	if code := issuer.GetEstablishmentCode(); code != nil {
		result.CodEstable = code
	}
	if code := issuer.GetEstablishmentMHCode(); code != nil {
		result.CodEstableMH = code
	}
	if code := issuer.GetPOSCode(); code != nil {
		result.CodPuntoVenta = code
	}
	if code := issuer.GetPOSMHCode(); code != nil {
		result.CodPuntoVentaMH = code
	}

	return result
}

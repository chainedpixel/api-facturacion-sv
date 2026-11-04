package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseIssuer maps the issuer of an electronic invoice to an issuer model -> Source: Response
func MapCommonResponseIssuer(issuer interfaces.Issuer) structs.DTEIssuer {
	result := structs.DTEIssuer{
		NIT:           issuer.GetNIT(),
		NRC:           issuer.GetNRC(),
		Nombre:        issuer.GetName(),
		CodActividad:  issuer.GetActivityCode(),
		DescActividad: issuer.GetActivityDescription(),
		Direccion:     MapCommonResponseAddress(issuer.GetAddress()),
		Telefono:      issuer.GetPhone(),
		Correo:        issuer.GetEmail(),
		CodEstable:    issuer.GetEstablishmentCode(),
		CodPuntoVenta: issuer.GetPOSCode(),
	}

	if name := issuer.GetCommercialName(); name != "" {
		result.NombreComercial = &name
	}

	return result
}

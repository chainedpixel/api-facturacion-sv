package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRetentionResponseIssuer maps the issuer for retention response according to MH schema.
func MapRetentionResponseIssuer(issuer interfaces.Issuer) structs.RetentionIssuer {
	result := structs.RetentionIssuer{
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
	if code := issuer.GetEstablishmentCode(); code != nil {
		result.CodEstable = code
	}
	if code := issuer.GetPOSCode(); code != nil {
		result.CodPuntoVenta = code
	}

	return result
}

package response_mapper

import (
	commonModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/invalidation"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func ToMHInvalidation(doc interface{}) *structs.InvalidationResponse {
	if doc == nil {
		return nil
	}

	cast := doc.(*invalidation_models.InvalidationDocument)
	return &structs.InvalidationResponse{
		Identificacion: *MapIdentificationResponse(cast.Identification),
		Emisor:         *MapIssuerResponse(cast.Issuer),
		Documento:      *invalidation.MapInvalidatedDocumentResponse(cast.Document),
		Motivo:         *invalidation.MapInvalidationReasonResponse(cast.Reason),
	}
}

func MapIdentificationResponse(identification *commonModels.Identification) *structs.InvalidationIdentification {
	if identification == nil {
		return nil
	}

	return &structs.InvalidationIdentification{
		Version:          identification.Version.GetValue(),
		Ambiente:         identification.Ambient.GetValue(),
		CodigoGeneracion: identification.GenerationCode.GetValue(),
		FecAnula:         identification.EmissionDate.GetValue().Format("2006-01-02"),
		HorAnula:         identification.EmissionTime.GetValue().Format("15:04:05"),
	}
}

func MapIssuerResponse(issuer *commonModels.Issuer) *structs.InvalidationIssuer {
	if issuer == nil {
		return nil
	}

	return &structs.InvalidationIssuer{
		NIT:                   issuer.NIT.GetValue(),
		Nombre:                issuer.Name,
		TipoEstablecimiento:   issuer.EstablishmentType.GetValue(),
		NombreComercial:       &issuer.CommercialName,
		Telefono:              issuer.Phone.GetValue(),
		Correo:                issuer.Email.GetValue(),
		CodigoEstablecimiento: issuer.EstablishmentCode,
		POSCodigo:             issuer.POSCode,
	}
}

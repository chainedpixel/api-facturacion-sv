package response_mapper

import (
	commonModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/invalidation"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// ToMHInvalidation maps an invalidation domain document to the Hacienda response payload.
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

// MapIdentificationResponse maps identification metadata for an invalidation event.
func MapIdentificationResponse(identification *commonModels.Identification) *structs.InvalidationIdentification {
	if identification == nil {
		return nil
	}

	return &structs.InvalidationIdentification{
		Version:          identification.Version.GetValue(),
		Ambiente:         identification.Ambient.GetValue(),
		CodigoGeneracion: identification.GenerationCode.GetValue(),
		FecEmi:           identification.EmissionDate.GetValue().Format("2006-01-02"),
		HorEmi:           identification.EmissionTime.GetValue().Format("15:04:05"),
		Fusion:           nil,
	}
}

// MapIssuerResponse maps issuer data for an invalidation event.
func MapIssuerResponse(issuer *commonModels.Issuer) *structs.InvalidationIssuer {
	if issuer == nil {
		return nil
	}

	var codEstableMH string
	if issuer.EstablishmentMHCode != nil {
		codEstableMH = *issuer.EstablishmentMHCode
	}

	var codPuntoVentaMH string
	if issuer.POSMHCode != nil {
		codPuntoVentaMH = *issuer.POSMHCode
	}

	return &structs.InvalidationIssuer{
		NIT:             issuer.NIT.GetValue(),
		Nombre:          issuer.Name,
		CodEstableMH:    codEstableMH,
		CodEstable:      issuer.EstablishmentCode,
		CodPuntoVentaMH: codPuntoVentaMH,
		CodPuntoVenta:   issuer.POSCode,
		Telefono:        issuer.Phone.GetValue(),
		Correo:          issuer.Email.GetValue(),
	}
}

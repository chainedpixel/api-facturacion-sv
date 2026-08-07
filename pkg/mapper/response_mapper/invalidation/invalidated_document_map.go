package invalidation

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapInvalidatedDocumentResponse(doc *invalidation_models.InvalidatedDocument) *structs.DocumentResponse {
	if doc == nil {
		return nil
	}

	result := &structs.DocumentResponse{
		TipoDte:          doc.Type.GetValue(),
		CodigoGeneracion: doc.GenerationCode.GetValue(),
		SelloRecibido:    doc.ReceptionStamp,
		NumeroControl:    doc.ControlNumber.GetValue(),
		FecEmi:           doc.EmissionDate.GetValue().Format("2006-01-02"),
		MontoIva:         doc.IVAAmount.GetValue(),
		TipoDocumento:    utils.ToStringPointer(doc.DocumentType.GetValue()),
		NumDocumento:     utils.ToStringPointer(doc.DocumentNumber.GetValue()),
		Correo:           utils.ToStringPointer(doc.Email.GetValue()),
		Telefono:         utils.ToStringPointer(doc.Phone.GetValue()),
	}

	if doc.Name != nil && *doc.Name != "" {
		result.Nombre = doc.Name
	}

	if doc.ReplacementCode != nil {
		result.CodigoGeneracionR = utils.ToStringPointer(doc.ReplacementCode.GetValue())
	}

	return result
}

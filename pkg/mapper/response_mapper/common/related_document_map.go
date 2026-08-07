package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseRelatedDocuments maps the related documents of an electronic invoice to a related document model -> Source: Response
func MapCommonResponseRelatedDocuments(docs []interfaces.RelatedDocument) []structs.DTERelatedDocument {
	result := make([]structs.DTERelatedDocument, len(docs))
	for i, doc := range docs {
		result[i] = structs.DTERelatedDocument{
			TipoDocumento:   doc.GetDocumentType(),
			TipoGeneracion:  doc.GetGenerationType(),
			NumeroDocumento: doc.GetDocumentNumber(),
			FechaEmision:    doc.GetEmissionDate().Format("2006-01-02"),
		}
	}
	return result
}

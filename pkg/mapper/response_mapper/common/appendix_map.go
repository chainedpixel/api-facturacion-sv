package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseAppendix maps the appendixes of a document
func MapCommonResponseAppendix(request []interfaces.Appendix) []structs.DTEApendice {
	result := make([]structs.DTEApendice, len(request))
	for i, appendix := range request {
		result[i] = structs.DTEApendice{
			Campo:    appendix.GetField(),
			Etiqueta: appendix.GetLabel(),
			Valor:    appendix.GetValue(),
		}
	}

	return result
}

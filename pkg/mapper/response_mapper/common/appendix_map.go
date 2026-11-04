package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseAppendix maps the appendixes of a document.
// Se devuelve nil cuando no hay elementos: los schemas exigen null o un arreglo con minItems 1,
// por lo que un arreglo vacio seria rechazado.
func MapCommonResponseAppendix(request []interfaces.Appendix) []structs.DTEApendice {
	if len(request) == 0 {
		return nil
	}

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

package common

import (
	"math"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
	"github.com/shopspring/decimal"
)

// MapTaxes maps the taxes of an invoice
// MapTaxes mapea los tributos del resumen. Sin tributos devuelve nil (se envia null): Anexo IV, campos 136-138.
func MapTaxes(taxes []interfaces.Tax) []structs.DTETax {
	if len(taxes) == 0 {
		return nil
	}
	result := make([]structs.DTETax, 0, len(taxes))

	for _, tax := range taxes {
		valor := decimal.NewFromFloat(tax.GetValue())

		roundedValue := valor.Round(2)
		floatValue := roundedValue.InexactFloat64()

		if floatValue == math.Floor(floatValue) {
			floatValue = math.Round(floatValue*100) / 100
		}

		result = append(result, structs.DTETax{
			Codigo:      tax.GetCode(),
			Descripcion: tax.GetDescription(),
			Valor:       floatValue,
		})
	}

	return result
}

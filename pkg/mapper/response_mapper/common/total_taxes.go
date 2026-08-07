package common

import (
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/shopspring/decimal"
)

// MapTaxes maps the taxes of an invoice
func MapTaxes(taxes []interfaces.Tax) []structs.DTETax {
	result := make([]structs.DTETax, 0)

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

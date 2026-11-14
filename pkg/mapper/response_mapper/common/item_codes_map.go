package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
)

// MapItemCodes devuelve el codigo del producto (codigo) y el codigo del tributo (codTributo) de un item.
// codigo va null si no se informo; codTributo solo aplica a items tipo 4 (impuesto sujeto a calculo de IVA).
func MapItemCodes(item interfaces.Item) (code *string, taxCode *string) {
	if value := item.GetItemCode(); value != "" {
		code = &value
	}
	if value := item.GetTaxCode(); value != "" && item.GetType() == constants.Impuesto {
		taxCode = &value
	}
	return code, taxCode
}

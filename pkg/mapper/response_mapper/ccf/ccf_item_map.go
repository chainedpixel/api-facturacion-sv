package ccf

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapCCFResponseItem(items []ccf_models.CreditItem) []structs.DTEItem {
	result := make([]structs.DTEItem, len(items))
	for i, item := range items {
		result[i] = common.MapCommonItems(item)
		result[i].VentaNoSuj = item.NonSubjectSale.GetValue()
		result[i].VentaExenta = item.ExemptSale.GetValue()
		result[i].VentaGravada = item.TaxedSale.GetValue()
		result[i].PSV = item.SuggestedPrice.GetValue()
		result[i].NoGravado = item.NonTaxed.GetValue()
	}
	return result
}

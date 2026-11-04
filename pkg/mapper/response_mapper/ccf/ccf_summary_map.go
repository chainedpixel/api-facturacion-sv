package ccf

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapCCFResponseSummary(summary ccf_models.CreditSummary) *structs.DTESummary {
	result := common.MapCommonResponseSummary(summary)
	result.DescuGravada = summary.TaxedDiscount.GetValue()
	result.IvaRete = summary.IVARetention.GetValue()
	result.IvaPerci = summary.IVAPerception.GetValue()
	result.SaldoFavor = summary.BalanceInFavor.GetValue()
	return result
}

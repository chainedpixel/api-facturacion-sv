package retention_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/shopspring/decimal"
)

type RetentionModel struct {
	*models.DTEDocument
	RetentionItems   []RetentionItem
	RetentionSummary *RetentionSummary
}

func (r *RetentionModel) GetTotalByItems() (decimal.Decimal, decimal.Decimal) {
	var totalSubjectRetention, totalIVARetention decimal.Decimal

	for _, item := range r.RetentionItems {
		totalSubjectRetention = totalSubjectRetention.Add(item.RetentionAmount.GetValueAsDecimal())
		totalIVARetention = totalIVARetention.Add(item.RetentionIVA.GetValueAsDecimal())
	}

	return totalSubjectRetention, totalIVARetention
}

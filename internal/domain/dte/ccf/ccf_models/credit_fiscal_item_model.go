package ccf_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type CreditItem struct {
	*models.Item
	NonSubjectSale financial.Amount
	ExemptSale     financial.Amount
	TaxedSale      financial.Amount
	SuggestedPrice financial.Amount
	NonTaxed       financial.Amount
}

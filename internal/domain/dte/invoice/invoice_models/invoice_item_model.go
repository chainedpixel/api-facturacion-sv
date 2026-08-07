package invoice_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

// InvoiceItem represents an item of the electronic sales invoice
type InvoiceItem struct {
	*models.Item
	NonSubjectSale financial.Amount `json:"nonSubjectSale"`
	ExemptSale     financial.Amount `json:"exemptSale"`
	TaxedSale      financial.Amount `json:"taxedSale"`
	SuggestedPrice financial.Amount `json:"suggestedPrice"`
	NonTaxed       financial.Amount `json:"nonTaxed"`
	IVAItem        financial.Amount `json:"ivaItem"`
}

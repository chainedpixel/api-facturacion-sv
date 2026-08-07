package debit_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type DebitNoteItem struct {
	*models.Item
	NonSubjectSale financial.Amount `json:"nonSubjectSale"`
	ExemptSale     financial.Amount `json:"exemptSale"`
	TaxedSale      financial.Amount `json:"taxedSale"`
}

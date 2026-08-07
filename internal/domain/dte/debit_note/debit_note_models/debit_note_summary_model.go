package debit_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type DebitNoteSummary struct {
	*models.Summary
	TaxedDiscount   financial.Amount `json:"taxedDiscount"`
	IVAPerception   financial.Amount `json:"ivaPerception"`
	IVARetention    financial.Amount `json:"ivaRetention"`
	IncomeRetention financial.Amount `json:"incomeRetention"`
}

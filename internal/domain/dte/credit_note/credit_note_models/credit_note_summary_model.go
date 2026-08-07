package credit_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type CreditNoteSummary struct {
	*models.Summary
	TaxedDiscount   financial.Amount
	IVAPerception   financial.Amount
	IVARetention    financial.Amount
	IncomeRetention financial.Amount
}

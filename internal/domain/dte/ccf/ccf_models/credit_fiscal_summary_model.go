package ccf_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type CreditSummary struct {
	*models.Summary
	TaxedDiscount           financial.Amount
	IVAPerception           financial.Amount
	IVARetention            financial.Amount
	BalanceInFavor          financial.Amount
	IncomeRetention         financial.Amount
	ElectronicPaymentNumber *string
}

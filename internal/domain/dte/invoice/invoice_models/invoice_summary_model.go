package invoice_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type InvoiceSummary struct {
	*models.Summary
	TaxedDiscount           financial.Amount `json:"taxedDiscount"`
	IVARetention            financial.Amount `json:"IVARetention"`
	IncomeRetention         financial.Amount `json:"incomeRetention"`
	TotalIva                financial.Amount `json:"totalIva"`
	BalanceInFavor          financial.Amount `json:"balanceInFavor"`
	ElectronicPaymentNumber *string          `json:"electronicPaymentNumber,omitempty"`
}

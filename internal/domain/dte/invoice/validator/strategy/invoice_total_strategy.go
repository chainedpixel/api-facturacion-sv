package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
)

type InvoiceTotalsStrategy struct {
	Document *invoice_models.ElectronicInvoice
}

func (s *InvoiceTotalsStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	validations := []func() *dte_errors.DTEError{
		s.validateSubTotal,
		s.validateDiscounts,
		s.validateTotalOperation,
	}

	for _, validate := range validations {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// validateSubTotal validates the subtotal of the electronic invoice
func (s *InvoiceTotalsStrategy) validateSubTotal() *dte_errors.DTEError {
	expectedSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue())

	if !s.compareTotalsWithTolerance(expectedSubTotal, actualSubTotal, 0.0001) {
		return dte_errors.NewDTEErrorSimple("InvalidSubTotal",
			actualSubTotal.InexactFloat64(),
			expectedSubTotal.InexactFloat64())
	}
	return nil
}

// validateDiscounts validates the discounts of the electronic invoice
func (s *InvoiceTotalsStrategy) validateDiscounts() *dte_errors.DTEError {
	totalDiscount := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalDiscount.GetValue())
	taxedDiscount := decimal.NewFromFloat(s.Document.InvoiceSummary.TaxedDiscount.GetValue())
	exemptDiscount := decimal.NewFromFloat(s.Document.InvoiceSummary.ExemptDiscount.GetValue())
	nonSubjectDiscount := decimal.NewFromFloat(s.Document.InvoiceSummary.NonSubjectDiscount.GetValue())

	if totalDiscount.LessThan(decimal.Zero) {
		return dte_errors.NewDTEErrorSimple("NegativeDiscount", "TotalDiscount", totalDiscount)
	}
	if taxedDiscount.LessThan(decimal.Zero) {
		return dte_errors.NewDTEErrorSimple("NegativeDiscount", "TaxedDiscount", taxedDiscount)
	}
	if exemptDiscount.LessThan(decimal.Zero) {
		return dte_errors.NewDTEErrorSimple("NegativeDiscount", "ExemptDiscount", exemptDiscount)
	}
	if nonSubjectDiscount.LessThan(decimal.Zero) {
		return dte_errors.NewDTEErrorSimple("NegativeDiscount", "NonSubjectDiscount", nonSubjectDiscount)
	}

	totalTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())
	totalExempt := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())
	totalNonSubject := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())

	if taxedDiscount.GreaterThan(totalTaxed) {
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"TaxedDiscount",
			taxedDiscount.InexactFloat64(),
			totalTaxed.InexactFloat64())
	}

	if exemptDiscount.GreaterThan(totalExempt) {
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"ExemptDiscount",
			exemptDiscount.InexactFloat64(),
			totalExempt.InexactFloat64())
	}

	if nonSubjectDiscount.GreaterThan(totalNonSubject) {
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"NonSubjectDiscount",
			nonSubjectDiscount.InexactFloat64(),
			totalNonSubject.InexactFloat64())
	}

	var itemsDiscountSum decimal.Decimal
	for _, item := range s.Document.InvoiceItems {
		itemDiscount := decimal.NewFromFloat(item.GetDiscount()).Mul(decimal.NewFromFloat(item.GetQuantity()).Mul(decimal.NewFromFloat(item.GetUnitPrice())))
		itemsDiscountSum = itemsDiscountSum.Add(itemDiscount.Div(decimal.NewFromInt(100)))
	}

	return nil
}

// validateTotalOperation validates the total operation amount of the electronic invoice
func (s *InvoiceTotalsStrategy) validateTotalOperation() *dte_errors.DTEError {
	expectedTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue())

	for _, tax := range s.Document.InvoiceSummary.TotalTaxes {
		taxValue := decimal.NewFromFloat(tax.GetValue())
		expectedTotal = expectedTotal.Add(taxValue)
	}

	actualTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalOperation.GetValue())

	if !s.compareTotalsWithTolerance(expectedTotal, actualTotal, 0.0001) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalOperation",
			actualTotal.InexactFloat64(),
			expectedTotal.InexactFloat64())
	}
	return nil
}

// calculateExpectedTotal calculates the expected total of the operation
func (s *InvoiceTotalsStrategy) calculateExpectedTotal() decimal.Decimal {
	subTotalSales := decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotalSales.GetValue())

	nonTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonTaxed.GetValue())

	var totalTributos decimal.Decimal
	for _, tax := range s.Document.InvoiceSummary.TotalTaxes {
		taxValue := decimal.NewFromFloat(tax.GetValue())
		totalTributos = totalTributos.Add(taxValue)
	}

	expectedTotal := subTotalSales.Add(nonTaxed).Add(totalTributos)

	return expectedTotal
}

// compareTotalsWithTolerance compares two totals with a given tolerance
func (s *InvoiceTotalsStrategy) compareTotalsWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}

package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
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
		s.validateSummaryTaxCodes,
		s.validateSubTotalSales,
		s.validateSubTotal,
		s.validateDiscounts,
		s.validateTotalOperation,
		s.validateBalanceInFavorPayments,
	}

	for _, validate := range validations {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// validateSummaryTaxCodes valida que el resumen de FE no incluya los tributos 20 ni C3 (Anexo IV, campo 136):
// en FE el IVA va incluido en el precio y se informa en totalIva
func (s *InvoiceTotalsStrategy) validateSummaryTaxCodes() *dte_errors.DTEError {
	for _, tax := range s.Document.InvoiceSummary.TotalTaxes {
		if tax.GetCode() == constants.TaxIVA || tax.GetCode() == constants.TaxIVAExport {
			return dte_errors.NewDTEErrorSimple("TaxNotAllowedInInvoiceSummary", tax.GetCode())
		}
	}
	return nil
}

// validateSubTotalSales valida la suma de operaciones (campo 129)
func (s *InvoiceTotalsStrategy) validateSubTotalSales() *dte_errors.DTEError {
	summary := s.Document.InvoiceSummary
	return commonStrategy.ValidateSubTotalSales(
		summary.TotalNonSubject.GetValue(),
		summary.TotalExempt.GetValue(),
		summary.TotalTaxed.GetValue(),
		summary.SubTotalSales.GetValue())
}

// validateSubTotal validates the subtotal of the electronic invoice (campo 139):
// subTotalVentas - descuNoSuj - descuExenta - descuGravada
func (s *InvoiceTotalsStrategy) validateSubTotal() *dte_errors.DTEError {
	expectedSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue())

	if !commonStrategy.WithinTolerance(expectedSubTotal, actualSubTotal) {
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

	return nil
}

// validateTotalOperation validates the total operation amount of the electronic invoice (campo 150)
func (s *InvoiceTotalsStrategy) validateTotalOperation() *dte_errors.DTEError {
	summary := s.Document.InvoiceSummary
	return commonStrategy.ValidateTotalOperation(
		summary.SubTotal.GetValue(),
		summary.TotalTaxes,
		summary.TotalOperation.GetValue())
}

// validateBalanceInFavorPayments requires the no-payment-due payment form when the invoice carries a balance in favor.
func (s *InvoiceTotalsStrategy) validateBalanceInFavorPayments() *dte_errors.DTEError {
	if s.Document.InvoiceSummary.Summary == nil {
		return nil
	}
	return commonStrategy.ValidateBalanceInFavorPayments(s.Document.InvoiceSummary.Summary, s.Document.InvoiceSummary.BalanceInFavor.GetValue())
}

package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type CCFTaxStrategy struct {
	Document *ccf_models.CreditFiscalDocument
}

// Validate - Validates the CCF-specific fields
func (s *CCFTaxStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	if err := s.validateBaseTotals(); err != nil {
		logs.Error("Error validating base totals")
		return err
	}

	if err := s.validateIVA(); err != nil {
		logs.Error("Error validating IVA")
		return err
	}

	if err := s.validatePerception(); err != nil {
		logs.Error("Error validating perception")
		return err
	}

	if err := s.validateMonetaryAmounts(); err != nil {
		logs.Error("Error validating monetary amounts")
		return err
	}

	if err := s.validateTotalAmounts(); err != nil {
		logs.Error("Error validating total amounts")
		return err
	}

	if err := s.validateNonTaxedAmount(); err != nil {
		logs.Error("Error validating non-taxed amount")
		return err
	}

	return nil
}

func (s *CCFTaxStrategy) validateNonTaxedAmount() *dte_errors.DTEError {
	totalNonTaxed := s.Document.CreditSummary.TotalNonTaxed.GetValue()

	var sumItemsNonTaxed float64
	for _, item := range s.Document.CreditItems {
		sumItemsNonTaxed += item.NonTaxed.GetValue()
	}

	if totalNonTaxed > 0 && sumItemsNonTaxed == 0 {
		logs.Error("Invalid non-taxed amount", map[string]interface{}{
			"summaryTotal": totalNonTaxed,
			"itemsSum":     sumItemsNonTaxed,
		})
		return dte_errors.NewDTEErrorSimple("InvalidNonTaxedAmount")
	}

	if totalNonTaxed != sumItemsNonTaxed {
		logs.Error("Non-taxed amount mismatch", map[string]interface{}{
			"summaryTotal": totalNonTaxed,
			"itemsSum":     sumItemsNonTaxed,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalNonTaxed",
			sumItemsNonTaxed, totalNonTaxed)
	}

	return nil
}

func (s *CCFTaxStrategy) validateBaseTotals() *dte_errors.DTEError {
	var totalTaxed, totalNonSubject, totalExempt decimal.Decimal

	for _, item := range s.Document.CreditItems {
		totalTaxed = totalTaxed.Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))
		totalNonSubject = totalNonSubject.Add(decimal.NewFromFloat(item.NonSubjectSale.GetValue()))
		totalExempt = totalExempt.Add(decimal.NewFromFloat(item.ExemptSale.GetValue()))
	}

	summaryTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())
	summaryNonSubject := decimal.NewFromFloat(s.Document.CreditSummary.TotalNonSubject.GetValue())
	summaryExempt := decimal.NewFromFloat(s.Document.CreditSummary.TotalExempt.GetValue())

	diff := totalTaxed.Sub(summaryTaxed).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid taxed total", map[string]interface{}{
			"calculated": totalTaxed,
			"declared":   summaryTaxed,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalTaxed",
			summaryTaxed.InexactFloat64(),
			totalTaxed.InexactFloat64())
	}

	taxedDiscount := decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue())
	exemptDiscount := decimal.NewFromFloat(s.Document.CreditSummary.ExemptDiscount.GetValue())
	nonSubjectDiscount := decimal.NewFromFloat(s.Document.CreditSummary.NonSubjectDiscount.GetValue())

	if taxedDiscount.GreaterThan(totalTaxed) {
		logs.Error("Invalid taxed discount", map[string]interface{}{
			"taxedDiscount": taxedDiscount,
			"totalTaxed":    totalTaxed,
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"TaxedDiscount",
			taxedDiscount.InexactFloat64(),
			totalTaxed.InexactFloat64())
	}

	if exemptDiscount.GreaterThan(totalExempt) {
		logs.Error("Invalid exempt discount", map[string]interface{}{
			"exemptDiscount": exemptDiscount,
			"totalExempt":    totalExempt,
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"ExemptDiscount",
			exemptDiscount.InexactFloat64(),
			totalExempt.InexactFloat64())
	}

	if nonSubjectDiscount.GreaterThan(totalNonSubject) {
		logs.Error("Invalid non subject discount", map[string]interface{}{
			"nonSubjectDiscount": nonSubjectDiscount,
			"totalNonSubject":    totalNonSubject,
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsBase",
			"NonSubjectDiscount",
			nonSubjectDiscount.InexactFloat64(),
			totalNonSubject.InexactFloat64())
	}

	if !totalNonSubject.Equal(summaryNonSubject) {
		logs.Error("Invalid non-subject total", map[string]interface{}{
			"calculated": totalNonSubject,
			"declared":   summaryNonSubject,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalNonSubject",
			totalNonSubject.InexactFloat64(),
			summaryNonSubject.InexactFloat64())
	}

	if !totalExempt.Equal(summaryExempt) {
		logs.Error("Invalid exempt total", map[string]interface{}{
			"calculated": totalExempt,
			"declared":   summaryExempt,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalExempt",
			totalExempt.InexactFloat64(),
			summaryExempt.InexactFloat64())
	}

	expectedSubTotalSales := totalTaxed.Add(totalNonSubject).Add(totalExempt)
	actualSubTotalSales := decimal.NewFromFloat(s.Document.CreditSummary.SubTotalSales.GetValue())

	diff = expectedSubTotalSales.Sub(actualSubTotalSales).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid subtotal sales", map[string]interface{}{
			"calculated": expectedSubTotalSales,
			"declared":   actualSubTotalSales,
		})
		return dte_errors.NewDTEErrorSimple("InvalidSubTotalSales",
			expectedSubTotalSales.InexactFloat64(),
			actualSubTotalSales.InexactFloat64())
	}

	return nil
}

func (s *CCFTaxStrategy) validateIVA() *dte_errors.DTEError {
	baseTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())

	if !baseTaxed.GreaterThan(decimal.Zero) {

		if len(s.Document.CreditSummary.TotalTaxes) > 0 {
			logs.Error("Taxes present with zero taxed amount")
			return dte_errors.NewDTEErrorSimple("InvalidTaxes")
		}

		return nil
	}

	if len(s.Document.CreditSummary.TotalTaxes) == 0 {
		logs.Error("No taxes present with non-zero taxed amount")
		return dte_errors.NewDTEErrorSimple("MissingTaxes")
	}

	taxedDiscount := decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue())
	baseTaxedAfterDiscount := baseTaxed.Sub(taxedDiscount)

	for _, tax := range s.Document.CreditSummary.TotalTaxes {
		var expectedTax decimal.Decimal

		switch tax.GetCode() {
		case constants.TaxIVA:
			expectedTax = baseTaxedAfterDiscount.Mul(decimal.NewFromFloat(constants.TaxIvaAmount))
		case constants.TaxIVAExport:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxIVAExportAmount))
		case constants.TaxTourism:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxTourismAmount))
		case constants.TaxTourismAirport:
			expectedTax = decimal.NewFromFloat(constants.TaxTourismAirportAmount)
		case constants.TaxSpecialOther:
			continue
		}

		actualTax := decimal.NewFromFloat(tax.GetValue())
		diff := expectedTax.Sub(actualTax).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			logs.Error("Invalid tax calculation", map[string]interface{}{
				"taxCode":  tax.GetCode(),
				"expected": expectedTax,
				"actual":   actualTax,
			})

			return dte_errors.NewDTEErrorSimple("InvalidTaxCalculation",
				tax.GetCode(),
				expectedTax.InexactFloat64(),
				actualTax.InexactFloat64())
		}
	}

	return nil
}

func (s *CCFTaxStrategy) validatePerception() *dte_errors.DTEError {

	if s.Document.CreditSummary.IVAPerception.GetValue() != 0 {
		baseTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())
		expectedPerception := baseTaxed.Mul(decimal.NewFromFloat(0.01))
		actualPerception := decimal.NewFromFloat(s.Document.CreditSummary.IVAPerception.GetValue())

		diff := expectedPerception.Sub(actualPerception).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			return dte_errors.NewDTEErrorSimple("InvalidPerceptionAmount",
				actualPerception.StringFixed(2),
				expectedPerception.StringFixed(2))
		}
	}

	return nil
}

func (s *CCFTaxStrategy) validateTotalAmounts() *dte_errors.DTEError {
	totalOperation := decimal.NewFromFloat(s.Document.CreditSummary.TotalOperation.GetValue())

	taxedAmount := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())

	expectedSubTotal := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.CreditSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.CreditSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.CreditSummary.SubTotal.GetValue())
	diff := expectedSubTotal.Sub(actualSubTotal).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid subtotal calculation with discounts", map[string]interface{}{
			"expected":           expectedSubTotal,
			"actual":             actualSubTotal,
			"taxedDiscount":      s.Document.CreditSummary.TaxedDiscount.GetValue(),
			"exemptDiscount":     s.Document.CreditSummary.ExemptDiscount.GetValue(),
			"nonSubjectDiscount": s.Document.CreditSummary.NonSubjectDiscount.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidSubTotalCalculation",
			expectedSubTotal.InexactFloat64(),
			actualSubTotal.InexactFloat64())
	}

	if taxedAmount.GreaterThan(decimal.Zero) {
		hasIVA := false
		for _, tax := range s.Document.CreditSummary.TotalTaxes {
			if tax.GetCode() == constants.TaxIVA {
				actualIVA := decimal.NewFromFloat(tax.GetValue())
				if actualIVA.GreaterThan(decimal.Zero) {
					hasIVA = true
				}
				break
			}
		}

		if !hasIVA {
			logs.Error("Missing IVA for taxed amount", map[string]interface{}{
				"taxedAmount": taxedAmount,
			})
			return dte_errors.NewDTEErrorSimple("MissingIVAForTaxedAmount")
		}
	}

	// Anexo IV, campo 150: montoTotalOperacion = subTotal + tributos del resumen
	if err := commonStrategy.ValidateTotalOperation(s.Document.CreditSummary.SubTotal.GetValue(),
		s.Document.CreditSummary.TotalTaxes, s.Document.CreditSummary.TotalOperation.GetValue()); err != nil {
		logs.Error("Invalid total operation", map[string]interface{}{
			"subTotal":       s.Document.CreditSummary.SubTotal.GetValue(),
			"totalOperation": s.Document.CreditSummary.TotalOperation.GetValue(),
		})
		return err
	}

	// Normativa v2.0 (Anexo VI): se elimino "Retencion renta" para CCFE
	if s.Document.CreditSummary.IncomeRetention.GetValue() != 0 {
		logs.Error("Income retention is not allowed in CCF", map[string]interface{}{
			"incomeRetention": s.Document.CreditSummary.IncomeRetention.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("IncomeRetentionNotAllowed")
	}

	// Anexo IV, campo 153 (CCFE): Monto total de la operacion + IVA percibido - IVA retenido +/- cargos/abonos
	totalToPay := totalOperation.
		Add(decimal.NewFromFloat(s.Document.CreditSummary.IVAPerception.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.IVARetention.GetValue())).
		Add(decimal.NewFromFloat(s.Document.CreditSummary.TotalNonTaxed.GetValue()))

	// Regla 7: cuando existan abonos a favor del receptor y el resultado sea negativo, se ingresa 0.00
	if totalToPay.IsNegative() {
		totalToPay = decimal.Zero
	}

	actualTotalToPay := decimal.NewFromFloat(s.Document.CreditSummary.TotalToPay.GetValue())

	diff = totalToPay.Sub(actualTotalToPay).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid total to pay", map[string]interface{}{
			"calculated":    totalToPay,
			"declared":      actualTotalToPay,
			"difference":    diff,
			"operation":     totalOperation,
			"perception":    s.Document.CreditSummary.IVAPerception.GetValue(),
			"ivaRetention":  s.Document.CreditSummary.IVARetention.GetValue(),
			"totalNonTaxed": s.Document.CreditSummary.TotalNonTaxed.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalToPayCalculation",
			actualTotalToPay.InexactFloat64(),
			totalToPay.InexactFloat64())
	}

	return nil
}

func ValidateMonetaryAmount(amount float64, fieldName string) *dte_errors.DTEError {
	decValue := decimal.NewFromFloat(amount)
	multiplier := decimal.NewFromInt(100)
	scaled := decValue.Mul(multiplier)

	diff := scaled.Sub(decimal.NewFromInt(scaled.IntPart())).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidMonetaryAmount",
			fieldName,
			amount)
	}

	return nil
}

func (s *CCFTaxStrategy) validateMonetaryAmounts() *dte_errors.DTEError {
	if err := ValidateMonetaryAmount(s.Document.CreditSummary.IVAPerception.GetValue(), "iva_perception"); err != nil {
		return err
	}

	if err := ValidateMonetaryAmount(s.Document.CreditSummary.TotalOperation.GetValue(), "total_operation"); err != nil {
		return err
	}

	if err := ValidateMonetaryAmount(s.Document.CreditSummary.TotalToPay.GetValue(), "total_to_pay"); err != nil {
		return err
	}

	for _, payment := range s.Document.CreditSummary.GetPaymentTypes() {
		if err := ValidateMonetaryAmount(payment.GetAmount(), "payment_amount"); err != nil {
			return err
		}
	}

	return nil
}

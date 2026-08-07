package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DebitNoteTaxStrategy struct {
	Document *debit_note_models.DebitNoteModel
}

func (s *DebitNoteTaxStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("NullDocument", "DebitNoteTaxStrategy")
	}

	validations := []func() *dte_errors.DTEError{
		s.validateBaseTotals,
		s.validateDiscounts,
		s.validateIVA,
		s.validatePerception,
		s.validateTotalAmounts,
	}

	for _, validate := range validations {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s *DebitNoteTaxStrategy) validateBaseTotals() *dte_errors.DTEError {
	var totalTaxed, totalNonSubject, totalExempt decimal.Decimal

	for _, item := range s.Document.DebitItems {
		totalTaxed = totalTaxed.Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))
		totalNonSubject = totalNonSubject.Add(decimal.NewFromFloat(item.NonSubjectSale.GetValue()))
		totalExempt = totalExempt.Add(decimal.NewFromFloat(item.ExemptSale.GetValue()))
	}

	summaryTaxed := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue())
	summaryNonSubject := decimal.NewFromFloat(s.Document.DebitSummary.TotalNonSubject.GetValue())
	summaryExempt := decimal.NewFromFloat(s.Document.DebitSummary.TotalExempt.GetValue())

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
	actualSubTotalSales := decimal.NewFromFloat(s.Document.DebitSummary.SubTotalSales.GetValue())

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

func (s *DebitNoteTaxStrategy) validateDiscounts() *dte_errors.DTEError {
	totalDiscount := decimal.NewFromFloat(s.Document.DebitSummary.TotalDiscount.GetValue())
	taxedDiscount := decimal.NewFromFloat(s.Document.DebitSummary.TaxedDiscount.GetValue())
	exemptDiscount := decimal.NewFromFloat(s.Document.DebitSummary.ExemptDiscount.GetValue())
	nonSubjectDiscount := decimal.NewFromFloat(s.Document.DebitSummary.NonSubjectDiscount.GetValue())

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

	totalTaxed := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue())
	totalExempt := decimal.NewFromFloat(s.Document.DebitSummary.TotalExempt.GetValue())
	totalNonSubject := decimal.NewFromFloat(s.Document.DebitSummary.TotalNonSubject.GetValue())

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
	for _, item := range s.Document.DebitItems {
		itemDiscount := decimal.NewFromFloat(item.GetDiscount())
		itemsDiscountSum = itemsDiscountSum.Add(itemDiscount)
	}

	summaryDiscountsSum := taxedDiscount.Add(exemptDiscount).Add(nonSubjectDiscount)
	expectedTotalDiscount := itemsDiscountSum.Add(summaryDiscountsSum)

	diff := expectedTotalDiscount.Sub(totalDiscount).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalDiscountCalculation",
			totalDiscount.InexactFloat64(),
			expectedTotalDiscount.InexactFloat64())
	}

	return nil
}

func (s *DebitNoteTaxStrategy) validateIVA() *dte_errors.DTEError {
	baseTaxed := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue())

	if !baseTaxed.GreaterThan(decimal.Zero) {
		if len(s.Document.DebitSummary.TotalTaxes) > 0 {
			logs.Error("Taxes present with zero taxed amount")
			return dte_errors.NewDTEErrorSimple("InvalidTaxes")
		}
		return nil
	}

	if len(s.Document.DebitSummary.TotalTaxes) == 0 {
		logs.Error("No taxes present with non-zero taxed amount")
		return dte_errors.NewDTEErrorSimple("MissingTaxes")
	}

	for _, tax := range s.Document.DebitSummary.TotalTaxes {
		var expectedTax decimal.Decimal

		switch tax.GetCode() {
		case constants.TaxIVA:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxIvaAmount))
		case constants.TaxIVAExport:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxIVAExportAmount))
		case constants.TaxTourism:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxTourismAmount))
		case constants.TaxTourismAirport:
			expectedTax = decimal.NewFromFloat(constants.TaxTourismAirportAmount)
		case constants.TaxFOVIAL:
			expectedTax = baseTaxed.Mul(decimal.NewFromFloat(constants.TaxFOVIALAmount))
		case constants.TaxCOTRANS:
			expectedTax = decimal.NewFromFloat(constants.TaxCOTRANSAmount)
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
		}
	}

	return nil
}

func (s *DebitNoteTaxStrategy) validatePerception() *dte_errors.DTEError {
	if s.Document.DebitSummary.IVAPerception.GetValue() != 0 {
		baseTaxed := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue())
		expectedPerception := baseTaxed.Mul(decimal.NewFromFloat(0.01))
		actualPerception := decimal.NewFromFloat(s.Document.DebitSummary.IVAPerception.GetValue())

		diff := expectedPerception.Sub(actualPerception).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			return dte_errors.NewDTEErrorSimple("InvalidPerceptionAmount",
				actualPerception.InexactFloat64(),
				expectedPerception.InexactFloat64())
		}
	}

	return nil
}

func (s *DebitNoteTaxStrategy) validateTotalAmounts() *dte_errors.DTEError {
	totalOperation := decimal.NewFromFloat(s.Document.DebitSummary.TotalOperation.GetValue())
	taxedAmount := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue())

	expectedSubTotal := decimal.NewFromFloat(s.Document.DebitSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.DebitSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.DebitSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.DebitSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.DebitSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.DebitSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.DebitSummary.SubTotal.GetValue())
	diff := expectedSubTotal.Sub(actualSubTotal).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid subtotal calculation with discounts", map[string]interface{}{
			"expected":           expectedSubTotal,
			"actual":             actualSubTotal,
			"taxedDiscount":      s.Document.DebitSummary.TaxedDiscount.GetValue(),
			"exemptDiscount":     s.Document.DebitSummary.ExemptDiscount.GetValue(),
			"nonSubjectDiscount": s.Document.DebitSummary.NonSubjectDiscount.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidSubTotalCalculation",
			expectedSubTotal.InexactFloat64(),
			actualSubTotal.InexactFloat64())
	}

	if taxedAmount.GreaterThan(decimal.Zero) {
		hasIVA := false
		for _, tax := range s.Document.DebitSummary.TotalTaxes {
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

	expectedTotalOperation := actualSubTotal

	if taxedAmount.GreaterThan(decimal.Zero) {
		perception := decimal.NewFromFloat(s.Document.DebitSummary.IVAPerception.GetValue())
		expectedTotalOperation = expectedTotalOperation.Add(perception)

		ivaRetention := decimal.NewFromFloat(s.Document.DebitSummary.IVARetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(ivaRetention)

		incomeRetention := decimal.NewFromFloat(s.Document.DebitSummary.IncomeRetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(incomeRetention)
	}

	for _, tax := range s.Document.DebitSummary.TotalTaxes {
		expectedTotalOperation = expectedTotalOperation.Add(decimal.NewFromFloat(tax.GetValue()))
	}

	totalNonTaxed := decimal.NewFromFloat(s.Document.DebitSummary.TotalNonTaxed.GetValue())
	if totalNonTaxed.GreaterThan(decimal.Zero) {
		expectedTotalOperation = expectedTotalOperation.Add(totalNonTaxed)
	}

	actualTotalToPay := decimal.NewFromFloat(s.Document.DebitSummary.TotalToPay.GetValue())

	diff = expectedTotalOperation.Sub(actualTotalToPay).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid total to pay", map[string]interface{}{
			"calculated":      expectedTotalOperation,
			"declared":        actualTotalToPay,
			"difference":      diff,
			"operation":       totalOperation,
			"perception":      s.Document.DebitSummary.IVAPerception.GetValue(),
			"ivaRetention":    s.Document.DebitSummary.IVARetention.GetValue(),
			"incomeRetention": s.Document.DebitSummary.IncomeRetention.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalToPayCalculation",
			actualTotalToPay.InexactFloat64(),
			expectedTotalOperation.InexactFloat64())
	}

	return nil
}

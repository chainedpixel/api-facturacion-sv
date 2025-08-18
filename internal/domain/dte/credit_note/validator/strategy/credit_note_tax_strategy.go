package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/credit_note/credit_note_models"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/logs"
)

type CreditNoteTaxStrategy struct {
	Document *credit_note_models.CreditNoteModel
}

// Validate - Valida los campos específicos de una Nota de Crédito
func (s *CreditNoteTaxStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	// 1. Validar totales base
	if err := s.validateBaseTotals(); err != nil {
		logs.Error("Error validating base totals")
		return err
	}

	// 2. Validar IVA
	if err := s.validateIVA(); err != nil {
		logs.Error("Error validating IVA")
		return err
	}

	// 3. Validar percepción
	if err := s.validatePerception(); err != nil {
		logs.Error("Error validating perception")
		return err
	}

	// 4. Validar montos monetarios
	if err := s.validateMonetaryAmounts(); err != nil {
		logs.Error("Error validating monetary amounts")
		return err
	}

	// 5. Validar montos totales
	if err := s.validateTotalAmounts(); err != nil {
		logs.Error("Error validating total amounts")
		return err
	}
	return nil
}

func (s *CreditNoteTaxStrategy) validateBaseTotals() *dte_errors.DTEError {
	// 1. Calcular totales desde items
	var totalTaxed, totalNonSubject, totalExempt decimal.Decimal

	for _, item := range s.Document.CreditItems {
		totalTaxed = totalTaxed.Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))
		totalNonSubject = totalNonSubject.Add(decimal.NewFromFloat(item.NonSubjectSale.GetValue()))
		totalExempt = totalExempt.Add(decimal.NewFromFloat(item.ExemptSale.GetValue()))
	}

	// 2. Validar que los totales coincidan con el resumen
	summaryTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())
	summaryNonSubject := decimal.NewFromFloat(s.Document.CreditSummary.TotalNonSubject.GetValue())
	summaryExempt := decimal.NewFromFloat(s.Document.CreditSummary.TotalExempt.GetValue())

	// Verificar total gravado
	// Usar una pequeña tolerancia para comparaciones con decimales
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

	//Verificar que los descuentos no sobrepasen el subtotal
	if decimal.NewFromFloat(s.Document.CreditSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue())) {
		logs.Error("Invalid taxed discount", map[string]interface{}{
			"taxedDiscount": s.Document.CreditSummary.TaxedDiscount.GetValue(),
			"subTotal":      s.Document.CreditSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"TaxedDiscount",
			s.Document.CreditSummary.TaxedDiscount.GetValue(),
			s.Document.CreditSummary.SubTotal.GetValue())
	}

	if decimal.NewFromFloat(s.Document.CreditSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.CreditSummary.ExemptDiscount.GetValue())) {
		logs.Error("Invalid exempt discount", map[string]interface{}{
			"exemptDiscount": s.Document.CreditSummary.ExemptDiscount.GetValue(),
			"subTotal":       s.Document.CreditSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"ExemptDiscount",
			s.Document.CreditSummary.ExemptDiscount.GetValue(),
			s.Document.CreditSummary.SubTotal.GetValue())
	}

	if decimal.NewFromFloat(s.Document.CreditSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.CreditSummary.NonSubjectDiscount.GetValue())) {
		logs.Error("Invalid non subject discount", map[string]interface{}{
			"nonSubjectDiscount": s.Document.CreditSummary.NonSubjectDiscount.GetValue(),
			"subTotal":           s.Document.CreditSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"NonSubjectDiscount",
			s.Document.CreditSummary.NonSubjectDiscount.GetValue(),
			s.Document.CreditSummary.SubTotal.GetValue())
	}

	// Verificar total no sujeto
	if !totalNonSubject.Equal(summaryNonSubject) {
		logs.Error("Invalid non-subject total", map[string]interface{}{
			"calculated": totalNonSubject,
			"declared":   summaryNonSubject,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalNonSubject",
			totalNonSubject.InexactFloat64(),
			summaryNonSubject.InexactFloat64())
	}

	// Verificar total exento
	if !totalExempt.Equal(summaryExempt) {
		logs.Error("Invalid exempt total", map[string]interface{}{
			"calculated": totalExempt,
			"declared":   summaryExempt,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalExempt",
			totalExempt.InexactFloat64(),
			summaryExempt.InexactFloat64())
	}

	// 3. Validar que subtotal de ventas sea la suma de todos los tipos
	expectedSubTotalSales := totalTaxed.Add(totalNonSubject).Add(totalExempt)
	actualSubTotalSales := decimal.NewFromFloat(s.Document.CreditSummary.SubTotalSales.GetValue())

	// Usar una pequeña tolerancia para comparaciones con decimales
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

func (s *CreditNoteTaxStrategy) validateIVA() *dte_errors.DTEError {
	baseTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())

	// Si no hay monto gravado, no se requieren impuestos
	if !baseTaxed.GreaterThan(decimal.Zero) {
		if len(s.Document.CreditSummary.TotalTaxes) > 0 {
			logs.Error("Taxes present with zero taxed amount")
			return dte_errors.NewDTEErrorSimple("InvalidTaxes")
		}
		return nil
	}

	// Verificar que tenga al menos un impuesto válido
	if len(s.Document.CreditSummary.TotalTaxes) == 0 {
		logs.Error("No taxes present with non-zero taxed amount")
		return dte_errors.NewDTEErrorSimple("MissingTaxes")
	}

	// Validar el cálculo de cada impuesto
	for _, tax := range s.Document.CreditSummary.TotalTaxes {
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
		// Usar una pequeña tolerancia para comparaciones con decimales
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

func (s *CreditNoteTaxStrategy) validatePerception() *dte_errors.DTEError {
	if s.Document.CreditSummary.IVAPerception.GetValue() != 0 {
		baseTaxed := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())
		expectedPerception := baseTaxed.Mul(decimal.NewFromFloat(0.01))
		actualPerception := decimal.NewFromFloat(s.Document.CreditSummary.IVAPerception.GetValue())

		// Usar una pequeña tolerancia para comparaciones con decimales
		diff := expectedPerception.Sub(actualPerception).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			return dte_errors.NewDTEErrorSimple("InvalidPerceptionAmount",
				actualPerception.StringFixed(2),
				expectedPerception.StringFixed(2))
		}
	}

	return nil
}

func (s *CreditNoteTaxStrategy) validateTotalAmounts() *dte_errors.DTEError {
	// Obtener total operación
	totalOperation := decimal.NewFromFloat(s.Document.CreditSummary.TotalOperation.GetValue())

	// Obtener montos que afectan el total a pagar
	taxedAmount := decimal.NewFromFloat(s.Document.CreditSummary.TotalTaxed.GetValue())

	expectedSubTotal := decimal.NewFromFloat(s.Document.CreditSummary.SubTotalSales.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.ExemptDiscount.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.CreditSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.CreditSummary.SubTotal.GetValue())
	// Usar una pequeña tolerancia para comparaciones con decimales
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

	// Calcular IVA con descuento
	if taxedAmount.GreaterThan(decimal.Zero) {
		taxedWithDiscount := taxedAmount.
			Sub(decimal.NewFromFloat(s.Document.CreditSummary.TaxedDiscount.GetValue()))
		expectedIVA := taxedWithDiscount.Mul(decimal.NewFromFloat(0.13))

		for _, tax := range s.Document.CreditSummary.TotalTaxes {
			if tax.GetCode() == constants.TaxIVA {
				actualIVA := decimal.NewFromFloat(tax.GetValue())
				// Usar una pequeña tolerancia para comparaciones con decimales
				diff := expectedIVA.Sub(actualIVA).Abs()
				if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
					logs.Error("Invalid IVA calculation with discount", map[string]interface{}{
						"expected":      expectedIVA,
						"actual":        actualIVA,
						"taxedAmount":   taxedAmount,
						"taxedDiscount": s.Document.CreditSummary.TaxedDiscount.GetValue(),
					})
					return dte_errors.NewDTEErrorSimple("InvalidIVACalculation",
						expectedIVA.InexactFloat64(),
						actualIVA.InexactFloat64())
				}
				break
			}
		}
	}

	// Inicializar el total a pagar con el total operación
	expectedTotalOperation := actualSubTotal

	if taxedAmount.GreaterThan(decimal.Zero) {
		// Agregar percepción
		perception := decimal.NewFromFloat(s.Document.CreditSummary.IVAPerception.GetValue())
		expectedTotalOperation = expectedTotalOperation.Add(perception)

		// Restar retención IVA
		ivaRetention := decimal.NewFromFloat(s.Document.CreditSummary.IVARetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(ivaRetention)

		// Restar retención de renta
		incomeRetention := decimal.NewFromFloat(s.Document.CreditSummary.IncomeRetention.GetValue())
		expectedTotalOperation = expectedTotalOperation.Sub(incomeRetention)
	}

	for _, taxes := range s.Document.CreditSummary.GetTotalTaxes() {
		expectedTotalOperation = expectedTotalOperation.Add(decimal.NewFromFloat(taxes.GetTotalAmount()))
	}

	// Usar una pequeña tolerancia para comparaciones con decimales
	diff = expectedTotalOperation.Sub(totalOperation).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Invalid total operation", map[string]interface{}{
			"calculated":      expectedTotalOperation,
			"declared":        totalOperation,
			"difference":      diff,
			"operation":       totalOperation,
			"perception":      s.Document.CreditSummary.IVAPerception.GetValue(),
			"ivaRetention":    s.Document.CreditSummary.IVARetention.GetValue(),
			"incomeRetention": s.Document.CreditSummary.IncomeRetention.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalOperation",
			totalOperation.InexactFloat64(),
			expectedTotalOperation.InexactFloat64())
	}

	return nil
}

func (s *CreditNoteTaxStrategy) validateMonetaryAmounts() *dte_errors.DTEError {
	// Validar IVA Perception
	if err := ValidateMonetaryAmount(s.Document.CreditSummary.IVAPerception.GetValue(), "iva_perception"); err != nil {
		return err
	}

	// Validar Total Operation
	if err := ValidateMonetaryAmount(s.Document.CreditSummary.TotalOperation.GetValue(), "total_operation"); err != nil {
		return err
	}

	// Validar Payment Amounts
	for _, payment := range s.Document.CreditSummary.GetPaymentTypes() {
		if err := ValidateMonetaryAmount(payment.GetAmount(), "payment_amount"); err != nil {
			return err
		}
	}

	return nil
}

func ValidateMonetaryAmount(amount float64, fieldName string) *dte_errors.DTEError {
	decValue := decimal.NewFromFloat(amount)
	multiplier := decimal.NewFromInt(100)
	scaled := decValue.Mul(multiplier)

	// Usar una pequeña tolerancia para comparaciones con decimales
	diff := scaled.Sub(decimal.NewFromInt(scaled.IntPart())).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidMonetaryAmount",
			fieldName,
			amount)
	}

	return nil
}

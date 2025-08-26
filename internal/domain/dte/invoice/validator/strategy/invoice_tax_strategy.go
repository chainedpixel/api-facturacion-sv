package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/invoice/invoice_models"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/logs"
)

// InvoiceTaxStrategy implementa la validación de impuestos de una invoice electrónica
type InvoiceTaxStrategy struct {
	*strategy.TaxCalculationStrategy
	Document *invoice_models.ElectronicInvoice
}

// Validate valida los impuestos de la invoice, sobreescribiendo el método de la interfaz
func (s *InvoiceTaxStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.InvoiceItems) == 0 {
		return nil
	}

	// 1. Validar totales base
	if err := s.validateBaseTotals(); err != nil {
		return err
	}

	// 2. Validar IVA
	if err := s.validateIVA(); err != nil {
		return err
	}

	// 3. Validar montos monetarios
	if err := s.validateMonetaryAmounts(); err != nil {
		return err
	}

	// 4. Validar montos totales
	if err := s.validateTotalAmounts(); err != nil {
		return err
	}

	// 5. Validar estrategia de items
	for _, item := range s.Document.InvoiceItems {
		if len(s.Document.RelatedDocuments) > 0 {

			if item.GetRelatedDoc() == nil {
				logs.Error("Missing related document in item when related_docs is present", map[string]interface{}{
					"itemNumber": item.GetNumber(),
				})
				return dte_errors.NewDTEErrorSimple("MissingItemRelatedDoc", item.GetNumber())
			}

			found := false
			itemRelatedDoc := *item.GetRelatedDoc()

			for _, relatedDoc := range s.Document.RelatedDocuments {
				if relatedDoc.GetDocumentNumber() == itemRelatedDoc {
					found = true
					break
				}
			}

			if !found {
				logs.Error("Item related document not found in document related docs", map[string]interface{}{
					"itemNumber": item.GetNumber(),
					"relatedDoc": itemRelatedDoc,
				})
				return dte_errors.NewDTEErrorSimple("InvalidItemRelatedDoc",
					item.GetNumber(),
					itemRelatedDoc)
			}
		}
	}

	return nil
}

func (s *InvoiceTaxStrategy) validateTotalAmounts() *dte_errors.DTEError {
	// Obtener total operación
	totalOperation := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalOperation.GetValue())

	// Obtener montos que afectan el total a pagar
	taxedAmount := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())

	expectedSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue()).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.TaxedDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.ExemptDiscount.GetValue())).
		Add(decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())).
		Sub(decimal.NewFromFloat(s.Document.InvoiceSummary.NonSubjectDiscount.GetValue()))

	actualSubTotal := decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue())
	if !s.CompareTaxWithTolerance(actualSubTotal, expectedSubTotal, 0.01) {
		logs.Error("Invalid subtotal calculation", map[string]interface{}{
			"expected": expectedSubTotal,
			"actual":   actualSubTotal,
		})
		return dte_errors.NewDTEErrorSimple("InvalidSubTotalCalculation",
			expectedSubTotal.InexactFloat64(),
			actualSubTotal.InexactFloat64())
	}

	if taxedAmount.GreaterThan(decimal.Zero) {
		if len(s.Document.InvoiceSummary.TotalTaxes) > 0 {
			hasIVA := false
			for _, tax := range s.Document.InvoiceSummary.TotalTaxes {
				if tax.GetCode() == constants.TaxIVA {
					hasIVA = true
					break
				}
			}

			if !hasIVA {
				logs.Error("Missing IVA tax in TotalTaxes", map[string]interface{}{
					"taxedAmount": taxedAmount,
					"totalTaxes":  len(s.Document.InvoiceSummary.TotalTaxes),
				})
				return dte_errors.NewDTEErrorSimple("MissingIVAInTaxes")
			}
		}
	}

	// Inicializar el total a pagar con el total operación
	totalToPay := totalOperation

	if taxedAmount.GreaterThan(decimal.Zero) {
		// Restar retención IVA
		ivaRetention := decimal.NewFromFloat(s.Document.InvoiceSummary.IVARetention.GetValue())
		totalToPay = totalToPay.Sub(ivaRetention)

		// Restar retención de renta
		incomeRetention := decimal.NewFromFloat(s.Document.InvoiceSummary.IncomeRetention.GetValue())
		totalToPay = totalToPay.Sub(incomeRetention)
	}

	// Validar que la retención de IVA y la retención de renta no sean mayores al monto gravado
	if s.Document.InvoiceSummary.IVARetention.GetValue() > 0 && taxedAmount.IsZero() {
		logs.Error("IVA retention without taxed amount", map[string]interface{}{
			"ivaRetention": s.Document.InvoiceSummary.IVARetention.GetValue(),
			"taxedAmount":  taxedAmount,
		})
		return dte_errors.NewDTEErrorSimple("InvalidIVARetentionWithoutTaxedAmount")
	}
	if s.Document.InvoiceSummary.IncomeRetention.GetValue() > 0 && taxedAmount.IsZero() {
		logs.Error("Income retention without taxed amount", map[string]interface{}{
			"incomeRetention": s.Document.InvoiceSummary.IncomeRetention.GetValue(),
			"taxedAmount":     taxedAmount,
		})
		return dte_errors.NewDTEErrorSimple("InvalidIncomeRetentionWithoutTaxedAmount")
	}

	// Validar que las ventas no gravadas no sean mayores al monto no gravado
	var nonTaxed decimal.Decimal
	for _, item := range s.Document.InvoiceItems {
		nonTaxed = nonTaxed.Add(decimal.NewFromFloat(item.NonTaxed.GetValue()))
	}
	totalNonTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonTaxed.GetValue())
	if (nonTaxed.GreaterThan(decimal.Zero) && totalNonTaxed.Equal(decimal.Zero)) || (nonTaxed.Equal(decimal.Zero) && totalNonTaxed.GreaterThan(decimal.Zero)) {
		return dte_errors.NewDTEErrorSimple("InvalidNonTaxedAmount")
	}

	diff := nonTaxed.Sub(totalNonTaxed).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidNonTaxedAmountCalculation",
			totalNonTaxed.InexactFloat64(),
			nonTaxed.InexactFloat64())
	}

	// Agregar monto no gravado si existe
	if totalNonTaxed.GreaterThan(decimal.Zero) {
		totalToPay = totalToPay.Add(totalNonTaxed)
	}

	actualTotalToPay := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalToPay.GetValue())

	// Usar una pequeña tolerancia para comparaciones con decimales
	if !s.CompareTaxWithTolerance(totalToPay, actualTotalToPay, 0.01) {
		logs.Error("Invalid total to pay", map[string]interface{}{
			"calculated":      totalToPay,
			"declared":        actualTotalToPay,
			"difference":      diff,
			"operation":       totalOperation,
			"ivaRetention":    s.Document.InvoiceSummary.IVARetention.GetValue(),
			"incomeRetention": s.Document.InvoiceSummary.IncomeRetention.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalToPayCalculation",
			totalToPay.InexactFloat64(),
			actualTotalToPay.InexactFloat64())
	}

	return nil
}

func ValidateMonetaryAmount(amount float64, fieldName string) *dte_errors.DTEError {
	decValue := decimal.NewFromFloat(amount)
	multiplier := decimal.NewFromInt(100)
	scaled := decValue.Mul(multiplier)

	if !scaled.Equal(decimal.NewFromInt(scaled.IntPart())) {
		return dte_errors.NewDTEErrorSimple("InvalidMonetaryAmount",
			fieldName,
			amount)
	}

	return nil
}

func (s *InvoiceTaxStrategy) validateIVA() *dte_errors.DTEError {
	baseTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())

	// Si no hay monto gravado, no debe haber impuestos
	if !baseTaxed.GreaterThan(decimal.Zero) {
		if len(s.Document.InvoiceSummary.TotalTaxes) > 0 {
			return dte_errors.NewDTEErrorSimple("InvalidTaxes")
		}
		return nil
	}

	// Validar cada impuesto
	for _, tax := range s.Document.InvoiceSummary.TotalTaxes {
		if err := s.validateTaxCalculation(tax, baseTaxed); err != nil {
			return err
		}
	}

	// Validar los totales de impuestos
	if err := s.validateSummaryTaxes(); err != nil {
		return err
	}

	return nil
}

func (s *InvoiceTaxStrategy) validateMonetaryAmounts() *dte_errors.DTEError {
	// Validar Total Operation
	if err := ValidateMonetaryAmount(s.Document.InvoiceSummary.TotalOperation.GetValue(), "total_operation"); err != nil {
		return err
	}

	// Validar IVA Retention
	if err := ValidateMonetaryAmount(s.Document.InvoiceSummary.IVARetention.GetValue(), "iva_retention"); err != nil {
		return err
	}

	// Validar Income Retention
	if err := ValidateMonetaryAmount(s.Document.InvoiceSummary.IncomeRetention.GetValue(), "income_retention"); err != nil {
		return err
	}

	// Validar Total To Pay
	if err := ValidateMonetaryAmount(s.Document.InvoiceSummary.TotalToPay.GetValue(), "total_to_pay"); err != nil {
		return err
	}

	// Validar Payment Amounts
	for _, payment := range s.Document.InvoiceSummary.GetPaymentTypes() {
		if err := ValidateMonetaryAmount(payment.GetAmount(), "payment_amount"); err != nil {
			return err
		}
	}

	return nil
}

func (s *InvoiceTaxStrategy) validateTaxCalculation(tax interfaces.Tax, baseTaxed decimal.Decimal) *dte_errors.DTEError {
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
		return nil
	}

	actualTax := decimal.NewFromFloat(tax.GetValue())
	diff := expectedTax.Sub(actualTax).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidTaxCalculation",
			tax.GetCode(),
			expectedTax.InexactFloat64(),
			actualTax.InexactFloat64())
	}

	return nil
}

func (s *InvoiceTaxStrategy) validateBaseTotals() *dte_errors.DTEError {

	//Verificar que los descuentos no sobrepasen el subtotal
	if decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.InvoiceSummary.TaxedDiscount.GetValue())) {
		logs.Error("Invalid taxed discount", map[string]interface{}{
			"taxedDiscount": s.Document.InvoiceSummary.TaxedDiscount.GetValue(),
			"subTotal":      s.Document.InvoiceSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"TaxedDiscount",
			s.Document.InvoiceSummary.TaxedDiscount.GetValue(),
			s.Document.InvoiceSummary.SubTotal.GetValue())
	}

	if decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.InvoiceSummary.ExemptDiscount.GetValue())) {
		logs.Error("Invalid exempt discount", map[string]interface{}{
			"exemptDiscount": s.Document.InvoiceSummary.ExemptDiscount.GetValue(),
			"subTotal":       s.Document.InvoiceSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"ExemptDiscount",
			s.Document.InvoiceSummary.ExemptDiscount.GetValue(),
			s.Document.InvoiceSummary.SubTotal.GetValue())
	}

	if decimal.NewFromFloat(s.Document.InvoiceSummary.SubTotal.GetValue()).LessThan(decimal.NewFromFloat(s.Document.InvoiceSummary.NonSubjectDiscount.GetValue())) {
		logs.Error("Invalid non subject discount", map[string]interface{}{
			"nonSubjectDiscount": s.Document.InvoiceSummary.NonSubjectDiscount.GetValue(),
			"subTotal":           s.Document.InvoiceSummary.SubTotal.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("DiscountExceedsSubtotal",
			"NonSubjectDiscount",
			s.Document.InvoiceSummary.NonSubjectDiscount.GetValue(),
			s.Document.InvoiceSummary.SubTotal.GetValue())
	}

	return nil
}

// validateSummaryTaxes valida los totales de impuestos del resumen
func (s *InvoiceTaxStrategy) validateSummaryTaxes() *dte_errors.DTEError {
	// Cuando hay descuentos a nivel de taxed_discount, no se valida la coincidencia
	// entre IVA de items e IVA del resumen porque pueden ser diferentes por diseño.

	summaryIVA := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalIva.GetValue())
	taxedAmount := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())

	if taxedAmount.GreaterThan(decimal.Zero) && summaryIVA.LessThanOrEqual(decimal.Zero) {
		logs.Error("Missing IVA when taxed amount is present", map[string]interface{}{
			"taxedAmount": taxedAmount.InexactFloat64(),
			"summaryIVA":  summaryIVA.InexactFloat64(),
		})
		return dte_errors.NewDTEErrorSimple("MissingIVAForTaxedAmount")
	}

	return nil
}

// CompareTaxWithTolerance Compara dos impuestos con una tolerancia dada, normalmente 0.0001
func (s *InvoiceTaxStrategy) CompareTaxWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}

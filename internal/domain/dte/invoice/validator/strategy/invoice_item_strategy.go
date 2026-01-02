package strategy

import (
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/shopspring/decimal"

	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/invoice/invoice_models"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/logs"
)

type InvoiceItemsStrategy struct {
	Document *invoice_models.ElectronicInvoice
}

func (s *InvoiceItemsStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.InvoiceItems) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "InvoiceItems")
	}

	// Validar número máximo de items
	if len(s.Document.InvoiceItems) > 2000 {
		return dte_errors.NewDTEErrorSimple("ExceededItemsLimit", len(s.Document.InvoiceItems))
	}

	// Validar cada item
	for _, item := range s.Document.InvoiceItems {

		if err := s.validateItemSaleTypes(&item); err != nil {
			return err
		}

		if err := s.validateItem(item); err != nil {
			return err
		}
	}

	// Validar totales
	return s.validateTotals()
}

// validateItem valida un item de la invoice electrónica de venta
// Verifica que la suma de ventas coincida con el total y que el monto no gravado no exceda el total del item
func (s *InvoiceItemsStrategy) validateItem(item invoice_models.InvoiceItem) *dte_errors.DTEError {
	// IVA item sin venta gravada

	if item.TaxedSale.GetValue() > 0 && item.GetUnitPrice() == 0 {
		logs.Error("Taxed sale present without IVA item", map[string]interface{}{
			"itemNumber": item.GetNumber(),
			"taxedSale":  item.TaxedSale.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("MissingItemUnitPrice", item.GetNumber())
	}

	if item.IVAItem.GetValue() > 0 && item.TaxedSale.GetValue() == 0 {
		logs.Error("IVA item present without taxed sale", map[string]interface{}{
			"itemNumber": item.GetNumber(),
			"ivaItem":    item.IVAItem.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidIVAItemWithoutTaxedSale",
			item.GetNumber())
	}

	//Cálculo de IVA item
	if item.IVAItem.GetValue() > 0 {
		baseGravable := decimal.NewFromFloat(item.TaxedSale.GetValue()).
			Div(decimal.NewFromFloat(1.13))

		expectedIvaItem := baseGravable.Mul(decimal.NewFromFloat(0.13))
		actualIvaItem := decimal.NewFromFloat(item.IVAItem.GetValue())

		diff := expectedIvaItem.Sub(actualIvaItem).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			logs.Error("Invalid IVA item calculation", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"expected":   expectedIvaItem,
				"actual":     actualIvaItem,
				"taxedSale":  item.TaxedSale.GetValue(),
				"discount":   item.GetDiscount(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidIVAItemCalculation",
				item.GetNumber(),
				expectedIvaItem.InexactFloat64(),
				actualIvaItem.InexactFloat64())
		}
	}

	if item.TaxedSale.GetValue() > 0 && item.GetUnitPrice() == 0 {
		logs.Error("Unit price cannot be zero when taxed sale is present", map[string]interface{}{
			"itemNumber": item.GetNumber(),
			"taxedSale":  item.TaxedSale.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidUnitPriceZero",
			item.GetNumber(), item.GetUnitPrice(), item.TaxedSale.GetValue())
	}

	// Suma de ventas por tipo
	totalSales := decimal.NewFromFloat(item.NonSubjectSale.GetValue()).
		Add(decimal.NewFromFloat(item.ExemptSale.GetValue())).
		Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))

	// Máximo posible (precio * cantidad)
	maxPossible := decimal.NewFromFloat(item.GetUnitPrice()).
		Mul(decimal.NewFromFloat(item.GetQuantity()))

	// Validar que el total de ventas no exceda el máximo posible
	if totalSales.GreaterThan(maxPossible) {
		return dte_errors.NewDTEErrorSimple("ExcessiveItemTotal",
			totalSales.InexactFloat64(),
			maxPossible.InexactFloat64())
	}

	// Validar montos no gravados
	if item.NonTaxed.GetValue() != 0 {
		if err := s.validateNonTaxedAmount(item); err != nil {
			return err
		}
	}

	// La suma de todas las ventas no puede exceder el máximo posible
	if totalSales.GreaterThan(maxPossible) {
		return dte_errors.NewDTEErrorSimple("ExcessiveItemTotal",
			totalSales.InexactFloat64(),
			maxPossible.InexactFloat64())
	}

	if item.TaxedSale.GetValue() > 0 {
		expectedTaxed := decimal.NewFromFloat(item.GetUnitPrice()).
			Mul(decimal.NewFromFloat(item.GetQuantity())).
			Sub(decimal.NewFromFloat(item.GetDiscount()))

		// Calcular la diferencia absoluta
		diff := decimal.NewFromFloat(item.TaxedSale.GetValue()).
			Sub(expectedTaxed).
			Abs()

		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			return dte_errors.NewDTEErrorSimple("InvalidTaxedAmount",
				item.TaxedSale.GetValue(),
				expectedTaxed.InexactFloat64())
		}
	}

	for _, tax := range item.Taxes {
		if tax == constants.TaxIVA && item.GetType() == constants.Producto {
			return dte_errors.NewDTEErrorSimple("InvalidTaxForProduct", item.GetNumber())
		}
	}

	if item.GetType() == constants.Producto {
		for _, summaryTax := range s.Document.InvoiceSummary.TotalTaxes {
			if summaryTax.GetCode() == constants.TaxIVA {
				return dte_errors.NewDTEErrorSimple("InvalidSummaryTaxForProduct")
			}
		}
	}

	return nil
}

// validateNonTaxedAmount valida que el monto no gravado no exceda el total del item
func (s *InvoiceItemsStrategy) validateNonTaxedAmount(item invoice_models.InvoiceItem) *dte_errors.DTEError {
	nonTaxed := decimal.NewFromFloat(item.NonTaxed.GetValue())

	// Si hay monto no gravado, no debe haber otros tipos de venta
	if nonTaxed.GreaterThan(decimal.Zero) {
		if item.TaxedSale.GetValue() > 0 ||
			item.ExemptSale.GetValue() > 0 ||
			item.NonSubjectSale.GetValue() > 0 {
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithNonTaxed",
				item.GetNumber())
		}

		// Para montos no gravados, el precio unitario debe ser 0
		if item.GetUnitPrice() != 0 {
			return dte_errors.NewDTEErrorSimple("InvalidUnitPriceForNonTaxed",
				item.GetNumber())
		}
	}

	return nil
}

// validateTotals valida los totales de los items de la invoice electrónica
func (s *InvoiceItemsStrategy) validateTotals() *dte_errors.DTEError {
	var totalTaxed, totalExempt, totalNonSubject decimal.Decimal

	// Sumar totales de todos los items
	for _, item := range s.Document.InvoiceItems {
		totalTaxed = totalTaxed.Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))
		totalExempt = totalExempt.Add(decimal.NewFromFloat(item.ExemptSale.GetValue()))
		totalNonSubject = totalNonSubject.Add(decimal.NewFromFloat(item.NonSubjectSale.GetValue()))
	}

	// Validar contra resumen con tolerancia
	tolerance := decimal.NewFromFloat(0.01)

	// Validar total gravado
	summaryTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())
	if totalTaxed.Sub(summaryTaxed).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalTaxed",
			summaryTaxed.InexactFloat64(),
			totalTaxed.InexactFloat64())
	}

	// Validar total exento
	summaryExempt := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())
	if totalExempt.Sub(summaryExempt).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalExempt",
			totalExempt.InexactFloat64(),
			summaryExempt.InexactFloat64())
	}

	// Validar total no sujeto
	summaryNonSubject := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())
	if totalNonSubject.Sub(summaryNonSubject).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalNonSubject",
			totalNonSubject.InexactFloat64(),
			summaryNonSubject.InexactFloat64())
	}

	return nil
}

func (s *InvoiceItemsStrategy) validateItemSaleTypes(item *invoice_models.InvoiceItem) *dte_errors.DTEError {
	// Validar venta no gravada
	if item.NonTaxed.GetValue() > 0 {
		// No debe tener otros tipos de venta
		if item.TaxedSale.GetValue() > 0 || item.ExemptSale.GetValue() > 0 || item.NonSubjectSale.GetValue() > 0 {
			logs.Error("Items with non-taxed amount cannot have other sale types", map[string]interface{}{
				"itemNumber":     item.GetNumber(),
				"nonTaxed":       item.NonTaxed.GetValue(),
				"taxedSale":      item.TaxedSale.GetValue(),
				"exemptSale":     item.ExemptSale.GetValue(),
				"nonSubjectSale": item.NonSubjectSale.GetValue(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithNonTaxed", item.GetNumber())
		}
	}

	// Validar que no haya ventas mixtas
	salesTypes := 0
	if item.TaxedSale.GetValue() > 0 {
		salesTypes++
	}
	if item.ExemptSale.GetValue() > 0 {
		salesTypes++
	}
	if item.NonSubjectSale.GetValue() > 0 {
		salesTypes++
	}
	if item.NonTaxed.GetValue() > 0 {
		salesTypes++
	}

	if salesTypes > 1 {
		logs.Error("Mixed sales types in single item", map[string]interface{}{
			"itemNumber":     item.GetNumber(),
			"taxedSale":      item.TaxedSale.GetValue(),
			"exemptSale":     item.ExemptSale.GetValue(),
			"nonSubjectSale": item.NonSubjectSale.GetValue(),
			"nonTaxed":       item.NonTaxed.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("MixedSalesTypesNotAllowed", item.GetNumber())
	}

	return nil
}

// compareTotalsWithTolerance compara dos decimales con una tolerancia dada
func (s *InvoiceItemsStrategy) compareTotalsWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}

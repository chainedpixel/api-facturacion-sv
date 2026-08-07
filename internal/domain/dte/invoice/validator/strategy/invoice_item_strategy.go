package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type InvoiceItemsStrategy struct {
	Document *invoice_models.ElectronicInvoice
}

func (s *InvoiceItemsStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.InvoiceItems) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "InvoiceItems")
	}

	if len(s.Document.InvoiceItems) > 2000 {
		return dte_errors.NewDTEErrorSimple("ExceededItemsLimit", len(s.Document.InvoiceItems))
	}

	for _, item := range s.Document.InvoiceItems {

		if err := s.validateItemSaleTypes(&item); err != nil {
			return err
		}

		if err := s.validateItem(item); err != nil {
			return err
		}
	}

	return s.validateTotals()
}

// validateItem validates an item of the electronic sales invoice
// Verifies that the sum of sales matches the total and that the non-taxed amount does not exceed the item total
func (s *InvoiceItemsStrategy) validateItem(item invoice_models.InvoiceItem) *dte_errors.DTEError {

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

	totalSales := decimal.NewFromFloat(item.NonSubjectSale.GetValue()).
		Add(decimal.NewFromFloat(item.ExemptSale.GetValue())).
		Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))

	maxPossible := decimal.NewFromFloat(item.GetUnitPrice()).
		Mul(decimal.NewFromFloat(item.GetQuantity()))

	if totalSales.GreaterThan(maxPossible) {
		return dte_errors.NewDTEErrorSimple("ExcessiveItemTotal",
			totalSales.InexactFloat64(),
			maxPossible.InexactFloat64())
	}

	if item.NonTaxed.GetValue() != 0 {
		if err := s.validateNonTaxedAmount(item); err != nil {
			return err
		}
	}

	if totalSales.GreaterThan(maxPossible) {
		return dte_errors.NewDTEErrorSimple("ExcessiveItemTotal",
			totalSales.InexactFloat64(),
			maxPossible.InexactFloat64())
	}

	if item.TaxedSale.GetValue() > 0 {
		expectedTaxed := decimal.NewFromFloat(item.GetUnitPrice()).
			Mul(decimal.NewFromFloat(item.GetQuantity())).
			Sub(decimal.NewFromFloat(item.GetDiscount()))

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

// validateNonTaxedAmount validates that the non-taxed amount does not exceed the item total
func (s *InvoiceItemsStrategy) validateNonTaxedAmount(item invoice_models.InvoiceItem) *dte_errors.DTEError {
	nonTaxed := decimal.NewFromFloat(item.NonTaxed.GetValue())

	if nonTaxed.GreaterThan(decimal.Zero) {
		if item.TaxedSale.GetValue() > 0 ||
			item.ExemptSale.GetValue() > 0 ||
			item.NonSubjectSale.GetValue() > 0 {
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithNonTaxed",
				item.GetNumber())
		}

		if item.GetUnitPrice() != 0 {
			return dte_errors.NewDTEErrorSimple("InvalidUnitPriceForNonTaxed",
				item.GetNumber())
		}
	}

	return nil
}

// validateTotals validates the totals of the electronic invoice items
func (s *InvoiceItemsStrategy) validateTotals() *dte_errors.DTEError {
	var totalTaxed, totalExempt, totalNonSubject decimal.Decimal

	for _, item := range s.Document.InvoiceItems {
		totalTaxed = totalTaxed.Add(decimal.NewFromFloat(item.TaxedSale.GetValue()))
		totalExempt = totalExempt.Add(decimal.NewFromFloat(item.ExemptSale.GetValue()))
		totalNonSubject = totalNonSubject.Add(decimal.NewFromFloat(item.NonSubjectSale.GetValue()))
	}

	tolerance := decimal.NewFromFloat(0.01)

	summaryTaxed := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalTaxed.GetValue())
	if totalTaxed.Sub(summaryTaxed).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalTaxed",
			summaryTaxed.InexactFloat64(),
			totalTaxed.InexactFloat64())
	}

	summaryExempt := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalExempt.GetValue())
	if totalExempt.Sub(summaryExempt).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalExempt",
			totalExempt.InexactFloat64(),
			summaryExempt.InexactFloat64())
	}

	summaryNonSubject := decimal.NewFromFloat(s.Document.InvoiceSummary.TotalNonSubject.GetValue())
	if totalNonSubject.Sub(summaryNonSubject).Abs().GreaterThan(tolerance) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalNonSubject",
			totalNonSubject.InexactFloat64(),
			summaryNonSubject.InexactFloat64())
	}

	return nil
}

func (s *InvoiceItemsStrategy) validateItemSaleTypes(item *invoice_models.InvoiceItem) *dte_errors.DTEError {
	if item.NonTaxed.GetValue() > 0 {
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

// compareTotalsWithTolerance compares two decimals with a given tolerance
func (s *InvoiceItemsStrategy) compareTotalsWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}

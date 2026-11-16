package strategy

import (
	"slices"

	"github.com/shopspring/decimal"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
)

// Reglas de calculo compartidas por los DTE (Normativa DTE v2.0, reglas 7.2 y 7.3 y Anexo IV).
// Se centralizan aqui para que todos los tipos de documento apliquen la misma formula y la misma holgura.

// CalculationTolerance es la holgura de +/- 0.01 que Hacienda acepta en todo campo calculado (regla 7.3)
var CalculationTolerance = decimal.RequireFromString("0.01")

// WithinTolerance indica si dos montos difieren como maximo en la holgura permitida
func WithinTolerance(expected, actual decimal.Decimal) bool {
	return expected.Sub(actual).Abs().LessThanOrEqual(CalculationTolerance)
}

// IVAPerceptionRate is the IVA perception rate (1%) applied to the taxed total after the global discount
var IVAPerceptionRate = decimal.RequireFromString("0.01")

// ValidateIVAPerception validates the IVA perception (1%) of CCF, NCE and NDE:
// ivaPerci = round(1% x (totalGravada - descuGravada), 2). A zero declared perception is not validated.
func ValidateIVAPerception(totalTaxed, taxedDiscount, declared float64) *dte_errors.DTEError {
	if declared == 0 {
		return nil
	}

	expected := decimal.NewFromFloat(totalTaxed).
		Sub(decimal.NewFromFloat(taxedDiscount)).
		Mul(IVAPerceptionRate).
		Round(2)
	actual := decimal.NewFromFloat(declared)

	if !WithinTolerance(expected, actual) {
		return dte_errors.NewDTEErrorSimple("InvalidPerceptionAmount",
			actual.StringFixed(2), expected.StringFixed(2))
	}

	return nil
}

// PercentageTaxBase returns the taxable base of a percentage tax other than IVA. When prices include IVA (FE),
// the base is the amount without IVA (taxed amount divided by 1.13); otherwise the taxed amount is used as is.
// Both behaviors were confirmed against Hacienda for the tourism tax (59).
func PercentageTaxBase(pricesIncludeIVA bool, taxedAmount decimal.Decimal) decimal.Decimal {
	if pricesIncludeIVA {
		return taxedAmount.Div(decimal.NewFromInt(1).Add(decimal.NewFromFloat(constants.TaxIvaAmount)))
	}
	return taxedAmount
}

// ValidateItemSaleAmount valida la venta (o compra) de un item (Anexo IV, campos 89-91 y FSE):
// (precioUni x cantidad) - montoDescu. Los items de cargos/abonos usan precio 0 y no aplican esta formula.
func ValidateItemSaleAmount(itemNumber int, unitPrice, quantity, discount, declaredSale float64) *dte_errors.DTEError {
	if unitPrice == 0 {
		return nil
	}

	expected := decimal.NewFromFloat(unitPrice).
		Mul(decimal.NewFromFloat(quantity)).
		Sub(decimal.NewFromFloat(discount))
	declared := decimal.NewFromFloat(declaredSale)

	if !WithinTolerance(expected, declared) {
		return dte_errors.NewDTEErrorSimple("InvalidItemSaleAmount",
			itemNumber, declared.InexactFloat64(), expected.InexactFloat64())
	}

	return nil
}

// ValidateSubTotalSales valida la suma de operaciones (campo 129):
// subTotalVentas = totalNoSuj + totalExenta + totalGravada
func ValidateSubTotalSales(totalNonSubject, totalExempt, totalTaxed, declared float64) *dte_errors.DTEError {
	expected := decimal.NewFromFloat(totalNonSubject).
		Add(decimal.NewFromFloat(totalExempt)).
		Add(decimal.NewFromFloat(totalTaxed))
	actual := decimal.NewFromFloat(declared)

	if !WithinTolerance(expected, actual) {
		return dte_errors.NewDTEErrorSimple("InvalidSubTotalSales",
			actual.InexactFloat64(), expected.InexactFloat64())
	}

	return nil
}

// ValidateTotalOperation valida el monto total de la operacion (campo 150, FE, CCFE y NRE):
// montoTotalOperacion = subTotal + suma de los tributos del resumen
func ValidateTotalOperation(subTotal float64, taxes []interfaces.Tax, declared float64) *dte_errors.DTEError {
	expected := decimal.NewFromFloat(subTotal)
	for _, tax := range taxes {
		expected = expected.Add(decimal.NewFromFloat(tax.GetValue()))
	}
	actual := decimal.NewFromFloat(declared)

	if !WithinTolerance(expected, actual) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalOperation",
			actual.InexactFloat64(), expected.InexactFloat64())
	}

	return nil
}

// ValidateNoGlobalDiscounts rejects global discounts and discount percentage in credit and debit notes,
// whose schema has no such fields. Item discounts remain allowed.
func ValidateNoGlobalDiscounts(nonSubjectDiscount, exemptDiscount, taxedDiscount, discountPercentage float64) *dte_errors.DTEError {
	discounts := []struct {
		field string
		value float64
	}{
		{"non_subject_discount", nonSubjectDiscount},
		{"exempt_discount", exemptDiscount},
		{"taxed_discount", taxedDiscount},
		{"discount_percentage", discountPercentage},
	}

	for _, discount := range discounts {
		if discount.value != 0 {
			return dte_errors.NewDTEErrorSimple("GlobalDiscountNotAllowedInAdjustment", discount.field)
		}
	}
	return nil
}

// AdjustmentTotals contiene los montos del resumen de una NCE o NDE que intervienen en los campos 150 y 153
type AdjustmentTotals struct {
	SubTotalSales   float64
	Taxes           []interfaces.Tax
	IVAPerception   float64
	IVARetention    float64
	IncomeRetention float64
	TotalNonTaxed   float64
	TotalOperation  float64
	TotalToPay      float64
}

// ValidateAdjustmentTotals valida los totales de NCE y NDE (Anexo IV, campos 150 y 153; Anexo VI):
//   - montoTotalOperacion = subTotalVentas + tributos
//   - totalPagar = montoTotalOperacion + IVA percibido - IVA retenido +/- cargos/abonos
//   - la retencion de renta fue eliminada para NCE y NDE
func ValidateAdjustmentTotals(totals AdjustmentTotals) *dte_errors.DTEError {
	if totals.IncomeRetention != 0 {
		return dte_errors.NewDTEErrorSimple("IncomeRetentionNotAllowed")
	}

	if err := ValidateTotalOperation(totals.SubTotalSales, totals.Taxes, totals.TotalOperation); err != nil {
		return err
	}

	expectedToPay := decimal.NewFromFloat(totals.TotalOperation).
		Add(decimal.NewFromFloat(totals.IVAPerception)).
		Sub(decimal.NewFromFloat(totals.IVARetention)).
		Add(decimal.NewFromFloat(totals.TotalNonTaxed))
	declaredToPay := decimal.NewFromFloat(totals.TotalToPay)

	if !WithinTolerance(expectedToPay, declaredToPay) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalToPayCalculation",
			declaredToPay.InexactFloat64(), expectedToPay.InexactFloat64())
	}

	return nil
}

// summaryConsistentTaxCodes lists the additional tax codes that are computed at summary level and therefore must be
// declared by both the items and the summary. IVA (20) is not listed: FE does not declare it in the summary and the
// other documents already enforce it through the taxed amount rules.
var summaryConsistentTaxCodes = []string{constants.TaxTourism, constants.TaxTourismAirport}

// ItemTaxes describes the tax codes carried by an item and its taxed sale amount.
type ItemTaxes struct {
	TaxedSale decimal.Decimal
	Codes     []string
}

// NewItemTaxes builds the ItemTaxes of an item from its declared taxed sale and tax codes.
func NewItemTaxes(taxedSale float64, codes []string) ItemTaxes {
	return ItemTaxes{TaxedSale: decimal.NewFromFloat(taxedSale), Codes: codes}
}

// TaxedAmountWithTax sums the taxed sale of the items that carry the given additional tax code. It is the base of
// the percentage taxes computed at summary level (such as tourism), which Hacienda applies only to those items.
func TaxedAmountWithTax(items []ItemTaxes, taxCode string) decimal.Decimal {
	total := decimal.Zero
	for _, item := range items {
		if slices.Contains(item.Codes, taxCode) {
			total = total.Add(item.TaxedSale)
		}
	}
	return total
}

// PercentageTaxAmount returns the expected value of a percentage tax other than IVA: the rate applied to the taxed
// sale of the items carrying the code, rounded to 2 decimals. The global taxed discount is not subtracted.
func PercentageTaxAmount(pricesIncludeIVA bool, items []ItemTaxes, taxCode string, rate float64) decimal.Decimal {
	return PercentageTaxBase(pricesIncludeIVA, TaxedAmountWithTax(items, taxCode)).
		Mul(decimal.NewFromFloat(rate)).Round(2)
}

// ValidateSummaryTaxConsistency verifies, for the additional taxes computed at summary level, that every code carried
// by an item with taxed sale has an entry in the summary and that every summary entry is carried by at least one item
// with taxed sale. Items without taxed sale are ignored because they have no base for the tax.
func ValidateSummaryTaxConsistency(items []ItemTaxes, summaryTaxes []interfaces.Tax) *dte_errors.DTEError {
	for _, code := range summaryConsistentTaxCodes {
		carried := slices.ContainsFunc(items, func(item ItemTaxes) bool {
			return item.TaxedSale.IsPositive() && slices.Contains(item.Codes, code)
		})
		declared := slices.ContainsFunc(summaryTaxes, func(tax interfaces.Tax) bool {
			return tax.GetCode() == code
		})

		if carried && !declared {
			return dte_errors.NewDTEErrorSimple("MissingSummaryTax", code)
		}
		if declared && !carried {
			return dte_errors.NewDTEErrorSimple("MissingItemTaxForSummary", code)
		}
	}

	return nil
}

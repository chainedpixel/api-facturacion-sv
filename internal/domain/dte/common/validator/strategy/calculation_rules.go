package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// Reglas de calculo compartidas por los DTE (Normativa DTE v2.0, reglas 7.2 y 7.3 y Anexo IV).
// Se centralizan aqui para que todos los tipos de documento apliquen la misma formula y la misma holgura.

// CalculationTolerance es la holgura de +/- 0.01 que Hacienda acepta en todo campo calculado (regla 7.3)
var CalculationTolerance = decimal.RequireFromString("0.01")

// WithinTolerance indica si dos montos difieren como maximo en la holgura permitida
func WithinTolerance(expected, actual decimal.Decimal) bool {
	return expected.Sub(actual).Abs().LessThanOrEqual(CalculationTolerance)
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

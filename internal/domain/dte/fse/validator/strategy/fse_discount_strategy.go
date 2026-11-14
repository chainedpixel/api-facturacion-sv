package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/fse/fse_models"
)

type FSEDiscountStrategy struct {
	Document *fse_models.FSEModel
}

func NewFSEDiscountStrategy(document *fse_models.FSEModel) *FSEDiscountStrategy {
	return &FSEDiscountStrategy{
		Document: document,
	}
}

func (v *FSEDiscountStrategy) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	if v.Document == nil {
		return dte_errors.NewDTEErrorSimple("FSEDiscountValidatorNullDocument")
	}

	if err := v.validateItemDiscounts(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := v.validateSummaryDiscounts(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := v.validateTotalConsistency(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

// validateItemDiscounts valida cada item: el descuento no excede el monto del item y
// compra = (precioUni x cantidad) - montoDescu (regla compartida con FE y CCF)
func (v *FSEDiscountStrategy) validateItemDiscounts() *dte_errors.DTEError {
	for i, item := range v.Document.FSEItems {
		grossValue := decimal.NewFromFloat(item.Quantity.GetValue()).Mul(decimal.NewFromFloat(item.UnitPrice.GetValue()))
		discount := decimal.NewFromFloat(item.Discount.GetValue())
		if discount.GreaterThan(grossValue) {
			return dte_errors.NewDTEErrorSimple("ExcessiveItemDiscount",
				i+1, discount.InexactFloat64(), grossValue.InexactFloat64())
		}

		if err := commonStrategy.ValidateItemSaleAmount(i+1, item.UnitPrice.GetValue(), item.Quantity.GetValue(),
			item.Discount.GetValue(), item.Purchase.GetValue()); err != nil {
			return err
		}
	}
	return nil
}

// validateSummaryDiscounts valida totalCompra = suma de compras y subTotal = totalCompra - descu (campo 139)
func (v *FSEDiscountStrategy) validateSummaryDiscounts() *dte_errors.DTEError {
	summary := v.Document.FSESummary

	var calculatedTotalPurchase decimal.Decimal
	for _, item := range v.Document.FSEItems {
		calculatedTotalPurchase = calculatedTotalPurchase.Add(decimal.NewFromFloat(item.Purchase.GetValue()))
	}

	declaredTotalPurchase := decimal.NewFromFloat(summary.TotalPurchase.GetValue())
	if !commonStrategy.WithinTolerance(calculatedTotalPurchase, declaredTotalPurchase) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalPurchase",
			declaredTotalPurchase.InexactFloat64(), calculatedTotalPurchase.InexactFloat64())
	}

	expectedSubTotal := declaredTotalPurchase.Sub(decimal.NewFromFloat(summary.GetNonSubjectDiscount()))
	actualSubTotal := decimal.NewFromFloat(summary.SubTotal.GetValue())
	if !commonStrategy.WithinTolerance(expectedSubTotal, actualSubTotal) {
		return dte_errors.NewDTEErrorSimple("InvalidSubTotal",
			actualSubTotal.InexactFloat64(), expectedSubTotal.InexactFloat64())
	}

	return nil
}

// validateTotalConsistency valida totalDescu = descuentos por item + descu (campo 135, informativo)
func (v *FSEDiscountStrategy) validateTotalConsistency() *dte_errors.DTEError {
	var itemDiscountsTotal decimal.Decimal
	for _, item := range v.Document.FSEItems {
		itemDiscountsTotal = itemDiscountsTotal.Add(decimal.NewFromFloat(item.Discount.GetValue()))
	}

	expectedTotalDiscount := itemDiscountsTotal.Add(decimal.NewFromFloat(v.Document.FSESummary.GetNonSubjectDiscount()))
	actualTotalDiscount := decimal.NewFromFloat(v.Document.FSESummary.TotalDiscount.GetValue())

	if !commonStrategy.WithinTolerance(expectedTotalDiscount, actualTotalDiscount) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalDiscount",
			actualTotalDiscount.InexactFloat64(), expectedTotalDiscount.InexactFloat64())
	}

	return nil
}

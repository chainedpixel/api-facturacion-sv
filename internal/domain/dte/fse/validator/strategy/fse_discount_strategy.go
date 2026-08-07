package strategy

import (
	"fmt"
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
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

// validateItemDiscounts validates that the per-item discounts are consistent
func (v *FSEDiscountStrategy) validateItemDiscounts() *dte_errors.DTEError {
	for i, item := range v.Document.FSEItems {
		expectedPurchase := (item.Quantity.GetValue() * item.UnitPrice.GetValue()) - item.Discount.GetValue()
		actualPurchase := item.Purchase.GetValue()

		if math.Abs(expectedPurchase-actualPurchase) > 0.01 {
			return dte_errors.NewDTEErrorSimple("InvalidItemPurchase",
				formatItemError(i+1, expectedPurchase, actualPurchase))
		}

		grossValue := item.Quantity.GetValue() * item.UnitPrice.GetValue()
		if item.Discount.GetValue() > grossValue {
			return dte_errors.NewDTEErrorSimple("ExcessiveItemDiscount",
				formatItemDiscountError(i+1, item.Discount.GetValue(), grossValue))
		}
	}
	return nil
}

// validateSummaryDiscounts validates that the summary discounts are consistent
func (v *FSEDiscountStrategy) validateSummaryDiscounts() *dte_errors.DTEError {
	summary := v.Document.FSESummary

	var calculatedTotalPurchase float64
	for _, item := range v.Document.FSEItems {
		calculatedTotalPurchase += item.Purchase.GetValue()
	}

	if math.Abs(calculatedTotalPurchase-summary.TotalPurchase.GetValue()) > 0.01 {
		return dte_errors.NewDTEErrorSimple("InvalidTotalPurchase",
			formatTotalPurchaseError(calculatedTotalPurchase, summary.TotalPurchase.GetValue()))
	}

	expectedSubTotal := summary.TotalPurchase.GetValue() - summary.GetNonSubjectDiscount()
	actualSubTotal := summary.SubTotal.GetValue()

	if math.Abs(expectedSubTotal-actualSubTotal) > 0.01 {
		return dte_errors.NewDTEErrorSimple("InvalidSubTotal",
			formatSubTotalError(expectedSubTotal, actualSubTotal))
	}

	return nil
}

// validateTotalConsistency validates that total_discount is the sum of discounts
func (v *FSEDiscountStrategy) validateTotalConsistency() *dte_errors.DTEError {
	var itemDiscountsTotal float64
	for _, item := range v.Document.FSEItems {
		itemDiscountsTotal += item.Discount.GetValue()
	}

	expectedTotalDiscount := itemDiscountsTotal + v.Document.FSESummary.GetNonSubjectDiscount()
	actualTotalDiscount := v.Document.FSESummary.TotalDiscount.GetValue()

	if math.Abs(expectedTotalDiscount-actualTotalDiscount) > 0.01 {
		return dte_errors.NewDTEErrorSimple("InvalidTotalDiscount",
			formatTotalDiscountError(expectedTotalDiscount, actualTotalDiscount))
	}

	return nil
}

func formatItemError(itemNumber int, expected, actual float64) string {
	return fmt.Sprintf("Item %d: purchase esperado %.2f, actual %.2f", itemNumber, expected, actual)
}

func formatItemDiscountError(itemNumber int, discount, grossValue float64) string {
	return fmt.Sprintf("Item %d: descuento %.2f excede valor bruto %.2f", itemNumber, discount, grossValue)
}

func formatTotalPurchaseError(expected, actual float64) string {
	return fmt.Sprintf("TotalPurchase esperado %.2f, actual %.2f", expected, actual)
}

func formatSubTotalError(expected, actual float64) string {
	return fmt.Sprintf("SubTotal esperado %.2f, actual %.2f", expected, actual)
}

func formatTotalDiscountError(expected, actual float64) string {
	return fmt.Sprintf("TotalDiscount esperado %.2f, actual %.2f", expected, actual)
}

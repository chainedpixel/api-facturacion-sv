package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
)

type FSETaxStrategy struct {
	Document *fse_models.FSEModel
}

func NewFSETaxStrategy(document *fse_models.FSEModel) *FSETaxStrategy {
	return &FSETaxStrategy{
		Document: document,
	}
}

func (v *FSETaxStrategy) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	if v.Document == nil {
		return dte_errors.NewDTEErrorSimple("FSETaxInvalidDocument")
	}

	if err := v.validateRetentions(); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	if err := v.validateTotals(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := v.validateNoIVAFields(); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

func (v *FSETaxStrategy) validateRetentions() []*dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	summary := v.Document.FSESummary

	if !summary.IVARetention.IsValid() || summary.IVARetention.GetValue() < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSETaxInvalidIVARetention"))
	}

	if !summary.IncomeRetention.IsValid() || summary.IncomeRetention.GetValue() < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSETaxInvalidIncomeRetention"))
	}

	return errors
}

func (v *FSETaxStrategy) validateTotals() *dte_errors.DTEError {
	summary := v.Document.FSESummary

	var expectedTotal float64
	for _, item := range v.Document.FSEItems {
		expectedTotal += item.Purchase.GetValue()
	}

	if summary.TotalPurchase.GetValue() != expectedTotal {
	}

	expectedTotalToPay := summary.TotalPurchase.GetValue() -
		summary.IVARetention.GetValue() -
		summary.IncomeRetention.GetValue()

	if summary.TotalToPay.GetValue() != expectedTotalToPay {
		return dte_errors.NewDTEErrorSimple("FSETaxInvalidTotalToPay")
	}

	return nil
}

func (v *FSETaxStrategy) validateNoIVAFields() []*dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	summary := v.Document.FSESummary.Summary

	if summary.TotalTaxed.GetValue() > 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("FSETaxInvalidTaxedSale"))
	}

	return errors
}

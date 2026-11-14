package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/fse/fse_models"
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

	// Normativa v2.0 (Anexo VI): se elimino "IVA Retenido" para FSEE
	if summary.IVARetention.GetValue() != 0 {
		return dte_errors.NewDTEErrorSimple("IVARetentionNotAllowedInFSE")
	}

	// Anexo IV, campo 153 (FSEE): totalPagar = subTotal - reteRenta.
	// El total de compras y el subtotal se validan en FSEDiscountStrategy.
	expectedTotalToPay := decimal.NewFromFloat(summary.SubTotal.GetValue()).
		Sub(decimal.NewFromFloat(summary.IncomeRetention.GetValue()))
	actualTotalToPay := decimal.NewFromFloat(summary.TotalToPay.GetValue())

	if !commonStrategy.WithinTolerance(expectedTotalToPay, actualTotalToPay) {
		return dte_errors.NewDTEErrorSimple("FSETaxInvalidTotalToPay",
			actualTotalToPay.InexactFloat64(), expectedTotalToPay.InexactFloat64())
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

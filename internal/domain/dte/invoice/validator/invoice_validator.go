package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	strategy2 "github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/validator/strategy"
)

type InvoiceRulesValidator struct {
	document   *invoice_models.ElectronicInvoice
	strategies []interfaces.DTEValidationStrategy
}

// NewInvoiceRulesValidator Creates a rules validator for electronic invoices
func NewInvoiceRulesValidator(doc *invoice_models.ElectronicInvoice) *InvoiceRulesValidator {
	validator := &InvoiceRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy2.InvoiceItemsStrategy{Document: doc},
			&strategy2.InvoiceTaxStrategy{Document: doc},
			&strategy2.InvoiceTotalsStrategy{Document: doc},
		},
	}
	return validator
}

// Validate Executes the validations of the electronic invoice.
func (v *InvoiceRulesValidator) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	for _, strategyValidator := range v.strategies {
		if err := strategyValidator.Validate(); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

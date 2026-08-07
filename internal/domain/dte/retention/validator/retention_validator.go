package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/validator/strategy"
)

type RetentionRulesValidator struct {
	document   *retention_models.RetentionModel
	strategies []interfaces.DTEValidationStrategy
}

// NewRetentionRulesValidator Creates a rules validator for electronic invoices
func NewRetentionRulesValidator(doc *retention_models.RetentionModel) *RetentionRulesValidator {
	validator := &RetentionRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.RetentionItemStrategy{Document: doc},
			&strategy.RetentionTotalStrategy{Document: doc},
		},
	}
	return validator
}

// Validate Executes the validations of the electronic invoice.
func (v *RetentionRulesValidator) Validate() *dte_errors.DTEError {
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

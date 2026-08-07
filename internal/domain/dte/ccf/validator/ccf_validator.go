package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type CCFRulesValidator struct {
	document   *ccf_models.CreditFiscalDocument
	strategies []interfaces.DTEValidationStrategy
}

// NewCCFRulesValidator Creates a rules validator for CCF
func NewCCFRulesValidator(doc *ccf_models.CreditFiscalDocument) *CCFRulesValidator {
	validator := &CCFRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.CCFItemStrategy{Document: doc},
			&strategy.CCFTaxStrategy{Document: doc},
			&strategy.CCFReceiverStrategy{Document: doc},
			&strategy.CCFRelatedDocStrategy{Document: doc},
		},
	}
	return validator
}

// Validate Executes the fiscal credit voucher validations.
func (v *CCFRulesValidator) Validate() *dte_errors.DTEError {
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

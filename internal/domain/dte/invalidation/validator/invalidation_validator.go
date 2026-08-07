package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/validator/strategy"
)

type InvalidationRulesValidator struct {
	document   *invalidation_models.InvalidationDocument
	strategies []interfaces.DTEValidationStrategy
}

func NewInvalidationRulesValidator(doc *invalidation_models.InvalidationDocument) *InvalidationRulesValidator {
	validator := &InvalidationRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.InvalidationBasicStrategy{Document: doc},
			&strategy.InvalidationDocumentStrategy{Document: doc},
			&strategy.InvalidationReasonStrategy{Document: doc},
			&strategy.InvalidationDateStrategy{Document: doc},
		},
	}
	return validator
}

func (v *InvalidationRulesValidator) Validate() *dte_errors.DTEError {
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

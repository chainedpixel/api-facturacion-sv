package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DTERulesValidator struct {
	document   interfaces.DTEDocument
	strategies []interfaces.DTEValidationStrategy
}

// NewDTERulesValidator Creates a DTE rules validator
func NewDTERulesValidator(doc interfaces.DTEDocument) *DTERulesValidator {
	validator := &DTERulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.BasicRulesStrategy{Document: doc},
			&strategy.TemporalValidationStrategy{Document: doc},
			&strategy.ContingencyStrategy{Document: doc},
			&strategy.ModelTypeStrategy{Document: doc},
			&strategy.TaxCalculationStrategy{Document: doc},
			&strategy.PaymentTotalStrategy{Document: doc},
			&strategy.ItemValidationStrategy{Document: doc},
			&strategy.ExtensionStrategy{Document: doc},
			&strategy.DocumentTypeStrategy{Document: doc},
			&strategy.RelatedDocsStrategy{Document: doc},
			&strategy.ThirdPartyStrategy{Document: doc},
			&strategy.OtherDocumentsStrategy{Document: doc},
		},
	}
	return validator
}

// Validate Validates the rules of a DTE document according to the defined strategies
func (v *DTERulesValidator) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	for i, strategyValidator := range v.strategies {
		logs.Debug("Starting validation for strategy", map[string]interface{}{"strategy": i + 1})
		if err := strategyValidator.Validate(); err != nil {
			logs.Error("Failed to validate strategy", map[string]interface{}{"strategy": i + 1, "error": err.Error()})
			validationErrors = append(validationErrors, err)
		}
		logs.Debug("Finished validation for strategy", map[string]interface{}{"strategy": i + 1})
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

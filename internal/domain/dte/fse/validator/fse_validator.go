package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/validator/strategy"
)

type FSERulesValidator struct {
	document   *fse_models.FSEModel
	strategies []interfaces.DTEValidationStrategy
}

func NewFSERulesValidator(document *fse_models.FSEModel) *FSERulesValidator {
	validator := &FSERulesValidator{
		document: document,
	}

	validator.strategies = []interfaces.DTEValidationStrategy{
		strategy.NewFSEItemStrategy(document),
		strategy.NewFSETaxStrategy(document),
		strategy.NewFSEReceiverStrategy(document),
		strategy.NewFSEDiscountStrategy(document),
	}

	return validator
}

func (v *FSERulesValidator) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	if v.document == nil {
		return dte_errors.NewDTEErrorSimple("FSEValidatorNullDocument")
	}

	for _, strategy := range v.strategies {
		if err := strategy.Validate(); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

func (v *FSERulesValidator) GetDocument() *fse_models.FSEModel {
	return v.document
}

func (v *FSERulesValidator) AddStrategy(strategy interfaces.DTEValidationStrategy) {
	v.strategies = append(v.strategies, strategy)
}

func (v *FSERulesValidator) GetStrategies() []interfaces.DTEValidationStrategy {
	return v.strategies
}

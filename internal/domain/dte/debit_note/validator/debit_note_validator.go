package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/validator/strategy"
)

type DebitNoteRulesValidator struct {
	document   *debit_note_models.DebitNoteModel
	strategies []interfaces.DTEValidationStrategy
}

func NewDebitNoteRulesValidator(doc *debit_note_models.DebitNoteModel) *DebitNoteRulesValidator {
	validator := &DebitNoteRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.DebitNoteItemStrategy{Document: doc},
			&strategy.DebitNoteTaxStrategy{Document: doc},
			&strategy.DebitNoteRelatedDocStrategy{Document: doc},
		},
	}
	return validator
}

func (v *DebitNoteRulesValidator) Validate() *dte_errors.DTEError {
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

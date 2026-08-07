package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/validator/strategy"
)

type CreditNoteRulesValidator struct {
	document   *credit_note_models.CreditNoteModel
	strategies []interfaces.DTEValidationStrategy
}

func NewCreditNoteRulesValidator(doc *credit_note_models.CreditNoteModel) *CreditNoteRulesValidator {
	validator := &CreditNoteRulesValidator{
		document: doc,
		strategies: []interfaces.DTEValidationStrategy{
			&strategy.CreditNoteItemStrategy{Document: doc},
			&strategy.CreditNoteTaxStrategy{Document: doc},
			&strategy.CreditNoteRelatedDocStrategy{Document: doc},
		},
	}
	return validator
}

// Validate Executes the electronic credit note validations.
func (v *CreditNoteRulesValidator) Validate() *dte_errors.DTEError {
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

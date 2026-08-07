package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/validator/strategy"
)

type RemissionNoteRulesValidator struct {
	document   *remission_note_models.RemissionNoteModel
	strategies []interfaces.DTEValidationStrategy
}

func NewRemissionNoteRulesValidator(document *remission_note_models.RemissionNoteModel) *RemissionNoteRulesValidator {
	validator := &RemissionNoteRulesValidator{
		document: document,
	}

	if document != nil {
		validator.strategies = []interfaces.DTEValidationStrategy{
			&strategy.RemissionNoteItemStrategy{Document: document},
			&strategy.RemissionNoteSummaryStrategy{Document: document},
			&strategy.RemissionNoteReceiverStrategy{Document: document},
		}
	}

	return validator
}

func (v *RemissionNoteRulesValidator) Validate() *dte_errors.DTEError {
	if v.document == nil {
		return dte_errors.NewDTEErrorSimple("ValidationFailed", "RemissionNoteRulesValidator", "Document is nil")
	}

	var errors []*dte_errors.DTEError
	for _, strategy := range v.strategies {
		if err := strategy.Validate(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
)

type RemissionNoteItemStrategy struct {
	Document *remission_note_models.RemissionNoteModel
}

func (s *RemissionNoteItemStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("RemissionNoteItemValidationFailed")
	}

	if len(s.Document.RemissionItems) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "items")
	}

	var errors []*dte_errors.DTEError

	for i, item := range s.Document.RemissionItems {
		if err := s.validateItem(item, i+1); err != nil {
			errors = append(errors, err)
		}
	}

	if err := s.validateSequentialNumbers(); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

func (s *RemissionNoteItemStrategy) validateItem(item remission_note_models.RemissionNoteItem, position int) *dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	if item.Item == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "item")
	}

	if item.NonSubjectSale.GetValue() < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "non_subject_sale"))
	}

	if item.ExemptSale.GetValue() < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "exempt_sale"))
	}

	if item.TaxedSale.GetValue() < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "taxed_sale"))
	}

	totalSales := item.NonSubjectSale.GetValue() + item.ExemptSale.GetValue() + item.TaxedSale.GetValue()
	if totalSales == 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("AtLeastOneSaleTypeRequired"))
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

func (s *RemissionNoteItemStrategy) validateSequentialNumbers() *dte_errors.DTEError {
	for i, item := range s.Document.RemissionItems {
		expectedNumber := i + 1
		if item.Item.GetNumber() != expectedNumber {
			return dte_errors.NewDTEErrorSimple("InvalidSequence")
		}
	}
	return nil
}

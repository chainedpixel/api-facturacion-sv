package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
)

type RemissionNoteSummaryStrategy struct {
	Document *remission_note_models.RemissionNoteModel
}

func (s *RemissionNoteSummaryStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("RemissionNoteSummaryValidationFailed")
	}

	if s.Document.Summary == nil {
		return dte_errors.NewDTEErrorSimple("RequiredSummary", "Nota de Remisión")
	}

	var errors []*dte_errors.DTEError

	if err := s.validateTotalsConsistency(); err != nil {
		errors = append(errors, err)
	}

	if err := s.validateNonNegativeAmounts(); err != nil {
		errors = append(errors, err)
	}

	if err := s.validateSubtotalCalculation(); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

func (s *RemissionNoteSummaryStrategy) validateTotalsConsistency() *dte_errors.DTEError {
	var itemsNonSubject, itemsExempt, itemsTaxed float64

	for _, item := range s.Document.RemissionItems {
		itemsNonSubject += item.NonSubjectSale.GetValue()
		itemsExempt += item.ExemptSale.GetValue()
		itemsTaxed += item.TaxedSale.GetValue()
	}

	tolerance := 0.01

	if summaryNonSubject := s.Document.Summary.GetNonSubjectTotal(); summaryNonSubject != nil {
		if abs(*summaryNonSubject-itemsNonSubject) > tolerance {
			return dte_errors.NewDTEErrorSimple("InvalidTotalNonSubject", *summaryNonSubject, itemsNonSubject)
		}
	}

	if summaryExempt := s.Document.Summary.GetExemptTotal(); summaryExempt != nil {
		if abs(*summaryExempt-itemsExempt) > tolerance {
			return dte_errors.NewDTEErrorSimple("InvalidTotalExempt", *summaryExempt, itemsExempt)
		}
	}

	if summaryTaxed := s.Document.Summary.GetTaxedTotal(); summaryTaxed != nil {
		if abs(*summaryTaxed-itemsTaxed) > tolerance {
			return dte_errors.NewDTEErrorSimple("InvalidTotalTaxed", *summaryTaxed, itemsTaxed)
		}
	}

	return nil
}

func (s *RemissionNoteSummaryStrategy) validateNonNegativeAmounts() *dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	if totalNonSubject := s.Document.Summary.GetNonSubjectTotal(); totalNonSubject != nil && *totalNonSubject < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "non_subject_total"))
	}

	if totalExempt := s.Document.Summary.GetExemptTotal(); totalExempt != nil && *totalExempt < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "exempt_total"))
	}

	if totalTaxed := s.Document.Summary.GetTaxedTotal(); totalTaxed != nil && *totalTaxed < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "taxed_total"))
	}

	if totalAmount := s.Document.Summary.GetTotalAmount(); totalAmount != nil && *totalAmount < 0 {
		errors = append(errors, dte_errors.NewDTEErrorSimple("InvalidAmount", "total_amount"))
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

func (s *RemissionNoteSummaryStrategy) validateSubtotalCalculation() *dte_errors.DTEError {
	if s.Document.Summary.GetSubtotal() == nil || s.Document.Summary.GetTotalAmount() == nil {
		return nil
	}

	subtotal := *s.Document.Summary.GetSubtotal()
	totalAmount := *s.Document.Summary.GetTotalAmount()

	tolerance := 0.01
	if abs(subtotal-totalAmount) > tolerance {
		return dte_errors.NewDTEErrorSimple("InvalidTotalAmount", subtotal, totalAmount)
	}

	return nil
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

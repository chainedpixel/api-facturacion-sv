package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapRemissionNoteSummaryFromRequest(summary *structs.RemissionNoteSummaryRequest) (*remission_note_models.RemissionNoteSummary, error) {
	baseSummary := &models.Summary{}

	if err := mapBaseSummaryFields(summary, baseSummary); err != nil {
		return nil, err
	}

	remissionSummary := &remission_note_models.RemissionNoteSummary{
		Summary: baseSummary,
	}

	if summary.NonSubjectTotal != nil {
		nonSubjectTotal, err := financial.NewAmount(*summary.NonSubjectTotal)
		if err != nil {
			return nil, err
		}
		remissionSummary.NonSubjectTotal = nonSubjectTotal
	}

	if summary.ExemptTotal != nil {
		exemptTotal, err := financial.NewAmount(*summary.ExemptTotal)
		if err != nil {
			return nil, err
		}
		remissionSummary.ExemptTotal = exemptTotal
	}

	if summary.TaxedTotal != nil {
		taxedTotal, err := financial.NewAmount(*summary.TaxedTotal)
		if err != nil {
			return nil, err
		}
		remissionSummary.TaxedTotal = taxedTotal
	}

	if summary.SubtotalSales != nil {
		subtotalSales, err := financial.NewAmount(*summary.SubtotalSales)
		if err != nil {
			return nil, err
		}
		remissionSummary.SubtotalSales = subtotalSales
	}

	if summary.Subtotal != nil {
		subtotal, err := financial.NewAmount(*summary.Subtotal)
		if err != nil {
			return nil, err
		}
		remissionSummary.Subtotal = subtotal
	}

	if summary.TotalAmount != nil {
		totalAmount, err := financial.NewAmount(*summary.TotalAmount)
		if err != nil {
			return nil, err
		}
		remissionSummary.TotalAmount = totalAmount
	}

	if summary.TotalDiscount != nil {
		totalDiscount, err := financial.NewAmount(*summary.TotalDiscount)
		if err != nil {
			return nil, err
		}
		remissionSummary.TotalDiscount = totalDiscount
	}

	if summary.NonSubjectDiscount != nil {
		nonSubjectDiscount, err := financial.NewAmount(*summary.NonSubjectDiscount)
		if err != nil {
			return nil, err
		}
		remissionSummary.NonSubjectDiscount = nonSubjectDiscount
	}

	if summary.ExemptDiscount != nil {
		exemptDiscount, err := financial.NewAmount(*summary.ExemptDiscount)
		if err != nil {
			return nil, err
		}
		remissionSummary.ExemptDiscount = exemptDiscount
	}

	if summary.TaxedDiscount != nil {
		taxedDiscount, err := financial.NewAmount(*summary.TaxedDiscount)
		if err != nil {
			return nil, err
		}
		remissionSummary.TaxedDiscount = taxedDiscount
	}

	if summary.DiscountPercent != nil {
		remissionSummary.DiscountPercent = summary.DiscountPercent
	}

	if summary.AmountInWords == nil && summary.TotalAmount != nil {
		inLetters := utils.InLetters(*summary.TotalAmount)
		summary.AmountInWords = &inLetters
	}

	if summary.AmountInWords != nil {
		err := baseSummary.SetTotalInWords(*summary.AmountInWords)
		if err != nil {
			return nil, err
		}
	}

	return remissionSummary, nil
}

// mapBaseSummaryFields maps all fields to the base summary so they are available in the response
func mapBaseSummaryFields(summary *structs.RemissionNoteSummaryRequest, baseSummary *models.Summary) error {
	if err := baseSummary.SetTotalNonSubject(getFloatValue(summary.NonSubjectTotal)); err != nil {
		return err
	}
	if err := baseSummary.SetTotalExempt(getFloatValue(summary.ExemptTotal)); err != nil {
		return err
	}
	if err := baseSummary.SetTotalTaxed(getFloatValue(summary.TaxedTotal)); err != nil {
		return err
	}
	if err := baseSummary.SetSubTotal(getFloatValue(summary.Subtotal)); err != nil {
		return err
	}
	if err := baseSummary.SetSubtotalSales(getFloatValue(summary.SubtotalSales)); err != nil {
		return err
	}
	if err := baseSummary.SetNonSubjectDiscount(getFloatValue(summary.NonSubjectDiscount)); err != nil {
		return err
	}
	if err := baseSummary.SetExemptDiscount(getFloatValue(summary.ExemptDiscount)); err != nil {
		return err
	}
	if err := baseSummary.SetTotalDiscount(getFloatValue(summary.TotalDiscount)); err != nil {
		return err
	}
	if err := baseSummary.SetTotalOperation(getFloatValue(summary.TotalAmount)); err != nil {
		return err
	}
	if err := baseSummary.SetTotalToPay(getFloatValue(summary.TotalAmount)); err != nil {
		return err
	}
	if err := baseSummary.SetDiscountPercentage(getFloatValue(summary.DiscountPercent)); err != nil {
		return err
	}
	if summary.Observations != nil {
		if err := baseSummary.SetObservations(summary.Observations); err != nil {
			return err
		}
	}

	return nil
}

func getFloatValue(value *float64) float64 {
	if value == nil {
		return 0.0
	}
	return *value
}

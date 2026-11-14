package remission_note

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
)

func MapRemissionNoteSummary(model *remission_note_models.RemissionNoteSummary) *structs.MHRemissionNoteSummary {
	if model == nil {
		return nil
	}

	mhSummary := &structs.MHRemissionNoteSummary{}

	if model.Summary != nil {
		mhSummary.NonSubjectTotal = model.Summary.GetTotalNonSubject()
		mhSummary.ExemptTotal = model.Summary.GetTotalExempt()
		mhSummary.TaxedTotal = model.Summary.GetTotalTaxed()
		mhSummary.Subtotal = model.Summary.GetSubTotal()
		mhSummary.SubtotalSales = model.Summary.GetSubtotalSales()
		mhSummary.TotalAmount = model.Summary.GetTotalOperation()
		mhSummary.NonSubjectDiscount = model.Summary.GetNonSubjectDiscount()
		mhSummary.ExemptDiscount = model.Summary.GetExemptDiscount()
		mhSummary.TotalDiscount = model.Summary.GetTotalDiscount()
		mhSummary.Observaciones = model.Summary.GetObservations()
	}

	if model.TaxedDiscount != nil {
		mhSummary.TaxedDiscount = model.TaxedDiscount.GetValue()
	}

	if model.DiscountPercent != nil {
		mhSummary.DiscountPercent = model.DiscountPercent
	} else if model.Summary != nil {
		discountPercent := model.Summary.GetDiscountPercentage()
		mhSummary.DiscountPercent = &discountPercent
	}

	if model.Summary != nil {
		mhSummary.AmountInWords = model.Summary.GetTotalInWords()
	} else {
		mhSummary.AmountInWords = "CERO"
	}

	// Sin tributos se envia null (el schema admite array con elementos o null)
	if model.Summary != nil && len(model.Summary.GetTotalTaxes()) > 0 {
		taxes := make([]structs.DTETax, len(model.Summary.GetTotalTaxes()))
		for i, tax := range model.Summary.GetTotalTaxes() {
			taxes[i] = structs.DTETax{
				Codigo:      tax.GetCode(),
				Descripcion: tax.GetDescription(),
				Valor:       tax.GetValue(),
			}
		}
		mhSummary.Tributes = taxes
	}

	return mhSummary
}

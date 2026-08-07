package remission_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type RemissionNoteSummary struct {
	*models.Summary
	NonSubjectTotal    *financial.Amount        `json:"nonSubjectTotal,omitempty"`
	ExemptTotal        *financial.Amount        `json:"exemptTotal,omitempty"`
	TaxedTotal         *financial.Amount        `json:"taxedTotal,omitempty"`
	SubtotalSales      *financial.Amount        `json:"subtotalSales,omitempty"`
	NonSubjectDiscount *financial.Amount        `json:"nonSubjectDiscount,omitempty"`
	ExemptDiscount     *financial.Amount        `json:"exemptDiscount,omitempty"`
	TaxedDiscount      *financial.Amount        `json:"taxedDiscount,omitempty"`
	DiscountPercent    *float64                 `json:"discountPercent,omitempty"`
	TotalDiscount      *financial.Amount        `json:"totalDiscount,omitempty"`
	Subtotal           *financial.Amount        `json:"subtotal,omitempty"`
	TotalAmount        *financial.Amount        `json:"totalAmount,omitempty"`
	PaymentTypes       []interfaces.PaymentType `json:"paymentTypes,omitempty"`
}

func (r *RemissionNoteSummary) GetNonSubjectTotal() *float64 {
	if r.NonSubjectTotal == nil {
		return nil
	}
	value := r.NonSubjectTotal.GetValue()
	return &value
}

func (r *RemissionNoteSummary) GetExemptTotal() *float64 {
	if r.ExemptTotal == nil {
		return nil
	}
	value := r.ExemptTotal.GetValue()
	return &value
}

func (r *RemissionNoteSummary) GetTaxedTotal() *float64 {
	if r.TaxedTotal == nil {
		return nil
	}
	value := r.TaxedTotal.GetValue()
	return &value
}

func (r *RemissionNoteSummary) GetSubtotal() *float64 {
	if r.Subtotal == nil {
		return nil
	}
	value := r.Subtotal.GetValue()
	return &value
}

func (r *RemissionNoteSummary) GetTotalAmount() *float64 {
	if r.TotalAmount == nil {
		return nil
	}
	value := r.TotalAmount.GetValue()
	return &value
}

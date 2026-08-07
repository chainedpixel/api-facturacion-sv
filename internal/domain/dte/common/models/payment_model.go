package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// PaymentType is a structure that represents a payment type of a DTE, contains Code, Amount, Reference, Term and Period
type PaymentType struct {
	Code      financial.PaymentType  `json:"code"`
	Amount    financial.Amount       `json:"amount"`
	Reference string                 `json:"reference"`
	Term      *financial.PaymentTerm `json:"term,omitempty"`
	Period    *int                   `json:"period,omitempty"`
}

func (p *PaymentType) GetCode() string {
	return p.Code.GetValue()
}
func (p *PaymentType) GetAmount() float64 {
	return p.Amount.GetValue()
}
func (p *PaymentType) GetReference() string {
	return p.Reference
}
func (p *PaymentType) GetTerm() *string {
	if p.Term == nil {
		return nil
	}

	return utils.ToStringPointer(p.Term.GetValue())
}
func (p *PaymentType) GetPeriod() *int {
	if p == nil {
		return nil
	}

	return p.Period
}
func (p *PaymentType) GetPeriodPointer() *int {
	return p.Period
}

func (p *PaymentType) SetCode(code string) error {
	codeObj, err := financial.NewPaymentType(code)
	if err != nil {
		return err
	}
	p.Code = *codeObj
	return nil
}

func (p *PaymentType) SetAmount(amount float64) error {
	amountObj, err := financial.NewAmount(amount)
	if err != nil {
		return err
	}
	p.Amount = *amountObj
	return nil
}

func (p *PaymentType) SetReference(reference string) error {
	if reference == "" {
		return dte_errors.NewValidationError("RequiredField", "Reference")
	}
	p.Reference = reference
	return nil
}

func (p *PaymentType) SetTerm(term *string) error {
	if term == nil {
		p.Term = nil
		return nil
	}

	termObj, err := financial.NewPaymentTerm(*term)
	if err != nil {
		return err
	}
	p.Term = termObj
	return nil
}

func (p *PaymentType) SetPeriod(period *int) error {
	p.Period = period
	return nil
}

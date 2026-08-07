package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

// TaxAmount is a structure that represents an amount of a tax in a DTE, contains TotalAmount
type TaxAmount struct {
	TotalAmount financial.Amount `json:"totalAmount,omitempty"`
}

func (t *TaxAmount) GetTotalAmount() financial.Amount {
	return t.TotalAmount
}

// Tax is a structure that represents a tax in a DTE, contains Code, Description and Value
type Tax struct {
	Code        financial.TaxType `json:"code"`
	Description string            `json:"description"`
	Value       *TaxAmount        `json:"value,omitempty"`
}

func (t *Tax) GetTotalAmount() float64 {
	return t.Value.TotalAmount.GetValue()
}

func (t *Tax) GetCode() string {
	return t.Code.GetValue()
}
func (t *Tax) GetDescription() string {
	return t.Description
}
func (t *Tax) GetValue() float64 {
	return t.Value.TotalAmount.GetValue()
}

func (t *Tax) SetTotalAmount(totalAmount float64) error {
	taObj, err := financial.NewAmount(totalAmount)
	if err != nil {
		return err
	}
	if t.Value == nil {
		t.Value = &TaxAmount{}
	}
	t.Value.TotalAmount = *taObj
	return nil
}

func (t *Tax) SetCode(code string) error {
	codeObj, err := financial.NewTaxType(code)
	if err != nil {
		return err
	}
	t.Code = *codeObj
	return nil
}

func (t *Tax) SetDescription(description string) error {
	if description == "" {
		return dte_errors.NewValidationError("RequiredField", "Description")
	}
	t.Description = description
	return nil
}

func (t *Tax) SetValue(value float64) error {
	return t.SetTotalAmount(value)
}

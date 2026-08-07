package base

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Phone struct {
	Value string `json:"value"`
}

func NewPhone(value string) (*Phone, error) {
	phone := &Phone{Value: value}
	if phone.IsValid() {
		return phone, nil
	}
	return &Phone{}, dte_errors.NewValidationError("InvalidPhone", value)
}

func NewValidatedPhone(value string) *Phone {
	return &Phone{Value: value}
}

// IsValid validates that the phone number has between 8 and 30 digits
func (p *Phone) IsValid() bool {
	return len(p.Value) >= 8 && len(p.Value) <= 30
}

func (p *Phone) Equals(other interfaces.ValueObject[string]) bool {
	return p.GetValue() == other.GetValue()
}

func (p *Phone) GetValue() string {
	return p.Value
}

func (p *Phone) ToString() string {
	return p.Value
}

package financial

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Currency struct {
	Value string `json:"value"`
}

func NewCurrency(value string) (*Currency, error) {
	currency := &Currency{Value: value}
	if currency.IsValid() {
		return currency, nil
	}
	return &Currency{}, dte_errors.NewValidationError("InvalidCurrency", value)
}

func NewValidatedCurrency(value string) *Currency {
	return &Currency{Value: value}
}

func (c *Currency) IsValid() bool {
	return c.Value == "USD"
}

func (c *Currency) Equals(other interfaces.ValueObject[string]) bool {
	return c.GetValue() == other.GetValue()
}

func (c *Currency) GetValue() string {
	return c.Value
}

func (c *Currency) ToString() string {
	return c.Value
}

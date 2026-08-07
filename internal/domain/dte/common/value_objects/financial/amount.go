package financial

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Amount struct {
	Value float64 `json:"value"`
}

func NewAmount(value float64) (*Amount, error) {
	decValue := decimal.NewFromFloat(value)

	decValue = decValue.Round(8)
	roundedValue, _ := decValue.Float64()
	amount := &Amount{Value: roundedValue}

	if !amount.IsValid() {
		return nil, dte_errors.NewValidationError("InvalidAmount",
			fmt.Sprintf("%f", roundedValue))
	}

	return amount, nil
}

func NewAmountForTotal(value float64) (*Amount, error) {
	decValue := decimal.NewFromFloat(value)

	multiplier := decimal.NewFromInt(100)
	scaled := decValue.Mul(multiplier)

	if !scaled.Equal(decimal.NewFromInt(scaled.IntPart())) {
		return nil, dte_errors.NewValidationError("InvalidDecimals", fmt.Sprintf("%f", value))
	}

	roundedValue, _ := decValue.Round(2).Float64()
	amount := &Amount{Value: roundedValue}

	if !amount.IsValid() {
		return nil, dte_errors.NewValidationError("InvalidAmount",
			fmt.Sprintf("%f", value))
	}

	return amount, nil
}

func NewValidatedAmount(value float64) *Amount {
	return &Amount{Value: value}
}

// IsValid validates that the Amount value is greater than or equal to 0 and less than or equal to 99999999999.99
func (a *Amount) IsValid() bool {
	decValue := decimal.NewFromFloat(a.Value)

	return decValue.GreaterThanOrEqual(decimal.Zero) &&
		decValue.LessThanOrEqual(decimal.NewFromFloat(99999999999.99))
}

func (a *Amount) Equals(other interfaces.ValueObject[float64]) bool {
	return math.Abs(a.GetValue()-other.GetValue()) < 0.000001
}

func (a *Amount) GetValue() float64 {
	if a == nil {
		return 0
	}

	return a.Value
}

func (a *Amount) GetValueAsDecimal() decimal.Decimal {
	if a == nil {
		return decimal.Zero
	}

	return decimal.NewFromFloat(a.Value)
}

func (a *Amount) ToString() string {
	return fmt.Sprintf("%.2f", a.Value)
}

func (a *Amount) Add(other *Amount) {
	sum := decimal.NewFromFloat(a.Value).
		Add(decimal.NewFromFloat(other.Value))

	result, _ := sum.Round(2).Float64()
	a.Value = result
}

func (a *Amount) Mul(value float64) (*Amount, error) {
	product := decimal.NewFromFloat(a.Value).
		Mul(decimal.NewFromFloat(value))

	result, _ := product.Round(2).Float64()
	return NewAmount(result)
}

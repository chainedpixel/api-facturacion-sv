package financial

import (
	"fmt"
	"math"
	"strconv"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Tax struct {
	Value float64 `json:"value"`
}

func NewTax(value float64) (*Tax, error) {
	tax := &Tax{Value: value}
	if tax.IsValid() {
		return tax, nil
	}
	return &Tax{}, dte_errors.NewValidationError("InvalidTax", fmt.Sprintf("%.2f", value))
}

func NewValidatedTax(value float64) *Tax {
	return &Tax{Value: value}
}

// IsValid validates that the Tax value is greater than or equal to 0 and less than 100000000000 (100 billion)
func (t *Tax) IsValid() bool {
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", t.Value), 64)
	return value >= 0 && value < 100000000000
}

func (t *Tax) Equals(other interfaces.ValueObject[float64]) bool {
	return math.Abs(t.GetValue()-other.GetValue()) < 0.000001
}

func (t *Tax) GetValue() float64 {
	return t.Value
}

func (t *Tax) ToString() string {
	return fmt.Sprintf("%.2f", t.Value)
}

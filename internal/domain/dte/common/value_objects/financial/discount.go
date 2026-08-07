package financial

import (
	"fmt"
	"math"
	"strconv"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Discount struct {
	Value float64 `json:"value"`
}

func NewDiscount(value float64) (*Discount, error) {
	discount := &Discount{Value: value}
	if discount.IsValid() {
		return discount, nil
	}
	return &Discount{}, dte_errors.NewValidationError("InvalidDiscount", fmt.Sprintf("%.2f", value))
}

func NewValidatedDiscount(value float64) *Discount {
	return &Discount{Value: value}
}

// IsValid validates that the discount percentage is in the range [0, 100].
func (d *Discount) IsValid() bool {
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", d.Value), 64)
	return value >= 0 && value <= 100
}

func (d *Discount) Equals(other interfaces.ValueObject[float64]) bool {
	return math.Abs(d.GetValue()-other.GetValue()) < 0.000001
}

func (d *Discount) GetValue() float64 {
	return d.Value
}

func (d *Discount) ToString() string {
	return fmt.Sprintf("%.2f", d.Value)
}

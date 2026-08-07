package financial

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type PaymentType struct {
	Value string `json:"value"`
}

func NewPaymentType(value string) (*PaymentType, error) {
	pt := &PaymentType{Value: value}
	if pt.IsValid() {
		return pt, nil
	}
	return &PaymentType{}, dte_errors.NewValidationError("InvalidLength", "payment.code", "01-14 o 99", value)
}

func NewValidatedPaymentType(value string) *PaymentType {
	return &PaymentType{Value: value}
}

// IsValid validates that the PaymentType value is 01 to 14, 99 (two digits)
func (pt *PaymentType) IsValid() bool {
	pattern := `^(0[1-9]||1[0-4]||99)$`
	matched, _ := regexp.MatchString(pattern, pt.Value)
	return matched
}

func (pt *PaymentType) Equals(other interfaces.ValueObject[string]) bool {
	return pt.GetValue() == other.GetValue()
}

func (pt *PaymentType) GetValue() string {
	return pt.Value
}

func (pt *PaymentType) ToString() string {
	return pt.Value
}

package financial

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type PaymentTerm struct {
	Value string
}

func NewPaymentTerm(value string) (*PaymentTerm, error) {
	paymentTerm := &PaymentTerm{Value: value}
	if paymentTerm.IsValid() {
		return paymentTerm, nil
	}
	return &PaymentTerm{}, dte_errors.NewValidationError("InvalidPattern", "operation_condition", "01-03", value)
}

func NewValidatedPaymentTerm(value string) *PaymentTerm {
	return &PaymentTerm{Value: value}
}

func (p *PaymentTerm) GetValue() string {
	return p.Value
}

func (p *PaymentTerm) IsValid() bool {
	pattern := `^0[1-3]$`
	matched, _ := regexp.MatchString(pattern, p.Value)
	return matched
}

func (p *PaymentTerm) Equals(other interfaces.ValueObject[string]) bool {
	return p.Value == other.GetValue()
}

func (p *PaymentTerm) ToString() string {
	return p.Value
}

package base

import (
	"github.com/badoux/checkmail"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// SkipMXValidation disables live DNS MX record lookup during email validation.
// Set to true in test environments to avoid network calls.
var SkipMXValidation bool

type Email struct {
	Value string `json:"value"`
}

func NewEmail(value string) (*Email, error) {
	email := &Email{Value: value}
	if email.IsValid() {
		return email, nil
	}
	return &Email{}, dte_errors.NewValidationError("InvalidEmail", value)
}

func NewValidatedEmail(value string) *Email {
	return &Email{Value: value}
}

// IsValid validates that the email has correct format, a valid length, and (unless SkipMXValidation is set) a reachable MX record.
func (e *Email) IsValid() bool {
	if err := checkmail.ValidateFormat(e.Value); err != nil {
		return false
	}

	if !SkipMXValidation {
		if err := checkmail.ValidateMX(e.Value); err != nil {
			return false
		}
	}

	return len(e.Value) >= 3 && len(e.Value) <= 100
}

func (e *Email) Equals(other interfaces.ValueObject[string]) bool {
	return e.GetValue() == other.GetValue()
}

func (e *Email) GetValue() string {
	return e.Value
}

func (e *Email) ToString() string {
	return e.Value
}

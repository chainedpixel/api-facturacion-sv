package identification

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ActivityCode struct {
	Value string `json:"value"`
}

func NewActivityCode(value string) (*ActivityCode, error) {
	code := &ActivityCode{Value: value}
	if code.IsValid() {
		return code, nil
	}
	return &ActivityCode{}, dte_errors.NewValidationError("InvalidPattern", "activity_code", "123456", value)
}

func NewValidatedActivityCode(value string) *ActivityCode {
	return &ActivityCode{Value: value}
}

// IsValid validates that the economic activity code has between 2 and 6 digits
func (ac *ActivityCode) IsValid() bool {
	pattern := `^[0-9]{2,6}$`
	matched, _ := regexp.MatchString(pattern, ac.Value)
	return matched
}

func (ac *ActivityCode) Equals(other interfaces.ValueObject[string]) bool {
	return ac.GetValue() == other.GetValue()
}

func (ac *ActivityCode) GetValue() string {
	return ac.Value
}

func (ac *ActivityCode) ToString() string {
	return ac.Value
}

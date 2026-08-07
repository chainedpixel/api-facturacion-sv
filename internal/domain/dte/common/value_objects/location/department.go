package location

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Department struct {
	Value string `json:"value"`
}

func NewDepartment(value string) (*Department, error) {
	dept := &Department{Value: value}
	if dept.IsValid() {
		return dept, nil
	}
	return &Department{}, dte_errors.NewValidationError("InvalidPattern", "Department", "01-14", value)
}

func NewValidatedDepartment(value string) *Department {
	return &Department{Value: value}
}

// IsValid validates that the Department value is a number between 01 and 14 (two digits)
func (d *Department) IsValid() bool {
	pattern := `^0[1-9]|1[0-4]$`
	matched, _ := regexp.MatchString(pattern, d.Value)
	return matched
}

func (d *Department) Equals(other interfaces.ValueObject[string]) bool {
	return d.GetValue() == other.GetValue()
}

func (d *Department) GetValue() string {
	return d.Value
}

func (d *Department) ToString() string {
	return d.Value
}

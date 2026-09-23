package location

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// District represents a municipal district code according to CAT-008
type District struct {
	Value string `json:"value"`
}

// NewDistrict creates and validates a new District value object
func NewDistrict(value string) (*District, error) {
	district := &District{Value: value}
	if district.IsValid() {
		return district, nil
	}
	return &District{}, dte_errors.NewValidationError("InvalidPattern", "District", "01-99", value)
}

// NewValidatedDistrict creates a District value object without validation
func NewValidatedDistrict(value string) *District {
	return &District{Value: value}
}

// IsValid checks if the district value is a valid two-digit code
func (d *District) IsValid() bool {
	matched, _ := regexp.MatchString(`^[0-9]{2}$`, d.Value)
	return matched && d.Value != "00"
}

// Equals checks equality with another ValueObject
func (d *District) Equals(other interfaces.ValueObject[string]) bool {
	return d.GetValue() == other.GetValue()
}

// GetValue returns the string representation of the district code
func (d *District) GetValue() string {
	return d.Value
}

// ToString returns the string representation of the district code
func (d *District) ToString() string {
	return d.Value
}

package location

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Municipality struct {
	Value      string     `json:"value"`
	Department Department `json:"-"`
}

// NewMunicipality creates a new Municipality object with the specified value and department
func NewMunicipality(value string, department Department) (*Municipality, error) {
	mun := &Municipality{
		Value:      value,
		Department: department,
	}
	if mun.IsValid() {
		return mun, nil
	}
	return &Municipality{}, dte_errors.NewValidationError("InvalidMunicipality", value, department.GetValue(), getValidExamplesForDepartment(&department))
}

func NewValidatedMunicipality(value string, department string) *Municipality {
	return &Municipality{
		Value:      value,
		Department: *NewValidatedDepartment(department),
	}
}

// IsValid validates that the Municipality value is a valid number for the specified department
func (m *Municipality) IsValid() bool {
	pattern := m.getMunicipalityPattern()
	matched, _ := regexp.MatchString(pattern, m.Value)
	return matched
}

// getMunicipalityPattern returns the regular expression pattern to validate the Municipality value according to the department
func (m *Municipality) getMunicipalityPattern() string {
	switch m.Department.Value {
	case "01":
		return `^(13|14|15)$`
	case "02":
		return `^(14|15|16|17)$`
	case "03":
		return `^(17|18|19|20)$`
	case "04":
		return `^(34|35|36)$`
	case "05":
		return `^(23|24|25|26|27|28)$`
	case "06":
		return `^(20|21|22|23|24)$`
	case "07":
		return `^(17|18)$`
	case "08":
		return `^(23|24|25)$`
	case "09":
		return `^(10|11)$`
	case "10":
		return `^(14|15)$`
	case "11":
		return `^(24|25|26)$`
	case "12":
		return `^(21|22|23)$`
	case "13":
		return `^(27|28)$`
	case "14":
		return `^(19|20)$`
	default:
		return `^(00)$`
	}
}

func (m *Municipality) Equals(other interfaces.ValueObject[string]) bool {
	return m.GetValue() == other.GetValue()
}

func (m *Municipality) GetValue() string {
	return m.Value
}

func (m *Municipality) ToString() string {
	return m.Value
}

// getValidExamplesForDepartment returns examples of valid municipality codes for the department
func getValidExamplesForDepartment(department *Department) string {
	switch department.GetValue() {
	case "01":
		return "13, 14, 15"
	case "02":
		return "14, 15, 16, 17"
	case "03":
		return "17, 18, 19, 20"
	case "04":
		return "34, 35, 36"
	case "05":
		return "23, 24, 25, 26"
	case "06":
		return "20, 21, 22, 23, 24"
	case "07":
		return "17, 18"
	case "08":
		return "23, 24, 25"
	case "09":
		return "10, 11"
	case "10":
		return "14, 15"
	case "11":
		return "24, 25, 26"
	case "12":
		return "21, 22, 23"
	case "13":
		return "27, 28"
	case "14":
		return "19, 20"
	default:
		return "00"
	}
}

package identification

import (
	"strings"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// DocumentNumber represents the identification document number, an optional attribute of a receiver
type DocumentNumber struct {
	Value string
}

func NewDocumentNumber(value string, dteType string) (*DocumentNumber, error) {

	if dteType == constants.NIT {
		if strings.Contains(value, "-") {
			value = strings.ReplaceAll(value, "-", "")
		}
	}
	value = strings.TrimSpace(value)
	documentNumber := &DocumentNumber{Value: value}
	if documentNumber.IsValid() {
		return documentNumber, nil
	}
	return &DocumentNumber{}, dte_errors.NewValidationError("InvalidDocumentNumber", value)
}

func NewValidatedDocumentNumber(value string) *DocumentNumber {
	return &DocumentNumber{Value: value}
}

func (dn *DocumentNumber) IsValid() bool {
	return len(dn.Value) >= 3 && len(dn.Value) <= 20
}

func (dn *DocumentNumber) GetValue() string {
	return dn.Value
}

func (dn *DocumentNumber) Equals(other interfaces.ValueObject[string]) bool {
	return dn.Value == other.GetValue()
}

func (dn *DocumentNumber) ToString() string {
	return dn.Value
}

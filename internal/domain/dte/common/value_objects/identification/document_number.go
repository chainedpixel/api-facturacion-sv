package identification

import (
	"strings"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
)

// DocumentNumber represents the identification document number, an optional attribute of a receiver
type DocumentNumber struct {
	Value string
}

func NewDocumentNumber(value string, dteType string) (*DocumentNumber, error) {

	// NIT y DUI se envian sin guiones; el DUI debe tener 9 digitos (Anexo IV, campo 39.10)
	if dteType == constants.NIT || dteType == constants.DUI {
		value = strings.ReplaceAll(value, "-", "")
	}
	value = strings.TrimSpace(value)
	documentNumber := &DocumentNumber{Value: value}
	if dteType == constants.DUI && !isNineDigits(value) {
		return &DocumentNumber{}, dte_errors.NewValidationError("InvalidDocumentNumber", value)
	}
	if documentNumber.IsValid() {
		return documentNumber, nil
	}
	return &DocumentNumber{}, dte_errors.NewValidationError("InvalidDocumentNumber", value)
}

func isNineDigits(value string) bool {
	if len(value) != 9 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

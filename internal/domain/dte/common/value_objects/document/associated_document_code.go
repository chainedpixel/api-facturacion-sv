package document

import (
	"strconv"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type AssociatedDocumentCode struct {
	value int
}

func NewAssociatedDocumentCode(value int) (*AssociatedDocumentCode, error) {
	adc := &AssociatedDocumentCode{value: value}
	if adc.IsValid() {
		return adc, nil
	}
	return &AssociatedDocumentCode{}, dte_errors.NewValidationError("InvalidAssociatedDocumentCode", value)
}

func NewValidatedAssociatedDocumentCode(value int) *AssociatedDocumentCode {
	return &AssociatedDocumentCode{value: value}
}

func (adc *AssociatedDocumentCode) IsValid() bool {
	for _, v := range constants.AllowedAssociatedDocumentCodes {
		if adc.value == v {
			return true
		}
	}
	return false
}

func (adc *AssociatedDocumentCode) GetValue() int {
	return adc.value
}

func (adc *AssociatedDocumentCode) Equals(other interfaces.ValueObject[int]) bool {
	return adc.value == other.GetValue()
}

func (adc *AssociatedDocumentCode) ToString() string {
	return strconv.Itoa(adc.value)
}

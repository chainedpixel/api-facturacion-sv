package document

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type InvalidationType struct {
	Value int
}

func NewInvalidationType(value int) (*InvalidationType, error) {
	it := &InvalidationType{Value: value}
	if it.IsValid() {
		return it, nil
	}
	return nil, dte_errors.NewValidationError("InvalidInvalidationType", value)
}

func (it *InvalidationType) IsValid() bool {
	return it.Value >= 1 && it.Value <= 3
}

func (it *InvalidationType) GetValue() int {
	return it.Value
}

func (it *InvalidationType) Equals(other interfaces.ValueObject[int]) bool {
	return it.Value == other.GetValue()
}

func (it *InvalidationType) ToString() string {
	return fmt.Sprintf("%d", it.Value)
}

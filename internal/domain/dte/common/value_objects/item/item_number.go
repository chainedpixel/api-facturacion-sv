package item

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ItemNumber struct {
	Value int
}

func NewItemNumber(value int) (*ItemNumber, error) {
	itemNumber := &ItemNumber{Value: value}
	if itemNumber.IsValid() {
		return itemNumber, nil
	}
	return &ItemNumber{}, dte_errors.NewValidationError("InvalidLength", "item.number", "1 a 2000", fmt.Sprintf("%d", value))
}

func NewValidatedItemNumber(value int) *ItemNumber {
	return &ItemNumber{Value: value}
}

func (in *ItemNumber) GetValue() int {
	return in.Value
}

func (in *ItemNumber) IsValid() bool {
	return in.Value > 0 && in.Value < 2000
}

func (in *ItemNumber) Equals(other interfaces.ValueObject[int]) bool {
	return in.Value == other.GetValue()
}

func (in *ItemNumber) ToString() string {
	return fmt.Sprintf("%d", in.Value)
}

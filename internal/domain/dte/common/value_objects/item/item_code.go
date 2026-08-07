package item

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ItemCode struct {
	Value string
}

func NewItemCode(value string) (*ItemCode, error) {
	itemCode := &ItemCode{Value: value}
	if itemCode.IsValid() {
		return itemCode, nil
	}
	return &ItemCode{}, dte_errors.NewValidationError("InvalidLength", "item.code", "1 a 25", value)
}

func NewValidatedItemCode(value string) *ItemCode {
	return &ItemCode{Value: value}
}

func (ic *ItemCode) GetValue() string {
	return ic.Value
}

func (ic *ItemCode) IsValid() bool {
	return len(ic.Value) > 0 && len(ic.Value) < 25
}

func (ic *ItemCode) Equals(other interfaces.ValueObject[string]) bool {
	return ic.Value == other.GetValue()
}

func (ic *ItemCode) ToString() string {
	return ic.Value
}

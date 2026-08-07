package item

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ItemType struct {
	Value int
}

func NewItemType(value int) (*ItemType, error) {
	itemType := &ItemType{Value: value}
	if itemType.IsValid() {
		return itemType, nil
	}
	return &ItemType{}, dte_errors.NewValidationError("InvalidItemType", value)
}

func NewValidatedItemType(value int) *ItemType {
	return &ItemType{Value: value}
}

func (i *ItemType) GetValue() int {
	return i.Value
}

func (i *ItemType) IsValid() bool {
	for _, allowed := range constants.AllowedItemTypes {
		if i.Value == allowed {
			return true
		}
	}
	return false
}

func (i *ItemType) Equals(other interfaces.ValueObject[int]) bool {
	return i.Value == other.GetValue()
}

func (i *ItemType) ToString() string {
	return fmt.Sprintf("%d", i.Value)
}

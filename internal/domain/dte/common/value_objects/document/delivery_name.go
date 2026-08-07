package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type DeliveryName struct {
	Value string `json:"value"`
}

func NewDeliveryName(value string) (*DeliveryName, error) {
	name := &DeliveryName{Value: value}
	if name.IsValid() {
		return name, nil
	}
	return &DeliveryName{}, dte_errors.NewValidationError("InvalidDeliveryName", value)
}

func NewValidatedDeliveryName(value string) *DeliveryName {
	return &DeliveryName{Value: value}
}

func (n *DeliveryName) IsValid() bool {
	return len(n.Value) >= 1 && len(n.Value) <= 100
}

func (n *DeliveryName) Equals(other interfaces.ValueObject[string]) bool {
	return n.GetValue() == other.GetValue()
}

func (n *DeliveryName) GetValue() string {
	return n.Value
}

func (n *DeliveryName) ToString() string {
	return n.Value
}

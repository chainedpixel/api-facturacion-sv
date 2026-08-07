package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type DeliveryDocument struct {
	Value string `json:"value"`
}

func NewDeliveryDocument(value string) (*DeliveryDocument, error) {
	doc := &DeliveryDocument{Value: value}
	if doc.IsValid() {
		return doc, nil
	}
	return &DeliveryDocument{}, dte_errors.NewValidationError("InvalidDeliveryDocument", value)
}

func NewValidatedDeliveryDocument(value string) *DeliveryDocument {
	return &DeliveryDocument{Value: value}
}

func (d *DeliveryDocument) IsValid() bool {
	return len(d.Value) >= 1 && len(d.Value) <= 25
}

func (d *DeliveryDocument) Equals(other interfaces.ValueObject[string]) bool {
	return d.GetValue() == other.GetValue()
}

func (d *DeliveryDocument) GetValue() string {
	return d.Value
}

func (d *DeliveryDocument) ToString() string {
	return d.Value
}

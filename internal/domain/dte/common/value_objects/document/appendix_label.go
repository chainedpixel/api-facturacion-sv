package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type AppendixLabel struct {
	Value string `json:"value"`
}

func NewAppendixLabel(value string) (*AppendixLabel, error) {
	label := &AppendixLabel{Value: value}
	if label.IsValid() {
		return label, nil
	}
	return &AppendixLabel{}, dte_errors.NewValidationError("InvalidAppendixLabel", value)
}

func NewValidatedAppendixLabel(value string) *AppendixLabel {
	return &AppendixLabel{Value: value}
}

func (l *AppendixLabel) IsValid() bool {
	return len(l.Value) >= 3 && len(l.Value) <= 50
}

func (l *AppendixLabel) Equals(other interfaces.ValueObject[string]) bool {
	return l.Value == other.GetValue()
}

func (l *AppendixLabel) GetValue() string {
	return l.Value
}

func (l *AppendixLabel) ToString() string {
	return l.Value
}

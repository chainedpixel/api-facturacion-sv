package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ContingencyReason struct {
	Value string
}

func NewContingencyReason(value string) (*ContingencyReason, error) {
	reason := &ContingencyReason{Value: value}
	if !reason.IsValid() {
		return nil, dte_errors.NewValidationError("InvalidLength", "contingency reason", "5-150", value)
	}
	return reason, nil
}

func NewValidatedContingencyReason(value string) *ContingencyReason {
	return &ContingencyReason{Value: value}
}

func (r *ContingencyReason) IsValid() bool {
	return len(r.Value) >= 5 && len(r.Value) <= 150
}

func (r *ContingencyReason) ToString() string {
	return r.Value
}

func (r *ContingencyReason) Equals(other interfaces.ValueObject[string]) bool {
	return r.Value == other.GetValue()
}

func (r *ContingencyReason) GetValue() string {
	return r.Value
}

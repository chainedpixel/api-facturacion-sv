package document

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type OperationType struct {
	Value int `json:"value"`
}

func NewOperationType(value int) (*OperationType, error) {
	opType := &OperationType{Value: value}
	if opType.IsValid() {
		return opType, nil
	}
	return &OperationType{}, dte_errors.NewValidationError("InvalidNumberRange", "operation type", "1-2", fmt.Sprintf("%d", value))
}

func NewValidatedOperationType(value int) *OperationType {
	return &OperationType{Value: value}
}

// IsValid validates that the OperationType value is 1 or 2
func (ot *OperationType) IsValid() bool {
	return ot.Value == 1 || ot.Value == 2
}

func (ot *OperationType) Equals(other interfaces.ValueObject[int]) bool {
	return ot.GetValue() == other.GetValue()
}

func (ot *OperationType) GetValue() int {
	return ot.Value
}

func (ot *OperationType) ToString() string {
	return fmt.Sprintf("%d", ot.Value)
}

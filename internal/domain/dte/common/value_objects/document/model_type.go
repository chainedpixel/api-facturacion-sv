package document

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ModelType struct {
	Value int `json:"value"`
}

func NewModelType(value int) (*ModelType, error) {
	modelType := &ModelType{Value: value}
	if modelType.IsValid() {
		return modelType, nil
	}
	return &ModelType{}, dte_errors.NewValidationError("InvalidLength", "model type", "1-2", fmt.Sprintf("%d", value))
}

func NewValidatedModelType(value int) *ModelType {
	return &ModelType{Value: value}
}

// IsValid validates that the ModelType value is 1 or 2
func (mt *ModelType) IsValid() bool {
	return mt.Value == 1 || mt.Value == 2
}

func (mt *ModelType) Equals(other interfaces.ValueObject[int]) bool {
	return mt.GetValue() == other.GetValue()
}

func (mt *ModelType) GetValue() int {
	return mt.Value
}

func (mt *ModelType) ToString() string {
	return fmt.Sprintf("%d", mt.Value)
}

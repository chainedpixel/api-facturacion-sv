package item

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type UnitMeasure struct {
	Value int `json:"value"`
}

func NewUnitMeasure(value int) (*UnitMeasure, error) {
	unitMeasure := &UnitMeasure{Value: value}
	if unitMeasure.IsValid() {
		return unitMeasure, nil
	}
	return &UnitMeasure{}, dte_errors.NewValidationError("InvalidNumberRange", "unit_measure", "1-99", fmt.Sprintf("%d", value))
}

func NewValidatedUnitMeasure(value int) *UnitMeasure {
	return &UnitMeasure{Value: value}
}

func (a *UnitMeasure) IsValid() bool {
	return a.Value >= 1 && a.Value <= 99
}

func (a *UnitMeasure) GetValue() int {
	return a.Value
}

func (a *UnitMeasure) Equals(other interfaces.ValueObject[int]) bool {
	return a.GetValue() == other.GetValue()
}

func (a *UnitMeasure) ToString() string {
	return fmt.Sprintf("%d", a.Value)
}

package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Observation struct {
	Value string `json:"value"`
}

func NewObservation(value string) (*Observation, error) {
	obs := &Observation{Value: value}
	if obs.IsValid() {
		return obs, nil
	}
	return &Observation{}, dte_errors.NewValidationError("InvalidLength", "observation", "1-3000", value)
}

func NewValidatedObservation(value string) *Observation {
	return &Observation{Value: value}
}

func (o *Observation) IsValid() bool {
	return len(o.Value) <= 3000
}

func (o *Observation) Equals(other interfaces.ValueObject[string]) bool {
	return o.GetValue() == other.GetValue()
}

func (o *Observation) GetValue() string {
	return o.Value
}

func (o *Observation) ToString() string {
	return o.Value
}

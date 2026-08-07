package document

import (
	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type Ambient struct {
	Value string `json:"value"`
}

// NewAmbient Creates a new Ambient value object with the ambient value obtained from the environment
func NewAmbient() (*Ambient, error) {
	ambient := &Ambient{
		Value: config.Server.AmbientCode,
	}

	if ambient.IsValid() {
		return ambient, nil
	} else {
		return &Ambient{}, dte_errors.NewValidationError("InvalidAmbientCode", ambient.Value)
	}
}

func NewValidatedAmbient(value string) *Ambient {
	return &Ambient{Value: value}
}

func NewAmbientCustom(value string) (*Ambient, error) {
	ambient := &Ambient{Value: value}
	if ambient.IsValid() {
		return ambient, nil
	} else {
		return &Ambient{}, dte_errors.NewValidationError("InvalidAmbientCode", ambient.Value)
	}
}

// IsValid validates that the Ambient field value is one of the allowed values.
func (a *Ambient) IsValid() bool {
	for _, v := range constants.AllowedAmbientValues {
		if a.Value == v {
			return true
		}
	}
	return false
}

// Equals Compares the Ambient field value with the value of another Ambient value object
func (a *Ambient) Equals(ambient interfaces.ValueObject[string]) bool {
	return a.GetValue() == ambient.GetValue()
}

func (a *Ambient) GetValue() string {
	return a.Value
}

func (a *Ambient) ToString() string { return a.Value }

package identification

import (
	"strings"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/google/uuid"
)

type GenerationCode struct {
	Value string `json:"value"`
}

// NewGenerationCode creates a new GenerationCode with a random UUID
func NewGenerationCode() (*GenerationCode, error) {
	code := &GenerationCode{Value: strings.ToUpper(uuid.New().String())}
	if code.IsValid() {
		return code, nil
	}
	return &GenerationCode{}, dte_errors.NewValidationError("InvalidPattern", "generation_code", "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX", code.Value)
}

func NewValidatedGenerationCode(value string) *GenerationCode {
	return &GenerationCode{Value: value}
}

func (gc *GenerationCode) IsValid() bool {
	_, err := uuid.Parse(gc.Value)
	return err == nil && gc.Value == strings.ToUpper(gc.Value)
}

func (gc *GenerationCode) Equals(other interfaces.ValueObject[string]) bool {
	return gc.GetValue() == other.GetValue()
}

func (gc *GenerationCode) GetValue() string {
	return gc.Value
}

func (gc *GenerationCode) ToString() string {
	return gc.Value
}

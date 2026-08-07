package financial

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type TaxType struct {
	Value string `json:"value"`
}

func NewTaxType(value string) (*TaxType, error) {
	taxType := &TaxType{Value: value}
	if taxType.IsValid() {
		return taxType, nil
	}
	return &TaxType{}, dte_errors.NewValidationError("InvalidTaxType", value)
}

func NewValidatedTaxType(value string) *TaxType {
	return &TaxType{Value: value}
}

func (t *TaxType) GetValue() string {
	if t == nil {
		return ""
	}

	return t.Value
}

func (t *TaxType) IsValid() bool {
	for _, allowed := range constants.AllowedTaxTypes {
		if t.Value == allowed {
			return true
		}
	}
	return false
}

func (t *TaxType) Equals(other interfaces.ValueObject[string]) bool {
	return t.Value == other.GetValue()
}

func (t *TaxType) ToString() string {
	return t.Value
}

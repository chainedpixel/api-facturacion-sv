package document

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type RetentionCode struct {
	Value string `json:"value"`
}

func NewRetentionCode(value string) (*RetentionCode, error) {
	rc := &RetentionCode{Value: value}
	if rc.IsValid() {
		return rc, nil
	}
	return nil, dte_errors.NewValidationError("InvalidRetentionCode", value)
}

func NewValidatedRetentionCode(value string) *RetentionCode {
	return &RetentionCode{Value: value}
}

func (rc *RetentionCode) IsValid() bool {
	if constants.AllowedRetentionCodes[rc.Value] {
		return true
	}
	return false
}

func (rc *RetentionCode) GetValue() string {
	return rc.Value
}

func (rc *RetentionCode) Equals(other interfaces.ValueObject[string]) bool {
	return rc.GetValue() == other.GetValue()
}

func (rc *RetentionCode) ToString() string {
	return rc.Value
}

package document

import (
	"reflect"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type DTEType struct {
	Value string `json:"value"`
}

// NewDTEType creates a new valid electronic document type for emission
func NewDTEType(value string) (*DTEType, error) {
	tipoDte := &DTEType{Value: value}
	if tipoDte.IsValid() {
		return tipoDte, nil
	}
	return &DTEType{}, dte_errors.NewValidationError("InvalidDTEType", value)
}

func NewValidatedDTEType(value string) *DTEType {
	return &DTEType{Value: value}
}

// NewDTETypeForReceiver creates a new valid electronic document type for reception
func NewDTETypeForReceiver(value string) (*DTEType, error) {
	tipoDte := &DTEType{Value: value}
	if tipoDte.IsForReception() {
		return tipoDte, nil
	}
	return &DTEType{}, dte_errors.NewValidationError("InvalidDocumentForReceiver", value)
}

// NewDTETypeForRetention creates a new valid electronic document type for retention
func NewDTETypeForRetention(value string) (*DTEType, error) {
	tipoDte := &DTEType{Value: value}
	if tipoDte.IsForRetention() {
		return tipoDte, nil
	}
	return &DTEType{}, dte_errors.NewValidationError("InvalidDTETypeForRetention", value)
}

// IsValid validates that the value is a string and is a valid electronic document type
func (t *DTEType) IsValid() bool {
	return constants.ValidDTETypes[t.Value]
}

func (t *DTEType) IsForReception() bool {
	for _, v := range constants.ValidReceiverDTETypes {
		if t.Value == v {
			return true
		}
	}
	return false
}

func (t *DTEType) IsForRetention() bool {
	if constants.ValidRetentionDTETypes[t.Value] {
		return true
	}

	return false
}

func (t *DTEType) Equals(other interfaces.ValueObject[string]) bool {
	return t.GetValue() == other.GetValue()
}

func (t *DTEType) ToString() string {
	return reflect.ValueOf(t.Value).String()
}

func (t *DTEType) GetValue() string {
	return t.Value
}

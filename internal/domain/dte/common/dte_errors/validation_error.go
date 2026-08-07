package dte_errors

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
)

type ValidationError struct {
	ErrorType string
	Message   string
}

// NewValidationError Creates a new validation error with the error type and provided parameters
func NewValidationError(errorType string, params ...interface{}) *ValidationError {
	message := constants.GetErrorMessage(errorType, params...)
	return &ValidationError{ErrorType: errorType, Message: message}
}

func NewFormattedValidationError(err error) *ValidationError {
	return &ValidationError{ErrorType: "", Message: err.Error()}
}

// Error implements the error interface for the validation error
func (v *ValidationError) Error() string {
	return fmt.Sprintf("%s", v.Message)
}

// GetType returns the validation error type
func (v *ValidationError) GetType() string {
	if v == nil {
		return "UnknownError"
	}

	return v.ErrorType
}

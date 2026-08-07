package dte_errors

import "strings"

// CompositeError represents a composite error that contains multiple errors
type CompositeError struct {
	Errors []error
}

func NewCompositeError(errors ...error) *CompositeError {
	return &CompositeError{
		Errors: errors,
	}
}

// Error implements the error interface for the composite error to surface the primary error at the DTEDocument level and validation errors from value objects
func (e *CompositeError) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

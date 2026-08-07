package dte_errors

import (
	"fmt"
	"strings"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
)

type DTEError struct {
	ValidationErrors []error
	BusinessErrors   []*DTEError
	ErrorType        string
	Message          string
	Code             string
}

// getDTEErrorMessage retrieves the DTE error message with the provided parameters
func getDTEErrorMessage(errorType string, params ...interface{}) string {
	return constants.GetErrorMessage(errorType, params...)
}

// NewDTEErrorSimple Creates a new DTE error with the error type and provided parameters, without validation errors
func NewDTEErrorSimple(errorType string, params ...interface{}) *DTEError {
	return &DTEError{
		ValidationErrors: nil,
		ErrorType:        errorType,
		Message:          getDTEErrorMessage(errorType, params...),
		Code:             strings.ToUpper(errorType),
	}
}

// NewDTEErrorComposite Creates a new DTE error with the provided business errors and groups them into a single DTE error
func NewDTEErrorComposite(businessErrors []*DTEError) *DTEError {
	var messages []string
	var validErrors []*DTEError

	for _, err := range businessErrors {
		if err != nil {
			validErrors = append(validErrors, err)
			messages = append(messages, err.Error())
		}
	}

	return &DTEError{
		BusinessErrors: validErrors,
		Message:        strings.Join(messages, "; "),
		Code:           "MANY_ERRORS",
	}
}

// Error implements the error interface for the DTE error
func (e *DTEError) Error() string {
	if e == nil {
		return "Unknown DTE error"
	}

	return e.Message
}

// GetValidationErrorsString retrieves the validation errors associated with the DTE error, if any exist
func (e *DTEError) GetValidationErrorsString() []string {
	var messages []string
	for _, err := range e.ValidationErrors {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if e.BusinessErrors != nil {
		for _, err := range e.BusinessErrors {
			if err != nil {
				messages = append(messages, err.Message)
			}
		}
	}

	return messages
}

// GetCode retrieves the error code
func (e *DTEError) GetCode() string {
	if e == nil {
		return "UNKNOWN_DTE_ERROR"
	}

	if e.Code != "" {
		return e.Code
	}

	return strings.ToUpper(e.ErrorType)
}

// GetMessage retrieves the translated error message
func (e *DTEError) GetMessage() string {
	if e == nil {
		return "Unknown DTE error"
	}

	if len(e.ValidationErrors) > 0 || len(e.BusinessErrors) > 0 {
		return config.Translate("service_errors.FailedToCreateDTE")
	}

	key := fmt.Sprintf("service_errors.%s", e.ErrorType)
	translated := config.Translate(key)

	if translated == key {
		return e.Message
	}

	return translated
}

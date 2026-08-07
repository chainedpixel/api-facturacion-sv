package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/stretchr/testify/assert"
)

func AssertErrorCode(t *testing.T, err error, expectedCode string) {
	expectedCode = strings.ToLower(expectedCode)
	var validationErr *dte_errors.ValidationError
	var serviceErr *shared_error.ServiceError
	var dteError *dte_errors.DTEError
	var actualErr string

	if errors.As(err, &validationErr) {
		t.Logf("Validation error detected %v", validationErr)
		actualErr = strings.ToLower(validationErr.GetType())
	} else if errors.As(err, &serviceErr) {
		t.Logf("Service error detected %v", serviceErr)
		actualErr = strings.ToLower(serviceErr.GetCode())
	} else if errors.As(err, &dteError) {
		t.Logf("DTE error detected %v", dteError)
		actualErr = strings.ToLower(dteError.GetCode())
	} else {
		t.Errorf("Error type not recognized, expected validation or service error with code %s", expectedCode)
	}

	assert.Equal(t, expectedCode, actualErr, "Error code does not match")
}

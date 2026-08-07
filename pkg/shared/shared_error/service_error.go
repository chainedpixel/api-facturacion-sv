package shared_error

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
)

type ServiceError struct {
	Type      string
	Operation string
	Message   string
	Code      string
	Err       error
}

func (e *ServiceError) Error() string {

	if e.Err == nil {
		if config.Server.Debug {
			return fmt.Sprintf("[%s] %s", e.Code, e.Message)
		}
		return fmt.Sprintf("%s", e.Message)
	}

	if config.Server.Debug {
		return fmt.Sprintf("[%s] %s: %s | cause: %v", e.Type, e.Operation, e.Message, e.Err)
	}

	return fmt.Sprintf("%s -> %s", e.Message, e.Err)
}

func (e *ServiceError) GetErrError() []string {
	if e.Err == nil {
		return []string{config.Translate("service_errors.NoDetailsAvailable")}
	}

	rawDetails := strings.Split(e.Err.Error(), ";")
	var details []string
	for _, detail := range rawDetails {
		trimmed := strings.TrimSpace(detail)
		if trimmed != "" {
			details = append(details, trimmed)
		}
	}
	return details
}

// NewGeneralServiceError creates a new general service error with the service type, operation, message and error.
func NewGeneralServiceError(serviceType, op, msg string, err error) *ServiceError {
	return &ServiceError{
		Type:      serviceType,
		Operation: op,
		Message:   msg,
		Err:       err,
	}
}

func NewFormattedGeneralServiceError(serviceType, op, code string, args ...interface{}) *ServiceError {
	message := config.Translate(fmt.Sprintf("service_errors.%s", strings.ToLower(code)), args)
	return &ServiceError{
		Type:      serviceType,
		Operation: op,
		Message:   message,
		Code:      code,
	}
}

func NewFormattedGeneralServiceWithError(serviceType, op string, err error, code string, args ...interface{}) *ServiceError {
	message := config.Translate(fmt.Sprintf("service_errors.%s", strings.ToLower(code)), args)
	return &ServiceError{
		Type:      serviceType,
		Operation: op,
		Message:   message,
		Code:      code,
		Err:       err,
	}
}

func (e *ServiceError) GetCode() string {
	if e.Err != nil {
		var dteErr *dte_errors.ValidationError
		if errors.As(e.Err, &dteErr) {
			return dteErr.GetType()
		}
	}

	if e.Code != "" {
		return e.Code
	}

	return strings.ToLower(e.Type)
}

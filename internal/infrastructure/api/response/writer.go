package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type errorType string

const (
	errorValidation errorType = "VALIDATION"
	errorBusiness   errorType = "BUSINESS"
	errorSystem     errorType = "SYSTEM"
)

type ResponseWriter struct{}

// NewResponseWriter creates a new instance of ResponseWriter
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{}
}

// Success sends a successful response with the provided status code and data.
func (w *ResponseWriter) Success(rw http.ResponseWriter, status int, data interface{}, options *SuccessOptions) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)

	apiResponse := &APIResponse{
		Success: true,
		Data:    data,
	}

	if options == nil {
		json.NewEncoder(rw).Encode(apiResponse)
		return
	}

	qrLink := GenerateQRLink(options.Ambient, options.GenerationCode, options.EmissionDate)

	json.NewEncoder(rw).Encode(APIDTEResponse{
		Success:        true,
		ReceptionStamp: options.ReceptionStamp,
		QRLink:         &qrLink,
		Data:           data,
	})
}

// Error sends an error response with the provided status code and message.
func (w *ResponseWriter) Error(rw http.ResponseWriter, status int, message string, details []string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(APIErrorResponse{
		Error: &APIError{
			Message: message,
			Details: details,
			Code:    deriveErrorCode(status),
		},
	})
}

// HandleError handles the different error types and sends an error response with the corresponding status code and message.
func (w *ResponseWriter) HandleError(rw http.ResponseWriter, err error) {
	rw.Header().Set("Content-Type", "application/json")

	logs.Error("Error processing request", map[string]interface{}{
		"error_type": getErrorType(err),
		"error":      err.Error(),
	})

	switch errorType := getErrorType(err); errorType {
	case errorValidation:
		w.handleValidationError(rw, err)
	case errorBusiness:
		w.handleBusinessError(rw, err)
	default:
		w.handleSystemError(rw, err)
	}
}

// GenerateQRLink Generates a link to look up the invoice on the Hacienda website
func GenerateQRLink(ambiente, codGeneracion string, fechaEmision time.Time) string {
	return fmt.Sprintf("https://admin.factura.gob.sv/consultaPublica?ambiente=%s&codGen=%s&fechaEmi=%s",
		ambiente, codGeneracion, fechaEmision.Format("2006-01-02"))
}

// handleValidationError handles validation errors and sends an error response with the corresponding status code and message.
func (w *ResponseWriter) handleValidationError(rw http.ResponseWriter, err error) {
	var dteErr *dte_errors.DTEError
	if errors.As(err, &dteErr) {
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Message: dteErr.GetMessage(),
				Details: dteErr.GetValidationErrorsString(),
				Code:    dteErr.GetCode(),
			},
		})
		return
	}

	var validationErr *dte_errors.ValidationError
	if errors.As(err, &validationErr) {
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Message: validationErr.Error(),
				Details: []string{config.Translate("service_errors.NoDetailsAvailable")},
				Code:    strings.ToUpper(validationErr.GetType()),
			},
		})
	}
}

// handleBusinessError handles business errors and sends an error response with the corresponding status code and message.
func (w *ResponseWriter) handleBusinessError(rw http.ResponseWriter, err error) {
	var haciendaErr *hacienda_error.HaciendaResponseError
	if errors.As(err, &haciendaErr) {
		details := []string{
			fmt.Sprintf("State: %s", haciendaErr.Status),
			fmt.Sprintf("Processed at: %s", haciendaErr.ProcessedAt),
		}

		if len(haciendaErr.Observations) > 0 {
			details = append(details, haciendaErr.Observations...)
		}

		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Message: haciendaErr.Description,
				Code:    fmt.Sprintf("HACIENDA_%s", haciendaErr.Code),
				Details: details,
			},
		})
		return
	}

	var svcErr *shared_error.ServiceError
	if errors.As(err, &svcErr) {
		rw.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(rw).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Message: svcErr.Message,
				Details: svcErr.GetErrError(),
				Code:    strings.ToUpper(svcErr.GetCode()),
			},
		})
		return
	}

	var httpErr *hacienda_error.HTTPResponseError
	if errors.As(err, &httpErr) {
		rw.WriteHeader(httpErr.StatusCode)
		detail := config.Translate("service_errors.ContingencyActiveTransmission")
		json.NewEncoder(rw).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Message: httpErr.Error(),
				Details: []string{detail},
				Code:    fmt.Sprintf("HACIENDA_%d", httpErr.StatusCode),
			},
		})
		return
	}

	rw.WriteHeader(http.StatusBadRequest)

	json.NewEncoder(rw).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Message: err.Error(),
			Code:    "BUSINESS_ERROR",
		},
	})
}

// handleSystemError handles system errors and sends an error response with the corresponding status code and message.
func (w *ResponseWriter) handleSystemError(rw http.ResponseWriter, err error) {
	rw.WriteHeader(http.StatusInternalServerError)
	message := config.Translate("validation_errors.ServerError")
	json.NewEncoder(rw).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Message: message,
			Code:    "SYSTEM_ERROR",
		},
	})
}

// deriveErrorCode derives the error code according to the provided status and message.
func deriveErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	case http.StatusRequestTimeout:
		return "REQUEST_TIMEOUT"
	default:
		return "UNKNOWN_ERROR"
	}
}

// getErrorType retrieves the error type according to the provided error type.
func getErrorType(err error) errorType {
	switch err.(type) {
	case *dte_errors.DTEError, *dte_errors.ValidationError:
		return errorValidation
	case *shared_error.ServiceError, *hacienda_error.HaciendaResponseError, *hacienda_error.HTTPResponseError:
		return errorBusiness
	default:
		return errorSystem
	}
}

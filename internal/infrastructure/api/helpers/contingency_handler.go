package helpers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type ContingencyHandler struct {
	contingencyService contingency.ContingencyManager
}

type ContingencyResult struct {
	ContingencyType   int8
	ContingencyReason string
	ShouldRetry       bool
	RetryConfig       *RetryConfig
}

type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

type ErrClassification struct {
	Type       int8
	Reason     string
	Retryable  bool
	RetryDelay time.Duration
}

func NewContingencyHandler(
	contingencyService contingency.ContingencyManager,
) *ContingencyHandler {
	return &ContingencyHandler{
		contingencyService: contingencyService,
	}
}

func (ch *ContingencyHandler) HandleContingency(ctx context.Context,
	document interface{}, dteType string, err error) (*int8, *string) {

	logs.Info("Starting contingency handling", map[string]interface{}{
		"dteType": dteType,
		"error":   err.Error(),
	})

	if !ch.shouldHandleAsContingency(err) {
		logs.Info("Error does not require contingency handling")
		return nil, nil
	}

	result := ch.classifyError(err)

	if err := ch.storeForContingency(ctx, document, dteType, result); err != nil {
		logs.Error("CONTINGENCY SERVICE CRITICAL ERROR - Failed to store document in contingency", map[string]interface{}{"error": err.Error()})
		return nil, nil
	}

	logs.Info("Contingency handling finished", map[string]interface{}{"dteType": dteType, "contingencyType": result.ContingencyType})
	return &result.ContingencyType, &result.ContingencyReason
}

func (ch *ContingencyHandler) shouldHandleAsContingency(err error) bool {
	var validationErr *dte_errors.ValidationError
	if errors.As(err, &validationErr) {
		return false
	}

	var haciendaErr *hacienda_error.HaciendaResponseError
	if errors.As(err, &haciendaErr) {
		if haciendaErr.Status == "RECHAZADO" {
			return false
		}
	}

	var generalErr *shared_error.ServiceError
	if errors.As(err, &generalErr) {
		return false
	}

	var businessErr *dte_errors.DTEError
	if errors.As(err, &businessErr) {
		return false
	}

	return true
}

func (ch *ContingencyHandler) storeForContingency(
	ctx context.Context,
	document interface{},
	dteType string,
	result ContingencyResult,
) error {

	err := ch.contingencyService.StoreDocumentInContingency(
		ctx,
		document,
		dteType,
		result.ContingencyType,
		result.ContingencyReason,
	)

	if err != nil {
		logs.Error("Failed to store document in contingency", map[string]interface{}{
			"error":           err.Error(),
			"dteType":         dteType,
			"contingencyType": result.ContingencyType,
		})
		return err
	}

	logs.Info("Document stored in contingency successfully", map[string]interface{}{
		"dteType":         dteType,
		"contingencyType": result.ContingencyType,
	})

	return nil
}

func (ch *ContingencyHandler) classifyError(err error) ContingencyResult {
	var haciendaErr *hacienda_error.HaciendaResponseError
	if errors.As(err, &haciendaErr) {
		return ch.classifyHaciendaError(haciendaErr)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) && !containsAny(strings.ToLower(err.Error()), []string{"signer service", "redis"}) {
		return ch.classifyNetworkError(netErr)
	}

	if isContextError(err) {
		return ch.handleContextError(err)
	}

	var httpErr *hacienda_error.HTTPResponseError
	if errors.As(err, &httpErr) {
		return ch.classifyHTTPError(httpErr)
	}

	return ch.defaultErrorClassification(err)
}

func (ch *ContingencyHandler) classifyHaciendaError(err *hacienda_error.HaciendaResponseError) ContingencyResult {
	classification := ch.getHaciendaErrorClassification(err)

	return ContingencyResult{
		ContingencyType:   classification.Type,
		ContingencyReason: constants.GetContingencyReason(classification.Type),
		ShouldRetry:       classification.Retryable,
		RetryConfig: &RetryConfig{
			MaxAttempts:     3,
			InitialInterval: classification.RetryDelay,
			MaxInterval:     30 * time.Minute,
		},
	}
}

func (ch *ContingencyHandler) classifyNetworkError(netErr *net.OpError) ContingencyResult {
	errMsg := strings.ToLower(netErr.Error())

	switch {
	case containsAny(errMsg, []string{"connection refused", "connection reset", "broken pipe"}):
		return ContingencyResult{
			ContingencyType:   constants.FallaConexionSistema,
			ContingencyReason: constants.GetContingencyReason(constants.FallaConexionSistema),
			ShouldRetry:       true,
			RetryConfig: &RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 30 * time.Second,
				MaxInterval:     5 * time.Minute,
			},
		}

	case containsAny(errMsg, []string{"no route to host", "network unreachable", "i/o timeout", "no such host"}):
		return ContingencyResult{
			ContingencyType:   constants.FallaServicioInternet,
			ContingencyReason: constants.GetContingencyReason(constants.FallaServicioInternet),
			ShouldRetry:       true,
			RetryConfig: &RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 1 * time.Minute,
				MaxInterval:     10 * time.Minute,
			},
		}

	default:
		return ContingencyResult{
			ContingencyType:   constants.FallaConexionSistema,
			ContingencyReason: constants.GetContingencyReason(constants.FallaConexionSistema),
			ShouldRetry:       true,
			RetryConfig: &RetryConfig{
				MaxAttempts:     2,
				InitialInterval: 1 * time.Minute,
				MaxInterval:     5 * time.Minute,
			},
		}
	}
}

func (ch *ContingencyHandler) handleContextError(err error) ContingencyResult {
	return ContingencyResult{
		ContingencyType:   constants.NoDisponibilidadMH,
		ContingencyReason: constants.GetContingencyReason(constants.NoDisponibilidadMH),
		ShouldRetry:       true,
		RetryConfig: &RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 2 * time.Minute,
			MaxInterval:     15 * time.Minute,
		},
	}
}

func (ch *ContingencyHandler) classifyHTTPError(httpErr *hacienda_error.HTTPResponseError) ContingencyResult {
	switch httpErr.StatusCode {
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return ContingencyResult{
			ContingencyType:   constants.NoDisponibilidadMH,
			ContingencyReason: constants.GetContingencyReason(constants.NoDisponibilidadMH),
			ShouldRetry:       true,
			RetryConfig: &RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 5 * time.Minute,
				MaxInterval:     30 * time.Minute,
			},
		}
	default:
		return ch.defaultErrorClassification(httpErr)
	}
}

func (ch *ContingencyHandler) defaultErrorClassification(err error) ContingencyResult {
	return ContingencyResult{
		ContingencyType:   constants.OtroMotivo,
		ContingencyReason: constants.GetContingencyReason(constants.OtroMotivo),
		ShouldRetry:       false,
	}
}

func (ch *ContingencyHandler) getHaciendaErrorClassification(err *hacienda_error.HaciendaResponseError) ErrClassification {
	switch err.StatusCode {
	case http.StatusServiceUnavailable, http.StatusBadGateway:
		return ErrClassification{
			Type:       constants.NoDisponibilidadMH,
			Reason:     "Servicio de MH temporalmente no disponible",
			Retryable:  true,
			RetryDelay: 5 * time.Minute,
		}

	case http.StatusUnauthorized, http.StatusForbidden:
		if strings.Contains(strings.ToLower(err.Description), "token") {
			return ErrClassification{
				Type:       constants.FallaConexionSistema,
				Reason:     "Error de autenticación con MH - Token inválido",
				Retryable:  true,
				RetryDelay: 1 * time.Minute,
			}
		}
		return ErrClassification{
			Type:      constants.OtroMotivo,
			Reason:    fmt.Sprintf("Error de autorización: %s", err.Description),
			Retryable: false,
		}

	case http.StatusTooManyRequests:
		return ErrClassification{
			Type:       constants.NoDisponibilidadMH,
			Reason:     "Límite de peticiones alcanzado",
			Retryable:  true,
			RetryDelay: 15 * time.Minute,
		}

	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return ErrClassification{
			Type:       constants.NoDisponibilidadMH,
			Reason:     "Timeout en respuesta de MH",
			Retryable:  true,
			RetryDelay: 2 * time.Minute,
		}
	}

	return ch.classifyByErrorDescription(err)
}

func (ch *ContingencyHandler) classifyByErrorDescription(err *hacienda_error.HaciendaResponseError) ErrClassification {
	errMsg := strings.ToLower(err.Description)

	switch {
	case containsAny(errMsg, []string{"mantenimiento", "maintenance"}):
		return ErrClassification{
			Type:       constants.NoDisponibilidadMH,
			Reason:     "Sistema en mantenimiento",
			Retryable:  true,
			RetryDelay: 30 * time.Minute,
		}

	case containsAny(errMsg, []string{"sobrecarga", "overload"}):
		return ErrClassification{
			Type:       constants.NoDisponibilidadMH,
			Reason:     "Sistema sobrecargado",
			Retryable:  true,
			RetryDelay: 10 * time.Minute,
		}

	case containsAny(errMsg, []string{"validación", "validation"}):
		return ErrClassification{
			Type:      constants.OtroMotivo,
			Reason:    fmt.Sprintf("Error de validación: %s", err.Description),
			Retryable: false,
		}
	}

	ch.logUnhandledError(err)

	return ErrClassification{
		Type:      constants.OtroMotivo,
		Reason:    fmt.Sprintf("Error no clasificado: [%s] %s", err.Code, err.Description),
		Retryable: false,
	}
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func isContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

func (ch *ContingencyHandler) logUnhandledError(err *hacienda_error.HaciendaResponseError) {
	logs.Warn("Unhandled Hacienda error", map[string]interface{}{
		"statusCode": err.StatusCode,
		"code":       err.Code,
		"message":    err.Description,
	})
}

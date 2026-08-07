package handlers

import (
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/metrics"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type MetricsHandler struct {
	metricsManager metrics.MetricsManager
	responseWriter *response.ResponseWriter
}

func NewMetricsHandler(metricsManager metrics.MetricsManager) *MetricsHandler {
	return &MetricsHandler{
		metricsManager: metricsManager,
		responseWriter: response.NewResponseWriter(),
	}
}

// GetEndpointMetrics returns request metrics scoped to the caller's NIT, optionally filtered by endpoint and method.
func (h *MetricsHandler) GetEndpointMetrics(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*models.AuthClaims)

	endpoint := r.URL.Query().Get("endpoint")
	method := r.URL.Query().Get("method")

	if endpoint == "" || method == "" {
		allMetrics, err := h.metricsManager.GetAllMetricsEndpoint(claims.NIT)
		if err != nil {
			logs.Error("Failed to get all endpoint endpointMetrics", map[string]interface{}{
				"error":     err.Error(),
				"systemNIT": claims.NIT,
			})
			h.responseWriter.Error(w, http.StatusInternalServerError, "Failed to get endpoint endpointMetrics", nil)
			return
		}
		h.responseWriter.Success(w, http.StatusOK, allMetrics, nil)
		return
	}

	endpointMetrics, err := h.metricsManager.GetEndpointMetrics(claims.NIT, method, endpoint)
	if err != nil {
		logs.Error("Failed to get endpoint endpointMetrics", map[string]interface{}{
			"error":     err.Error(),
			"systemNIT": claims.NIT,
			"endpoint":  endpoint,
			"method":    method,
		})
		h.responseWriter.Error(w, http.StatusInternalServerError, "Failed to get endpoint endpointMetrics", nil)
		return
	}

	h.responseWriter.Success(w, http.StatusOK, endpointMetrics, nil)
}

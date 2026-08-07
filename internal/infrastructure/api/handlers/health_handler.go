package handlers

import (
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type HealthHandler struct {
	healthManager  health.HealthManager
	responseWriter *response.ResponseWriter
}

func NewHealthHandler(checkHealthUseCase health.HealthManager) *HealthHandler {
	return &HealthHandler{
		healthManager:  checkHealthUseCase,
		responseWriter: response.NewResponseWriter(),
	}
}

// CheckHealth reports the readiness of the core dependencies (DB, cache, Hacienda).
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	logs.Info("Starting health check")
	defer logs.Info("Health check finished")

	status, err := h.healthManager.CheckHealth()
	if err != nil {
		logs.Error("Health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.responseWriter.Error(w, http.StatusInternalServerError, "Health check failed", []string{err.Error()})
		return
	}

	h.responseWriter.Success(w, http.StatusOK, status, nil)
}

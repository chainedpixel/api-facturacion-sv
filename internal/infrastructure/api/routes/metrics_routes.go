package routes

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/gorilla/mux"
)

func RegisterMetricsRoutes(router *mux.Router, handler *handlers.MetricsHandler) {
	router.HandleFunc("/metrics", handler.GetEndpointMetrics).Methods("GET")
}

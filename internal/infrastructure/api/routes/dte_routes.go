package routes

import (
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/handlers"

	"net/http"

	"github.com/gorilla/mux"
)

func RegisterDTERoutes(r *mux.Router, h *handlers.DTEHandler) {
	for path, _ := range h.GenericHandler.GetDocumentConfigs() {
		r.HandleFunc(path, h.GenericHandler.HandleCreate).Methods(http.MethodPost)
	}

	r.HandleFunc("/dte/invalidation", h.InvalidateDocument).Methods(http.MethodPost)
	r.HandleFunc("/dte/{id}", h.GetByGenerationCode).Methods(http.MethodGet)
	r.HandleFunc("/dte", h.GetAll).Methods(http.MethodGet)
}

package routes

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/gorilla/mux"
)

func RegisterPublicAuthRoutes(r *mux.Router, h *handlers.AuthHandler) {
	r.HandleFunc("/auth/register", h.Register).Methods("POST")
	r.HandleFunc("/auth/login", h.Login).Methods("POST")
}

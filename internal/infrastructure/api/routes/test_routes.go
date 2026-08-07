package routes

import (
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/handlers"
	"github.com/gorilla/mux"
)

func RegisterTestRoutes(router *mux.Router, testHandler *handlers.TestHandler) {
	router.HandleFunc("/test", testHandler.RunSystemTest).Methods("GET")
}

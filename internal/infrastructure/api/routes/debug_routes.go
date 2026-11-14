package routes

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/gorilla/mux"
)

func RegisterDebugRoutes(router *mux.Router, debugHandler *handlers.DebugNotifyHandler) {
	router.HandleFunc("/debug/notify-test", debugHandler.SendTestAlert).Methods("POST")
}

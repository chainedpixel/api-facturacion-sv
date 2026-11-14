package main

import (
	"os"

	"github.com/chainedpixel/api-facturacion-sv/internal/bootstrap"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
)

func main() {
	app := bootstrap.NewApplication()
	if err := app.Initialize(); err != nil {
		logs.Fatal("Failed to initialize application", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	if err := app.Start(); err != nil {
		logs.Fatal("Application error", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
}

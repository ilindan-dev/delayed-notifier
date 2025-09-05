package main

import (
	"github.com/ilindan-dev/delayed-notifier/internal/app"
	"go.uber.org/fx"
)

// main is the entry point for the API server application.
func main() {
	// We create and run the Fx application specifically for the API.
	fx.New(app.APIModule).Run()
}

package main

import (
	"github.com/ilindan-dev/delayed-notifier/internal/app"
	"go.uber.org/fx"
)

// main is the entry point for the background worker application.
func main() {
	fx.New(app.WorkerModule).Run()
}

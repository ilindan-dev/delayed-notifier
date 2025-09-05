// Package logger provides a configured zerolog instance.
package logger

import (
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/rs/zerolog"
	"os"
)

// NewLogger creates a new configured instance of zerolog.Logger.
// It reads the log level from the config and adds default fields like service name and caller.
func NewLogger(cfg *config.Config) (*zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(cfg.Logger.Level)
	if err != nil {
		// Default to info level if config is invalid or missing
		level = zerolog.InfoLevel
	}

	// For local development, a pretty console output is much more readable.
	// For production, you'd typically remove ConsoleWriter to get pure JSON.
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr}

	// Create a logger with a predefined context.
	logger := zerolog.New(consoleWriter).With().
		Timestamp().                        // Adds "time" field
		Str("service", "delayed-notifier"). // Adds "service" field for context
		Caller().                           // Adds "caller":"/path/to/file.go:line"
		Logger().
		Level(level) // Set the minimum log level

	return &logger, nil
}

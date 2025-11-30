package config

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger creates a new logger based on the configuration.
func NewLogger(cfg LoggerConfig) zerolog.Logger {
	// Set log level
	var level zerolog.Level
	switch cfg.Level {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	// Configure output format
	var logger zerolog.Logger
	if cfg.Format == "console" {
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return logger
}

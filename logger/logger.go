package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds the configuration for the logger.
type Config struct {
	// Level is the logging level. It can be "trace", "debug", "info", "warn", "error", "fatal", or "panic".
	//
	// Default: info
	Level string

	// IsDevelopment indicates whether the application is in development mode.
	//
	// Default: false
	IsDevelopment bool
}

// Default config values
const (
	DefaultLevel = "info"
)

// Logger is a wrapper around zerolog to provide a structured logger.
type Logger struct {
	config Config
}

// setConfig sets the configuration for the logger.
func (l *Logger) setConfig(config Config) {
	l.config = config

	if l.config.Level == "" {
		l.config.Level = DefaultLevel
	}
}

// New creates a new Logger instance with the provided configuration.
func New(config Config) *Logger {
	logger := &Logger{
		config: Config{},
	}
	logger.setConfig(config)
	return logger
}

// Setup initializes the logger with the provided configuration.
func (l *Logger) Setup() error {
	level, err := zerolog.ParseLevel(l.config.Level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	if l.config.IsDevelopment {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).
			With().
			Timestamp().
			Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).
			With().
			Timestamp().
			Logger()
	}

	return nil
}

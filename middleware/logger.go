package middleware

import (
	"slices"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggerConfig holds the configuration for the logger middleware
type LoggerConfig struct {
	// ExcludePaths is a list of paths to exclude from logging
	ExcludePaths []string `json:"exclude_paths"`

	// AppName is the name of the application
	AppName string `json:"app_name"`

	// Version is the version of the application
	Version string `json:"version"`
}

const (
	defaultAppName = "myapp"
	defaultVersion = "1.0.0"
)

// Logger is a middleware function that logs HTTP requests and responses
func Logger(config LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.AppName == "" {
			config.AppName = defaultAppName
		}
		if config.Version == "" {
			config.Version = defaultVersion
		}
		if config.ExcludePaths == nil {
			config.ExcludePaths = []string{}
		}

		if slices.Contains(config.ExcludePaths, c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Determine log level based on status code
		level := zerolog.InfoLevel
		if statusCode >= 500 {
			level = zerolog.ErrorLevel
		} else if statusCode >= 400 {
			level = zerolog.WarnLevel
		}

		// Optional error message
		var errorMessage string
		if len(c.Errors) > 0 {
			errorMessage = c.Errors.String()
		}

		// Log entry
		log.WithLevel(level).
			Str("request_id", requestid.Get(c)).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", statusCode).
			Int("latency_ms", int(duration.Milliseconds())).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Str("error_message", errorMessage).
			Str("service_name", config.AppName).
			Str("version", config.Version).
			Msg("HTTP Request")
	}
}

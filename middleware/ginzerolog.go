package middleware

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// GinZerologConfig defines the config for ginzerolog middleware.
type GinZerologConfig struct {
	// Next defines a function to skip this middleware when returned true.
	// Default: nil
	Next func(*gin.Context) bool

	// Logger is a custom zerolog logger instance.
	// Default: nil
	Logger *zerolog.Logger

	// GetLogger is a function that returns a custom zerolog logger for each request.
	// If it's defined, the returned logger will replace the Logger value.
	// Default: nil
	GetLogger func(*gin.Context) zerolog.Logger

	// Fields defines the fields to include in the log.
	// Default: []string{"ip", "latency", "status", "method", "url", "error"}
	Fields []string

	// WrapHeaders determines if headers should be wrapped in a dictionary.
	// If false: {"method":"POST", "header-key":"header value"}
	// If true: {"method":"POST", "reqHeaders": {"header-key":"header value"}}
	// Default: false
	WrapHeaders bool

	// FieldsSnakeCase determines if field names should use snake_case.
	// Default: false
	FieldsSnakeCase bool

	// Messages defines custom response messages.
	// Default: []string{"Server error", "Client error", "Success"}
	Messages []string

	// Levels defines custom response levels.
	// Default: []zerolog.Level{zerolog.ErrorLevel, zerolog.WarnLevel, zerolog.InfoLevel}
	Levels []zerolog.Level

	// SkipURIs defines the URIs to skip logging.
	// Default: []string{}
	SkipURIs []string

	// GetResBody defines a function to get response body when return non-nil.
	// This is useful when using compress middleware, where resBody is unreadable.
	// Default: nil
	GetResBody func(*gin.Context) []byte
}

// Default field names
const (
	FieldLatency       = "latency"
	FieldStatus        = "status"
	FieldMethod        = "method"
	FieldURL           = "url"
	FieldError         = "error"
	FieldTime          = "time"
	FieldPID           = "pid"
	FieldIP            = "ip"
	FieldIPs           = "ips"
	FieldHost          = "host"
	FieldPath          = "path"
	FieldProtocol      = "protocol"
	FieldPort          = "port"
	FieldUA            = "ua"
	FieldReferer       = "referer"
	FieldResBody       = "resBody"
	FieldQueryParams   = "queryParams"
	FieldBody          = "body"
	FieldBytesSent     = "bytesSent"
	FieldBytesReceived = "bytesReceived"
	FieldRequestID     = "requestId"
	FieldReqHeaders    = "reqHeaders"
	FieldResHeaders    = "resHeaders"
	FieldRoute         = "route"
)

// Snake case field names
const (
	FieldLatencySnake       = "latency"
	FieldStatusSnake        = "status"
	FieldMethodSnake        = "method"
	FieldURLSnake           = "url"
	FieldErrorSnake         = "error"
	FieldTimeSnake          = "time"
	FieldPIDSnake           = "pid"
	FieldIPSnake            = "ip"
	FieldIPsSnake           = "ips"
	FieldHostSnake          = "host"
	FieldPathSnake          = "path"
	FieldProtocolSnake      = "protocol"
	FieldPortSnake          = "port"
	FieldUASnake            = "ua"
	FieldRefererSnake       = "referer"
	FieldResBodySnake       = "res_body"
	FieldQueryParamsSnake   = "query_params"
	FieldBodySnake          = "body"
	FieldBytesSentSnake     = "bytes_sent"
	FieldBytesReceivedSnake = "bytes_received"
	FieldRequestIDSnake     = "request_id"
	FieldReqHeadersSnake    = "req_headers"
	FieldResHeadersSnake    = "res_headers"
	FieldRouteSnake         = "route"
)

// ConfigDefault is the default config
var ConfigDefault = GinZerologConfig{
	Next:            nil,
	Logger:          nil,
	GetLogger:       nil,
	Fields:          []string{FieldIP, FieldLatency, FieldStatus, FieldMethod, FieldURL, FieldError},
	WrapHeaders:     false,
	FieldsSnakeCase: false,
	Messages:        []string{"Server error", "Client error", "Success"},
	Levels:          []zerolog.Level{zerolog.ErrorLevel, zerolog.WarnLevel, zerolog.InfoLevel},
	SkipURIs:        []string{},
	GetResBody:      nil,
}

// New creates a new middleware handler
func NewGinZerolog(config ...GinZerologConfig) gin.HandlerFunc {
	// Set default config
	cfg := ConfigDefault

	// Override config if provided
	if len(config) > 0 {
		cfg = config[0]
	}

	// Set default logger if not provided
	if cfg.Logger == nil && cfg.GetLogger == nil {
		logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		cfg.Logger = &logger
	}

	// Set default fields if empty
	if len(cfg.Fields) == 0 {
		cfg.Fields = ConfigDefault.Fields
	}

	// Set default messages if empty
	if len(cfg.Messages) == 0 {
		cfg.Messages = ConfigDefault.Messages
	}

	// Set default levels if empty
	if len(cfg.Levels) == 0 {
		cfg.Levels = ConfigDefault.Levels
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// Skip middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			c.Next()
			return
		}

		// Skip URIs if configured
		for _, uri := range cfg.SkipURIs {
			if c.Request.URL.Path == uri {
				c.Next()
				return
			}
		}

		// Start timer
		start := time.Now()

		// Capture request body if needed
		var reqBody []byte
		if contains(cfg.Fields, FieldBody) {
			if c.Request.Body != nil {
				reqBody, _ = io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			}
		}

		// Use response writer wrapper to capture response body
		var resBodyBuffer bytes.Buffer
		var originalWriter gin.ResponseWriter
		if contains(cfg.Fields, FieldResBody) || cfg.GetResBody != nil {
			originalWriter = c.Writer
			c.Writer = &responseBodyWriter{
				ResponseWriter: c.Writer,
				buffer:         &resBodyBuffer,
			}
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get logger
		var logger zerolog.Logger
		if cfg.GetLogger != nil {
			logger = cfg.GetLogger(c)
		} else {
			logger = *cfg.Logger
		}

		// Determine log level and message
		status := c.Writer.Status()
		var level zerolog.Level
		var message string

		if status >= 500 {
			level = cfg.Levels[0] // Server error
			message = cfg.Messages[0]
		} else if status >= 400 {
			level = cfg.Levels[1] // Client error
			message = cfg.Messages[1]
		} else {
			level = cfg.Levels[2] // Success
			message = cfg.Messages[2]
		}

		// Start building log event
		event := logger.WithLevel(level)

		// Add fields based on configuration
		for _, field := range cfg.Fields {
			addField(event, field, c, latency, reqBody, &resBodyBuffer, cfg)
		}

		// Send log
		event.Msg(message)

		// Restore original writer if wrapped
		if originalWriter != nil {
			c.Writer = originalWriter
		}
	})
}

// responseBodyWriter wraps gin.ResponseWriter to capture response body
type responseBodyWriter struct {
	gin.ResponseWriter
	buffer *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.buffer.Write(b)
	return w.ResponseWriter.Write(b)
}

// addField adds a field to the log event based on field name
func addField(event *zerolog.Event, field string, c *gin.Context, latency time.Duration, reqBody []byte, resBody *bytes.Buffer, cfg GinZerologConfig) {
	fieldName := field
	if cfg.FieldsSnakeCase {
		fieldName = toSnakeCase(field)
	}

	switch field {
	case FieldLatency:
		event.Dur(fieldName, latency)
	case FieldStatus:
		event.Int(fieldName, c.Writer.Status())
	case FieldMethod:
		event.Str(fieldName, c.Request.Method)
	case FieldURL:
		event.Str(fieldName, c.Request.URL.String())
	case FieldError:
		if len(c.Errors) > 0 {
			event.Str(fieldName, c.Errors.String())
		}
	case FieldTime:
		event.Time(fieldName, time.Now())
	case FieldPID:
		event.Int(fieldName, os.Getpid())
	case FieldIP:
		event.Str(fieldName, c.ClientIP())
	case FieldIPs:
		event.Str(fieldName, c.GetHeader("X-Forwarded-For"))
	case FieldHost:
		event.Str(fieldName, c.Request.Host)
	case FieldPath:
		event.Str(fieldName, c.Request.URL.Path)
	case FieldProtocol:
		event.Str(fieldName, c.Request.Proto)
	case FieldPort:
		if port := c.Request.URL.Port(); port != "" {
			if p, err := strconv.Atoi(port); err == nil {
				event.Int(fieldName, p)
			}
		}
	case FieldUA:
		event.Str(fieldName, c.Request.UserAgent())
	case FieldReferer:
		event.Str(fieldName, c.Request.Referer())
	case FieldResBody:
		if cfg.GetResBody != nil {
			if body := cfg.GetResBody(c); body != nil {
				event.Bytes(fieldName, body)
			}
		} else if resBody != nil {
			event.Bytes(fieldName, resBody.Bytes())
		}
	case FieldQueryParams:
		if len(c.Request.URL.RawQuery) > 0 {
			if cfg.WrapHeaders {
				event.Interface(fieldName, c.Request.URL.Query())
			} else {
				event.Str(fieldName, c.Request.URL.RawQuery)
			}
		}
	case FieldBody:
		if len(reqBody) > 0 {
			event.Bytes(fieldName, reqBody)
		}
	case FieldBytesSent:
		event.Int(fieldName, c.Writer.Size())
	case FieldBytesReceived:
		event.Int64(fieldName, c.Request.ContentLength)
	case FieldRequestID:
		if reqID := c.GetHeader("X-Request-ID"); reqID != "" {
			event.Str(fieldName, reqID)
		}
	case FieldReqHeaders:
		if cfg.WrapHeaders {
			event.Interface(fieldName, c.Request.Header)
		} else {
			for key, values := range c.Request.Header {
				event.Strs(strings.ToLower(key), values)
			}
		}
	case FieldResHeaders:
		if cfg.WrapHeaders {
			event.Interface(fieldName, c.Writer.Header())
		} else {
			for key, values := range c.Writer.Header() {
				event.Strs(strings.ToLower(key), values)
			}
		}
	case FieldRoute:
		event.Str(fieldName, c.FullPath())
	}
}

// toSnakeCase converts field names to snake_case
func toSnakeCase(field string) string {
	switch field {
	case FieldResBody:
		return FieldResBodySnake
	case FieldQueryParams:
		return FieldQueryParamsSnake
	case FieldBytesSent:
		return FieldBytesSentSnake
	case FieldBytesReceived:
		return FieldBytesReceivedSnake
	case FieldRequestID:
		return FieldRequestIDSnake
	case FieldReqHeaders:
		return FieldReqHeadersSnake
	case FieldResHeaders:
		return FieldResHeadersSnake
	default:
		return field
	}
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

# GinZerolog Middleware

A comprehensive zerolog logging middleware for the Gin web framework, inspired by [fiberzerolog](https://docs.gofiber.io/contrib/fiberzerolog/).

## Features

- **Comprehensive Field Logging**: Log latency, status, method, URL, IP, user agent, headers, request/response bodies, and more
- **Configurable Fields**: Choose which fields to include in your logs
- **Snake Case Support**: Option to use snake_case field names
- **Custom Logger Support**: Use your own zerolog logger instance or get logger per request
- **URI Skipping**: Skip logging for specific URIs (e.g., health checks)
- **Header Wrapping**: Option to wrap headers in dictionaries or log them individually
- **Custom Messages & Levels**: Define custom log messages and levels based on response status
- **Response Body Capture**: Capture response bodies with optional custom handler for compressed responses
- **Performance Optimized**: Only captures request/response bodies when needed

## Installation

This middleware is part of the `golang-api-common` library and uses the Gin framework with zerolog.

```go
import "github.com/Primeskills-Web-Team/golang-api-common/v2/middleware"
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/Primeskills-Web-Team/golang-api-common/v2/middleware"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.New()

    // Use default configuration
    r.Use(middleware.NewGinZerolog())

    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello, World!"})
    })

    r.Run(":8080")
}
```

### Custom Configuration

```go
package main

import (
    "os"
    "time"

    "github.com/Primeskills-Web-Team/golang-api-common/v2/middleware"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"
)

func main() {
    r := gin.New()

    // Create custom logger
    logger := zerolog.New(zerolog.ConsoleWriter{
        Out:        os.Stdout,
        TimeFormat: time.RFC3339,
    }).With().Timestamp().Logger()

    // Use custom configuration
    r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
        Logger: &logger,
        Fields: []string{
            middleware.FieldLatency,
            middleware.FieldStatus,
            middleware.FieldMethod,
            middleware.FieldPath,
            middleware.FieldIP,
            middleware.FieldUA,
        },
        FieldsSnakeCase: true,
        SkipURIs: []string{"/health", "/metrics"},
    }))

    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello, World!"})
    })

    r.Run(":8080")
}
```

## Configuration Options

### GinZerologConfig

| Property          | Type                                | Description                       | Default                                              |
| ----------------- | ----------------------------------- | --------------------------------- | ---------------------------------------------------- |
| `Next`            | `func(*gin.Context) bool`           | Skip middleware when returns true | `nil`                                                |
| `Logger`          | `*zerolog.Logger`                   | Custom zerolog logger             | `zerolog.New(os.Stderr).With().Timestamp().Logger()` |
| `GetLogger`       | `func(*gin.Context) zerolog.Logger` | Get custom logger per request     | `nil`                                                |
| `Fields`          | `[]string`                          | Fields to include in logs         | `["latency", "status", "method", "url", "error"]`    |
| `WrapHeaders`     | `bool`                              | Wrap headers in dictionary        | `false`                                              |
| `FieldsSnakeCase` | `bool`                              | Use snake_case for field names    | `false`                                              |
| `Messages`        | `[]string`                          | Custom response messages          | `["Server error", "Client error", "Success"]`        |
| `Levels`          | `[]zerolog.Level`                   | Custom response levels            | `[ErrorLevel, WarnLevel, InfoLevel]`                 |
| `SkipURIs`        | `[]string`                          | URIs to skip logging              | `[]`                                                 |
| `GetResBody`      | `func(*gin.Context) []byte`         | Custom response body getter       | `nil`                                                |

## Available Fields

### Standard Fields

- `FieldLatency` - Request processing duration
- `FieldStatus` - HTTP response status code
- `FieldMethod` - HTTP method
- `FieldURL` - Full request URL
- `FieldPath` - Request path
- `FieldError` - Error message (if any)
- `FieldTime` - Current timestamp
- `FieldPID` - Process ID
- `FieldIP` - Client IP address
- `FieldIPs` - X-Forwarded-For header
- `FieldHost` - Request host
- `FieldProtocol` - HTTP protocol
- `FieldPort` - Request port
- `FieldUA` - User agent
- `FieldReferer` - Referer header

### Body and Headers

- `FieldBody` - Request body
- `FieldResBody` - Response body
- `FieldReqHeaders` - Request headers
- `FieldResHeaders` - Response headers
- `FieldQueryParams` - Query parameters
- `FieldBytesSent` - Bytes sent
- `FieldBytesReceived` - Bytes received
- `FieldRequestID` - Request ID (from X-Request-ID header)
- `FieldRoute` - Matched route pattern

## Usage Examples

### 1. Custom Fields Selection

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Fields: []string{
        middleware.FieldLatency,
        middleware.FieldStatus,
        middleware.FieldMethod,
        middleware.FieldPath,
        middleware.FieldIP,
        middleware.FieldUA,
        middleware.FieldError,
    },
}))
```

### 2. Snake Case Field Names

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    FieldsSnakeCase: true, // "resBody" becomes "res_body"
    Fields: []string{
        middleware.FieldLatency,
        middleware.FieldResBody,
        middleware.FieldQueryParams,
    },
}))
```

### 3. Skip Specific URIs

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    SkipURIs: []string{"/health", "/metrics", "/ping"},
}))
```

### 4. Request/Response Body Logging

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Fields: []string{
        middleware.FieldLatency,
        middleware.FieldStatus,
        middleware.FieldMethod,
        middleware.FieldPath,
        middleware.FieldBody,      // Log request body
        middleware.FieldResBody,   // Log response body
    },
}))
```

### 5. Header Logging

```go
// Individual header fields
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Fields: []string{
        middleware.FieldReqHeaders,
        middleware.FieldResHeaders,
    },
    WrapHeaders: false, // Headers logged as individual fields
}))

// Wrapped headers
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Fields: []string{
        middleware.FieldReqHeaders,
        middleware.FieldResHeaders,
    },
    WrapHeaders: true, // Headers wrapped in reqHeaders/resHeaders objects
}))
```

### 6. Per-Request Custom Logger

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    GetLogger: func(c *gin.Context) zerolog.Logger {
        // Add trace ID and user ID to each log entry
        return logger.With().
            Str("trace_id", c.GetHeader("X-Trace-ID")).
            Str("user_id", c.GetHeader("X-User-ID")).
            Logger()
    },
}))
```

### 7. Custom Messages and Levels

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Messages: []string{
        "Internal Server Error",    // 5xx status codes
        "Bad Request",             // 4xx status codes
        "Request Successful",      // 2xx-3xx status codes
    },
    Levels: []zerolog.Level{
        zerolog.ErrorLevel,        // 5xx status codes
        zerolog.WarnLevel,         // 4xx status codes
        zerolog.InfoLevel,         // 2xx-3xx status codes
    },
}))
```

### 8. Custom Response Body Handler (for Compression)

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Fields: []string{
        middleware.FieldLatency,
        middleware.FieldStatus,
        middleware.FieldMethod,
        middleware.FieldPath,
        middleware.FieldResBody,
    },
    GetResBody: func(c *gin.Context) []byte {
        // Custom logic to get response body
        // Useful when using compression middleware
        if body, exists := c.Get("uncompressed_response"); exists {
            if b, ok := body.([]byte); ok {
                return b
            }
        }
        return nil
    },
}))
```

### 9. Skip Based on Conditions

```go
r.Use(middleware.NewGinZerolog(middleware.GinZerologConfig{
    Next: func(c *gin.Context) bool {
        // Skip logging for health checks and static files
        path := c.Request.URL.Path
        return path == "/health" ||
               path == "/ping" ||
               strings.HasPrefix(path, "/static/")
    },
}))
```

## Performance Considerations

- **Body Capture**: Request and response body capture only happens when `FieldBody` or `FieldResBody` are included in the Fields configuration
- **Header Logging**: Headers are only processed when `FieldReqHeaders` or `FieldResHeaders` are specified
- **URI Skipping**: Use `SkipURIs` or `Next` function to skip logging for high-frequency endpoints like health checks
- **Field Selection**: Only include fields you actually need to minimize log size and processing overhead

## Output Examples

### Default Configuration Output

```json
{
	"level": "info",
	"latency": 1234567,
	"status": 200,
	"method": "GET",
	"url": "http://localhost:8080/users?page=1",
	"time": "2023-10-01T12:00:00Z",
	"message": "Success"
}
```

### Snake Case Output

```json
{
	"level": "info",
	"latency": 1234567,
	"status": 200,
	"method": "GET",
	"url": "http://localhost:8080/users?page=1",
	"res_body": "eyJ1c2VycyI6W119",
	"query_params": "page=1",
	"request_id": "req-123",
	"time": "2023-10-01T12:00:00Z",
	"message": "Success"
}
```

### Wrapped Headers Output

```json
{
	"level": "info",
	"latency": 1234567,
	"status": 200,
	"method": "GET",
	"url": "http://localhost:8080/users",
	"reqHeaders": {
		"accept": ["application/json"],
		"user-agent": ["MyApp/1.0"]
	},
	"resHeaders": {
		"content-type": ["application/json"],
		"x-request-id": ["req-123"]
	},
	"time": "2023-10-01T12:00:00Z",
	"message": "Success"
}
```

## Comparison with fiberzerolog

This middleware provides similar functionality to [fiberzerolog](https://docs.gofiber.io/contrib/fiberzerolog/) but adapted for the Gin framework:

| Feature                | fiberzerolog | ginzerolog |
| ---------------------- | ------------ | ---------- |
| Custom Logger          | ✅           | ✅         |
| Field Selection        | ✅           | ✅         |
| Snake Case Fields      | ✅           | ✅         |
| Skip URIs              | ✅           | ✅         |
| Header Wrapping        | ✅           | ✅         |
| Custom Messages/Levels | ✅           | ✅         |
| Response Body Capture  | ✅           | ✅         |
| Per-Request Logger     | ✅           | ✅         |
| Framework              | Fiber        | Gin        |

## License

This middleware is part of the golang-api-common library.

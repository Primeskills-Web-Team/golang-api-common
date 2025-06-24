# Logger Package

A structured logging package built on top of [zerolog](https://github.com/rs/zerolog) that provides a simple and configurable logging interface for Go applications.

## Features

- Structured JSON logging
- Configurable log levels
- Development mode with human-readable console output
- Timestamp included in all log entries
- Thread-safe logging

## Installation

```bash
go get github.com/your-repo/logger
```

## Usage

### Basic Usage

```go
package main

import (
    "github.com/your-repo/logger"
    "github.com/rs/zerolog/log"
)

func main() {
    // Create a new logger instance
    logger := logger.New(logger.Config{
        Level:         "info",
        IsDevelopment: true,
    })

    // Setup the logger
    if err := logger.Setup(); err != nil {
        panic(err)
    }

    // Use the logger
    log.Info().Msg("Application started")
    log.Error().Err(err).Msg("Something went wrong")
}
```

### Configuration

The logger can be configured using the `Config` struct:

```go
type Config struct {
    // Level is the logging level. It can be:
    // - "trace"
    // - "debug"
    // - "info"
    // - "warn"
    // - "error"
    // - "fatal"
    // - "panic"
    // Default: "info"
    Level string

    // IsDevelopment indicates whether the application is in development mode.
    // When true, logs are written in a human-readable format to stdout.
    // When false, logs are written in JSON format to stderr.
    // Default: false
    IsDevelopment bool
}
```

### Log Levels

The logger supports the following log levels:

- `trace`: Most verbose level, useful for debugging
- `debug`: Debugging information
- `info`: General operational information
- `warn`: Warning messages
- `error`: Error messages
- `fatal`: Fatal errors that cause the application to exit
- `panic`: Panic messages that cause the application to panic

### Development Mode

When `IsDevelopment` is set to `true`, the logger will output human-readable logs to stdout with timestamps in RFC3339 format. This is useful during development and debugging.

When `IsDevelopment` is set to `false`, the logger will output structured JSON logs to stderr, which is more suitable for production environments.

## Examples

### Info Logging

```go
log.Info().Msg("Application started")
```

### Error Logging

```go
err := someFunction()
if err != nil {
    log.Error().Err(err).Msg("Failed to execute someFunction")
}
```

### Structured Logging

```go
log.Info().
    Str("user_id", "123").
    Int("age", 25).
    Bool("is_active", true).
    Msg("User logged in")
```

## Best Practices

1. Set the appropriate log level for your environment
2. Use structured logging with relevant fields
3. Include error details when logging errors
4. Use development mode during local development
5. Use production mode (JSON output) in production environments

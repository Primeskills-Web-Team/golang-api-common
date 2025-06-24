# HTTP Client Package

A robust and configurable HTTP client package that provides a simple interface for making HTTP requests with built-in retry mechanism, logging, and error handling.

## Features

- Configurable timeout and retry settings
- Automatic retry with exponential backoff
- Request/response logging
- Context support for cancellation
- Default headers management
- Comprehensive error handling
- JSON request/response handling

## Installation

```bash
go get github.com/Primeskills-Web-Team/golang-api-common/v2/httpclient
```

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/Primeskills-Web-Team/golang-api-common/v2/httpclient"
)

func main() {
    // Create client with default configuration
    client := httpclient.NewHttpClient(nil)

    // Create context
    ctx := context.Background()

    // Define request data
    type RequestBody struct {
        Name string `json:"name"`
    }
    type ResponseBody struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }

    // Make request
    reqBody := RequestBody{Name: "John"}
    var respBody ResponseBody

    err := client.Call(
        ctx,
        "POST",
        "https://api.example.com/users",
        map[string]string{
            "Authorization": "Bearer token123",
        },
        reqBody,
        &respBody,
    )
    if err != nil {
        // Handle error
    }
}
```

### Custom Configuration

```go
config := &httpclient.Config{
    Timeout:          10 * time.Second,
    MaxRetries:       5,
    RetryWaitTime:    2 * time.Second,
    MaxRetryWaitTime: 20 * time.Second,
}

client := httpclient.NewHttpClient(config)
```

## Configuration Options

| Option           | Description                       | Default |
| ---------------- | --------------------------------- | ------- |
| Timeout          | Request timeout duration          | 30s     |
| MaxRetries       | Maximum number of retry attempts  | 3       |
| RetryWaitTime    | Initial wait time between retries | 1s      |
| MaxRetryWaitTime | Maximum wait time between retries | 10s     |

## Supported HTTP Methods

- GET
- POST
- PUT
- PATCH
- DELETE

## Error Handling

The client provides detailed error messages for various scenarios:

- Invalid HTTP method
- Request creation failures
- Network errors
- Response parsing errors
- HTTP status code errors (4xx, 5xx)

Example error handling:

```go
err := client.Call(ctx, "GET", url, headers, nil, &result)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "unsupported HTTP method"):
        // Handle invalid method
    case strings.Contains(err.Error(), "failed after"):
        // Handle retry exhaustion
    case strings.Contains(err.Error(), "request failed with status"):
        // Handle HTTP error status
    default:
        // Handle other errors
    }
}
```

## Logging

The client logs the following information:

- Request details (method, URL, headers, body)
- Response details (status code, body)
- Retry attempts with wait times
- Errors with context

## Best Practices

1. Always provide a context for request cancellation
2. Set appropriate timeout values for your use case
3. Configure retry settings based on your requirements
4. Handle errors appropriately
5. Use structured logging for better debugging
6. Set appropriate headers for your API
7. Validate response data

## Example with Full Configuration

```go
func makeAPICall() error {
    // Create custom configuration
    config := &httpclient.Config{
        Timeout:          5 * time.Second,
        MaxRetries:       3,
        RetryWaitTime:    1 * time.Second,
        MaxRetryWaitTime: 10 * time.Second,
    }

    // Create client
    client := httpclient.NewHttpClient(config)

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Define headers
    headers := map[string]string{
        "Content-Type":  "application/json",
        "Authorization": "Bearer token123",
        "X-Request-ID":  "req-123",
    }

    // Make request
    var result interface{}
    err := client.Call(ctx, "GET", "https://api.example.com", headers, nil, &result)
    if err != nil {
        return fmt.Errorf("API call failed: %w", err)
    }

    return nil
}
```

## Dependencies

- Standard library `net/http` package
- `github.com/rs/zerolog` for logging

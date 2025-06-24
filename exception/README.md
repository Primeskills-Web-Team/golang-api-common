# Exception Package

A standardized HTTP exception handling package that provides a consistent way to create and manage HTTP exceptions in Go applications.

## Features

- Standardized HTTP exception handling
- Predefined common HTTP exceptions
- JSON-serializable error responses
- Consistent error message formatting
- HTTP status code validation

## Installation

```bash
go get github.com/Primeskills-Web-Team/golang-api-common/v2/exception
```

## Usage

### Basic Usage

```go
import "github.com/Primeskills-Web-Team/golang-api-common/v2/exception"

func handler(w http.ResponseWriter, r *http.Request) {
    // Return a 400 Bad Request
    err := exception.BadRequest("Invalid input parameters")
    // Handle the error...

    // Return a 404 Not Found
    err = exception.NotFound("Resource not found")
    // Handle the error...
}
```

### Available Exceptions

The package provides the following predefined exceptions:

| Status Code | Function                              | Description             |
| ----------- | ------------------------------------- | ----------------------- |
| 400         | `BadRequest(message string)`          | Bad request error       |
| 401         | `Unauthorized(message string)`        | Unauthorized access     |
| 403         | `Forbidden(message string)`           | Forbidden access        |
| 404         | `NotFound(message string)`            | Resource not found      |
| 409         | `Conflict(message string)`            | Resource conflict       |
| 500         | `InternalServerError(message string)` | Internal server error   |
| 501         | `NotImplemented(message string)`      | Feature not implemented |
| 503         | `ServiceUnavailable(message string)`  | Service unavailable     |
| 504         | `GatewayTimeout(message string)`      | Gateway timeout         |

### Custom Exceptions

You can create custom HTTP exceptions using the `NewHTTPException` function:

```go
// Create a custom 418 I'm a teapot exception
err := exception.NewHTTPException(http.StatusTeapot, "I'm a teapot")
```

### HTTPException Structure

```go
type HTTPException struct {
    StatusCode int    `json:"statusCode"`
    Message    string `json:"message"`
}
```

## Error Handling

### Default Behavior

- If no status code is provided, defaults to 500 (Internal Server Error)
- If no message is provided, uses the standard HTTP status text
- All exceptions implement the `error` interface

### Example Response

```json
{
	"statusCode": 400,
	"message": "Invalid input parameters"
}
```

## Best Practices

1. Use predefined exceptions for common HTTP status codes
2. Provide clear and descriptive error messages
3. Handle exceptions at the appropriate level in your application
4. Log exceptions with appropriate context
5. Return consistent error responses across your API

## Integration with HTTP Handlers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    err := someOperation()
    if err != nil {
        if httpErr, ok := err.(*exception.HTTPException); ok {
            w.WriteHeader(httpErr.StatusCode)
            json.NewEncoder(w).Encode(httpErr)
            return
        }
        // Handle other types of errors...
    }
}
```

## Error Interface Implementation

The `HTTPException` implements the standard Go `error` interface:

```go
func (e *HTTPException) Error() string {
    return e.Message
}
```

## Dependencies

- Standard library `net/http` package

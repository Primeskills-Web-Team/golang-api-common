# Common Package

A collection of common utilities and handlers for Go applications using the Gin web framework. This package provides standardized response structures, health check handlers, and error handling utilities.

## Components

### 1. Response Structure (`response.go`)

Provides standardized response structures for API endpoints.

```go
// ResponseDto : Struct for API response
type ResponseDto struct {
    Success bool                 `json:"success"`
    Message string               `json:"message"`
    Errors  []ErrorValidationDto `json:"errors"`
    Data    interface{}          `json:"data"`
}

// ErrorValidationDto : Struct for error validation response
type ErrorValidationDto struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

**Features:**

- Consistent response format across all endpoints
- Support for success/error status
- Structured error validation messages
- Flexible data payload

**Example Response:**

```json
{
	"success": true,
	"message": "Operation successful",
	"errors": null,
	"data": {
		"id": 1,
		"name": "John Doe"
	}
}
```

### 2. Health Check Handler (`ping_handler.go`)

Provides a simple health check endpoint for service monitoring.

```go
func HandlePing(ctx *gin.Context)
```

**Features:**

- Simple "pong" response to verify service health
- Returns 200 OK status
- Uses standardized response format

**Usage:**

```go
router.GET("/ping", common.HandlePing)
```

**Example Response:**

```json
{
	"success": true,
	"message": "pong",
	"errors": null,
	"data": null
}
```

### 3. Not Found Handler (`notfound_handler.go`)

Handles 404 Not Found errors in a standardized way.

```go
func HandleNotFound(ctx *gin.Context)
```

**Features:**

- Consistent 404 error response
- Uses standardized response format
- Clear error message

**Usage:**

```go
router.NoRoute(common.HandleNotFound)
```

**Example Response:**

```json
{
	"success": false,
	"message": "Route not found",
	"errors": null,
	"data": null
}
```

## Best Practices

1. **Response Structure:**

   - Always use `ResponseDto` for API responses
   - Set appropriate success status
   - Provide clear and meaningful messages
   - Use proper error validation structure when needed

2. **Health Checks:**

   - Implement health check endpoint in all services
   - Use consistent endpoint naming (/ping)
   - Monitor health check responses

3. **Error Handling:**
   - Use standardized error responses
   - Implement proper 404 handling
   - Provide clear error messages

## Example Usage

### Setting up a basic router with common handlers:

```go
func SetupRouter() *gin.Engine {
    router := gin.New()

    // Health check endpoint
    router.GET("/ping", common.HandlePing)

    // API routes
    api := router.Group("/api")
    {
        api.GET("/users", func(ctx *gin.Context) {
            ctx.JSON(http.StatusOK, common.ResponseDto{
                Success: true,
                Message: "Users retrieved successfully",
                Data:    []User{},
            })
        })
    }

    // Handle 404
    router.NoRoute(common.HandleNotFound)

    return router
}
```

### Using ResponseDto with validation errors:

```go
func CreateUser(ctx *gin.Context) {
    // Validation errors
    errors := []common.ErrorValidationDto{
        {
            Field:   "email",
            Message: "Invalid email format",
        },
    }

    ctx.JSON(http.StatusBadRequest, common.ResponseDto{
        Success: false,
        Message: "Validation failed",
        Errors:  errors,
        Data:    nil,
    })
}
```

## Dependencies

- [Gin Web Framework](https://github.com/gin-gonic/gin)

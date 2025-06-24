# Middleware Package

A collection of middleware components for Go applications using the Gin web framework. This package provides essential middleware for authentication, validation, logging, error handling, and more.

## Available Middleware

### 1. Authentication Middleware (`auth.go`)

Handles JWT token validation and user authentication.

```go
type AuthMiddleware interface {
    Authorize(ctx *gin.Context)
}
```

**Features:**

- Validates Bearer token from request header
- Communicates with auth service for token validation
- Sets authenticated user in request context

**Usage:**

```go
authMiddleware := NewAuthMiddleware(httpClient, authServiceUrl)
router.Use(authMiddleware.Authorize)
```

### 2. Validation Middleware (`validation.go`)

Provides request validation for both body and query parameters.

```go
type ValidationMiddleware interface {
    ValidateBody(ctx *gin.Context, v any)
    ValidateQuery(ctx *gin.Context, v any)
}
```

**Features:**

- Validates request body against struct tags
- Validates query parameters
- Returns structured validation errors
- Sets validated data in request context

**Usage:**

```go
validationMiddleware := NewValidationMiddleware(validator)
router.POST("/users", validationMiddleware.ValidateBody(&UserRequest{}))
```

### 3. Logger Middleware (`logger.go`)

Provides comprehensive request logging with configurable options.

```go
type LoggerConfig struct {
    ExcludePaths []string
    AppName      string
    Version      string
}
```

**Features:**

- Request/response logging
- Configurable log levels based on status codes
- Request ID tracking
- Latency measurement
- Client IP and User Agent logging
- Excludable paths
- Service name and version tracking

**Usage:**

```go
config := LoggerConfig{
    AppName: "myapp",
    Version: "1.0.0",
    ExcludePaths: []string{"/health"},
}
router.Use(Logger(config))
```

### 4. Recovery Middleware (`recovery.go`)

Handles panic recovery and provides graceful error responses.

**Features:**

- Recovers from panics
- Returns 500 status code for unhandled panics
- Logs panic information

**Usage:**

```go
router.Use(gin.RecoveryWithWriter(Recover))
```

### 5. Exception Middleware (`exception.go`)

Provides centralized exception handling for the application.

**Features:**

- Handles HTTP exceptions
- Provides consistent error responses
- Logs error information
- Supports custom HTTP exceptions

**Usage:**

```go
router.Use(Exception())
```

### 6. Resource Header Middleware (`resource.go`)

Adds resource identification to response headers.

**Features:**

- Sets X-Resource header in responses
- Helps with resource tracking and monitoring

**Usage:**

```go
router.Use(ResourceHeader("users"))
```

## Best Practices

1. **Authentication:**

   - Always use authentication middleware for protected routes
   - Configure proper auth service URL
   - Handle token expiration gracefully

2. **Validation:**

   - Define clear validation rules using struct tags
   - Use appropriate validation middleware for each endpoint
   - Handle validation errors consistently

3. **Logging:**

   - Configure appropriate log levels
   - Exclude health check endpoints from logging
   - Include relevant context in logs

4. **Error Handling:**

   - Use exception middleware for centralized error handling
   - Implement custom HTTP exceptions when needed
   - Log errors with appropriate context

5. **Resource Headers:**
   - Use resource headers consistently across endpoints
   - Choose meaningful resource names

## Example Setup

```go
func SetupRouter() *gin.Engine {
    router := gin.New()

    // Middleware setup
    router.Use(gin.RecoveryWithWriter(Recover))
    router.Use(Exception())
    router.Use(Logger(LoggerConfig{
        AppName: "myapp",
        Version: "1.0.0",
    }))

    // Auth middleware for protected routes
    authMiddleware := NewAuthMiddleware(httpClient, authServiceUrl)
    protected := router.Group("/api")
    protected.Use(authMiddleware.Authorize)

    // Validation middleware
    validationMiddleware := NewValidationMiddleware(validator)

    // Routes with middleware
    protected.POST("/users", validationMiddleware.ValidateBody(&UserRequest{}))

    return router
}
```

## Dependencies

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Zerolog](https://github.com/rs/zerolog)
- [Gin Request ID](https://github.com/gin-contrib/requestid)

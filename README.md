# Golang API Common

A comprehensive collection of common utilities, middleware, and tools for building robust Go API applications. This package provides essential components for authentication, validation, logging, error handling, and more.

## Features

- 🔐 Authentication middleware with JWT support
- ✅ Request validation with custom rules
- 📝 Structured logging with configurable levels
- 🛡️ Error handling and recovery
- 📊 Pagination utilities
- 🔍 Request tracing
- 🎯 Type-safe utilities
- 🌐 Internationalization support

## Packages

### 1. Common (`/common`)

Standard response structures and common handlers.

```go
// Standard response structure
type ResponseDto struct {
    Success bool                 `json:"success"`
    Message string               `json:"message"`
    Errors  []ErrorValidationDto `json:"errors"`
    Data    interface{}          `json:"data"`
}
```

**Features:**

- Standardized API responses
- Health check handler
- 404 handler
- Error validation structure

[Read more about Common package](common/README.md)

### 2. Middleware (`/middleware`)

Essential middleware components for API applications.

```go
// Example middleware setup
router := gin.New()
router.Use(middleware.Logger(middleware.LoggerConfig{
    AppName: "myapp",
    Version: "1.0.0",
}))
router.Use(middleware.Exception())
router.Use(middleware.Recover())
```

**Features:**

- Authentication middleware
- Request validation
- Logging middleware
- Error recovery
- Exception handling
- Resource headers

[Read more about Middleware package](middleware/README.md)

### 3. Types (`/types`)

Common types and data structures.

```go
// User type example
type User struct {
    Id            int         `json:"id"`
    FullName      string      `json:"full_name"`
    Email         string      `json:"email"`
    // ... more fields
}

// Set implementations
type StringSet struct {
    data map[string]struct{}
}
```

**Features:**

- User-related types
- Set implementations (StringSet, UintSet)
- Type-safe data structures

[Read more about Types package](types/README.md)

### 4. Utils (`/utils`)

Utility functions for common operations.

```go
// Example utility usage
token := utils.GetBearerToken(ctx)
user := utils.ExtractUserFromCtx(ctx)
utils.SetPaginationHeader(ctx, page, limit, totalCount)
```

**Features:**

- Authentication utilities
- Validation helpers
- Pagination utilities
- Signature generation and validation

[Read more about Utils package](utils/README.md)

### 5. Validator (`/validator`)

Powerful validation package with custom rules support.

```go
// Example validation
type User struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}
```

**Features:**

- Struct validation
- Custom validation rules
- Internationalization support
- Error translation

[Read more about Validator package](validator/README.md)

## Installation

```bash
go get github.com/Primeskills-Web-Team/golang-api-common/v2
```

## Quick Start

```go
package main

import (
    "github.com/Primeskills-Web-Team/golang-api-common/v2/common"
    "github.com/Primeskills-Web-Team/golang-api-common/v2/middleware"
    "github.com/Primeskills-Web-Team/golang-api-common/v2/utils"
    "github.com/gin-gonic/gin"
)

func main() {
    router := gin.New()

    // Setup middleware
    router.Use(middleware.Logger(middleware.LoggerConfig{
        AppName: "myapp",
        Version: "1.0.0",
    }))
    router.Use(middleware.Exception())
    router.Use(middleware.Recover())

    // Health check
    router.GET("/ping", common.HandlePing)

    // Protected routes
    protected := router.Group("/api")
    protected.Use(middleware.Authorize())

    // Example route with validation
    protected.POST("/users", func(c *gin.Context) {
        // Your handler logic here
        c.JSON(200, common.ResponseDto{
            Success: true,
            Message: "User created successfully",
            Data:    nil,
        })
    })

    router.Run(":8080")
}
```

## Best Practices

1. **Error Handling:**

   - Use the exception middleware for centralized error handling
   - Return structured error responses
   - Log errors appropriately

2. **Authentication:**

   - Always use authentication middleware for protected routes
   - Validate tokens properly
   - Handle token expiration

3. **Validation:**

   - Validate all incoming requests
   - Use appropriate validation rules
   - Return clear validation errors

4. **Logging:**

   - Configure appropriate log levels
   - Include relevant context in logs
   - Use structured logging

5. **Performance:**
   - Reuse validator instances
   - Use appropriate middleware
   - Handle pagination properly

## Dependencies

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [go-playground/validator](https://github.com/go-playground/validator)
- [Zerolog](https://github.com/rs/zerolog)
- [go-playground/universal-translator](https://github.com/go-playground/universal-translator)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

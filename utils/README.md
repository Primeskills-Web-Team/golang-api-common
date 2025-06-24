# Utils Package

A collection of utility functions for Go applications using the Gin web framework. This package provides common utilities for validation, authentication, pagination, and signature generation.

## Components

### 1. Validation Utilities (`validation.go`)

Provides utilities for handling validated data and path parameters.

```go
// Extract validated data from context
func ExtractValidatedData[V any](c *gin.Context) *V

// Get path parameter with validation
func GetPathParam(c *gin.Context, paramName string) (string, error)
```

**Features:**

- Generic type support for validated data extraction
- Path parameter validation
- Error handling for missing parameters

**Example Usage:**

```go
// Extract validated request body
type CreateUserRequest struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

func CreateUser(c *gin.Context) {
    // After validation middleware
    data := utils.ExtractValidatedData[CreateUserRequest](c)
    if data == nil {
        // Handle error
        return
    }

    // Use validated data
    fmt.Println(data.Name, data.Email)
}

// Get path parameter
func GetUser(c *gin.Context) {
    userId, err := utils.GetPathParam(c, "id")
    if err != nil {
        // Handle error
        return
    }

    // Use userId
    fmt.Println("User ID:", userId)
}
```

### 2. Authentication Utilities (`auth.go`)

Provides utilities for handling authentication tokens and user context.

```go
// Extract Bearer token
func GetBearerToken(ctx *gin.Context) string

// Extract user from context
func ExtractUserFromCtx(ctx *gin.Context) *types.User
```

**Features:**

- Bearer token extraction
- User context extraction
- Type-safe user object handling

**Example Usage:**

```go
func ProtectedRoute(c *gin.Context) {
    // Get Bearer token
    token := utils.GetBearerToken(c)
    if token == "" {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // Get user from context
    user := utils.ExtractUserFromCtx(c)
    if user == nil {
        c.JSON(401, gin.H{"error": "User not found"})
        return
    }

    // Use user data
    fmt.Println("User:", user.FullName)
}
```

### 3. Signature Utilities (`signature.go`)

Provides utilities for generating and validating HMAC signatures.

```go
// Generate signature
func GenerateSignature(body interface{}, secret string) (string, error)

// Validate signature
func ValidateSignature(body interface{}, secret, signature string) (bool, error)
```

**Features:**

- HMAC SHA256 signature generation
- Signature validation
- Support for any JSON-serializable body

**Example Usage:**

```go
type WebhookPayload struct {
    Event string `json:"event"`
    Data  any    `json:"data"`
}

func HandleWebhook(c *gin.Context) {
    var payload WebhookPayload
    if err := c.BindJSON(&payload); err != nil {
        c.JSON(400, gin.H{"error": "Invalid payload"})
        return
    }

    // Get signature from header
    signature := c.GetHeader("X-Signature")

    // Validate signature
    isValid, err := utils.ValidateSignature(payload, "your-secret-key", signature)
    if err != nil || !isValid {
        c.JSON(401, gin.H{"error": "Invalid signature"})
        return
    }

    // Process webhook
    fmt.Println("Processing webhook:", payload.Event)
}
```

### 4. Pagination Utilities (`pagination.go`)

Provides utilities for handling pagination in API responses.

```go
func SetPaginationHeader(ctx *gin.Context, page, limit, totalCount int)
```

**Features:**

- Pagination header generation
- Next/previous page calculation
- Total pages calculation
- Standard pagination headers

**Example Usage:**

```go
func ListUsers(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

    // Get users from database
    users, totalCount := database.GetUsers(page, limit)

    // Set pagination headers
    utils.SetPaginationHeader(c, page, limit, totalCount)

    // Return response
    c.JSON(200, gin.H{
        "data": users,
    })
}
```

## Best Practices

1. **Validation:**

   - Always validate request data before processing
   - Use type-safe validation utilities
   - Handle validation errors appropriately

2. **Authentication:**

   - Always check for token presence
   - Validate user context before accessing protected resources
   - Handle authentication errors gracefully

3. **Signatures:**

   - Use strong secret keys
   - Validate signatures for all webhook requests
   - Handle signature validation errors

4. **Pagination:**
   - Set appropriate page sizes
   - Include pagination headers in list responses
   - Handle edge cases (first/last page)

## Response Headers

The pagination utility sets the following headers:

- `X-Total-Count`: Total number of items
- `X-Total-Pages`: Total number of pages
- `X-Page`: Current page number
- `X-Limit`: Number of items per page
- `X-Next-Page`: Next page number (if available)
- `X-Prev-Page`: Previous page number (if available)

## Dependencies

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Go Standard Library](https://golang.org/pkg/)

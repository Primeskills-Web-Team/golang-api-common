# Validator Package

A powerful validation package for Go applications that provides structured validation with custom validators and internationalization support. Built on top of [go-playground/validator](https://github.com/go-playground/validator).

## Features

- Struct validation with custom rules
- JSON and form field name support
- Custom validator registration
- Internationalization support
- Structured error messages
- Type-safe validation

## Components

### 1. Validator (`validator.go`)

The main validator component that handles struct validation and error translation.

```go
type Validator struct {
    validate   *validator.Validate
    translator ut.Translator
}
```

**Features:**

- Built-in validation rules
- Custom validator support
- Error translation
- JSON and form field name support
- Structured error responses

### 2. Custom Validator Interface (`custom_validator.go`)

Interface for implementing custom validation rules.

```go
type CustomValidator interface {
    Tag() string
    Func() validator.Func
    Translation() (translation string, customFunc validator.TranslationFunc)
}
```

## Usage

### Basic Usage

```go
// Create a new validator
v, err := validator.NewValidator()
if err != nil {
    log.Fatal(err)
}

// Define a struct with validation rules
type User struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,gte=0,lte=130"`
    Password string `json:"password" validate:"required,min=8"`
}

// Validate the struct
user := User{
    Name:     "John Doe",
    Email:    "invalid-email",
    Age:      150,
    Password: "123",
}

errors := v.ValidateStruct(user)
if errors != nil {
    // Handle validation errors
    for _, err := range errors {
        fmt.Printf("Field: %s, Error: %s\n", err.Field, err.Message)
    }
}
```

### Custom Validator Example

```go
// Define a custom validator for password strength
type PasswordStrengthValidator struct{}

func (v PasswordStrengthValidator) Tag() string {
    return "password_strength"
}

func (v PasswordStrengthValidator) Func() validator.Func {
    return func(fl validator.FieldLevel) bool {
        password := fl.Field().String()
        // Check for at least one uppercase, one lowercase, one number, and one special character
        hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
        hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
        hasNumber := strings.ContainsAny(password, "0123456789")
        hasSpecial := strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?")

        return hasUpper && hasLower && hasNumber && hasSpecial
    }
}

func (v PasswordStrengthValidator) Translation() (string, validator.TranslationFunc) {
    return "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
        func(ut ut.Translator, fe validator.FieldError) string {
            return fe.Translate(ut)
        }
}

// Register and use the custom validator
v, err := validator.NewValidator(
    validator.WithCustomValidator(PasswordStrengthValidator{}),
)
if err != nil {
    log.Fatal(err)
}

type User struct {
    Password string `json:"password" validate:"required,password_strength"`
}

user := User{Password: "weak"}
errors := v.ValidateStruct(user)
```

### Validation Tags

The validator supports all standard validation tags from go-playground/validator, including:

- `required`: Field must be set
- `email`: Must be a valid email address
- `min`: Minimum length for strings/slices
- `max`: Maximum length for strings/slices
- `gte`: Greater than or equal to
- `lte`: Less than or equal to
- `oneof`: Value must be one of the specified values
- And many more...

## Best Practices

1. **Validation Rules:**

   - Use appropriate validation tags for each field
   - Combine multiple validation rules when needed
   - Create custom validators for complex validation logic

2. **Error Handling:**

   - Always check for validation errors
   - Provide clear error messages
   - Use structured error responses

3. **Custom Validators:**

   - Keep validation logic simple and focused
   - Provide clear translation messages
   - Test custom validators thoroughly

4. **Performance:**
   - Reuse validator instances
   - Register custom validators at startup
   - Use appropriate validation tags

## Dependencies

- [go-playground/validator](https://github.com/go-playground/validator)
- [go-playground/universal-translator](https://github.com/go-playground/universal-translator)
- [go-playground/locales](https://github.com/go-playground/locales)

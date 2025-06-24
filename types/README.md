# Types Package

A collection of common types and data structures for Go applications. This package provides user-related types and Set implementations inspired by TypeScript's Set type.

## Components

### 1. User Types (`user.go`)

Provides user-related data structures for user management.

```go
type User struct {
    Id            int         `json:"id"`
    FullName      string      `json:"full_name"`
    Dob           string      `json:"dob"`
    Email         string      `json:"email"`
    Provider      string      `json:"provider"`
    RoleUser      []RoleUser  `json:"role"`
    UserCreator   UserCreator `json:"user_creator"`
    AccountActive string      `json:"account_active"`
    Password      string      `json:"-"`
    RoleId        int         `json:"role_id"`
    CPIsDone      int         `json:"cp_is_done"`
    Picture       string      `json:"picture"`
    CreatedAt     time.Time   `json:"created_at"`
    UpdatedAt     time.Time   `json:"updated_at"`
}
```

**Features:**

- Complete user entity structure
- Role management
- User creator tracking
- Timestamp tracking
- JSON serialization support
- Password field excluded from JSON

### 2. String Set (`string_set.go`)

A Set implementation for strings, inspired by TypeScript's Set type.

```go
type StringSet struct {
    data map[string]struct{}
}
```

**Features:**

- Unique string values
- O(1) lookup time
- Thread-safe operations
- Memory efficient using empty struct

**Methods:**

```go
// Create a new set
set := NewStringSet()

// Add elements
set.Add("value1")
set.Add("value2")

// Check existence
exists := set.Has("value1")

// Remove elements
set.Remove("value1")

// Get all values
values := set.Keys()

// Get size
size := set.Size()

// Clear set
set.Clear()
```

### 3. Uint Set (`uint_set.go`)

A Set implementation for unsigned integers, inspired by TypeScript's Set type.

```go
type UintSet struct {
    data map[uint]struct{}
}
```

**Features:**

- Unique uint values
- O(1) lookup time
- Thread-safe operations
- Memory efficient using empty struct

**Methods:**

```go
// Create a new set
set := NewUintSet()

// Add elements
set.Add(1)
set.Add(2)

// Check existence
exists := set.Has(1)

// Remove elements
set.Remove(1)

// Get all values
values := set.Values()

// Get size
size := set.Size()

// Clear set
set.Clear()
```

## Usage Examples

### User Type Example

```go
user := types.User{
    Id:       1,
    FullName: "John Doe",
    Email:    "john@example.com",
    RoleUser: []types.RoleUser{
        {
            Id:     1,
            RoleId: 1,
        },
    },
}
```

### String Set Example

```go
// Create a new string set
set := types.NewStringSet()

// Add some values
set.Add("apple")
set.Add("banana")
set.Add("orange")

// Check if a value exists
if set.Has("apple") {
    fmt.Println("Apple is in the set")
}

// Get all values
fruits := set.Keys()
fmt.Println(fruits) // ["apple", "banana", "orange"]

// Remove a value
set.Remove("banana")

// Get size
fmt.Println(set.Size()) // 2
```

### Uint Set Example

```go
// Create a new uint set
set := types.NewUintSet()

// Add some values
set.Add(1)
set.Add(2)
set.Add(3)

// Check if a value exists
if set.Has(1) {
    fmt.Println("1 is in the set")
}

// Get all values
numbers := set.Values()
fmt.Println(numbers) // [1, 2, 3]

// Remove a value
set.Remove(2)

// Get size
fmt.Println(set.Size()) // 2
```

## Best Practices

1. **User Types:**

   - Always validate user data before creating User instances
   - Use proper role management
   - Handle timestamps appropriately
   - Keep password field secure

2. **Set Types:**
   - Use Set types when you need unique values
   - Prefer Set over slice for membership testing
   - Clear sets when they're no longer needed
   - Use appropriate Set type (StringSet or UintSet) based on your needs

## Performance Considerations

- Both Set implementations use map with empty struct for optimal memory usage
- All operations (Add, Remove, Has) are O(1)
- Keys() and Values() operations are O(n) as they need to create a new slice
- Clear() operation is O(1) as it creates a new map

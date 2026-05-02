# error

First-class error values.

## Overview

The `error` type represents an error or exceptional condition. Errors are first-class values in Kavun, meaning they can
be stored, passed around, and operated on like any other value. This allows for elegant error handling and propagation.

## Declaration and Creation

### Construction

```go
e = error("something went wrong")
e2 = error("Database connection failed")
e3 = error("Invalid input: expected integer")
```

### From Values

```go
message = "Network timeout"
err = error(message)
```

## Member Functions

### General Functions

#### `copy()`

Returns a deep copy of the error.

**Arguments:** None

**Returns:** `error`

**Description:** Equivalent to the builtin `copy(x)`. Produces a fresh error value with a deep copy of the payload.

```go
e = error("boom")
c = e.copy()
```

### Accessor Functions

#### `value()`

Gets the error message.

**Arguments:** None

**Returns:** `any`

**Description:** Returns the payload/message that was wrapped in the error.

```go
e = error("something went wrong")
e.value()    // "something went wrong"

// Error with complex payload
details = {code: 404, message: "Not found"}
e = error(details)
e.value()    // {code: 404, message: "Not found"}
```

### Conversion Functions

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the error message as a string. If the error payload is not a string, it attempts to convert it to string format.

```go
e = error("something went wrong")
e.string()   // "something went wrong"

// Error with non-string payload
e2 = error(404)
e2.string()  // "404"
```

## Built-in Error Functions

### Error Detection

#### `is_error(x)`

Checks if a value is an error.

**Arguments:**

- `x` (any): Value to check

**Returns:** `bool`

**Description:** Returns `true` if the value is an error, `false` otherwise.

```go
e = error("failed")
is_error(e)           // true

value = 42
is_error(value)       // false

undefined_val = undefined
is_error(undefined_val)  // false
```

## Examples

### Basic Error Handling

```go
fmt = import("fmt")

// Create and check errors
result = error("operation failed")

if is_error(result) {
    fmt.println("Error occurred: " + result.string())
}
```

### Error Propagation

```go
fmt = import("fmt")

// Function that returns error on failure
divide = func(a, b) {
    if b == 0 {
        return error("division by zero")
    }
    return a / b
}

result = divide(10, 0)
if is_error(result) {
    fmt.println("Calculation failed: " + result.value().string())
}
```

### Error with Structured Data

```go
fmt = import("fmt")

// Error with detailed information
validate_user = func(data) {
    if data.name == undefined || data.name == "" {
        return error({
            code: "INVALID_NAME",
            message: "Name is required",
            field: "name"
        })
    }

    if data.age == undefined || data.age < 0 {
        return error({
            code: "INVALID_AGE",
            message: "Age must be non-negative",
            field: "age"
        })
    }

    return data
}

user = {name: "", age: 25}
result = validate_user(user)

if is_error(result) {
    details = result.value()
    fmt.println("Validation failed")
    fmt.println("Code: " + details.code)
    fmt.println("Field: " + details.field)
}
```

## Design Notes

- Errors are values, not exceptions - they don't interrupt execution
- Use conditional checks with `is_error()` to handle errors
- Errors can be returned from functions or stored in data structures
- The payload of an error can be any type, allowing flexible error representation

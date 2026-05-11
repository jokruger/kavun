# error

First-class error values.

## Overview

The `error` type represents an error or exceptional condition. Errors are first-class values in Kavun, meaning they can
be stored, passed around, and operated on like any other value. This allows for elegant error handling and propagation.

## Declaration and Creation

### Construction

```go
e1 = error("something went wrong")           // string payload (a message)
e2 = error({field: "name", code: 42})        // structured payload
```

`error(payload)` takes exactly one argument — any value that should be attached to the error. The payload can be
read back by `value()`. Calling `error()` with no arguments is rejected: an empty error carries no information
and is almost always a bug.

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

#### `format([spec])`

Renders the value as a string using the [Format Mini-Language](../format-mini-language.md).

**Arguments:**

- `spec` (optional, `string`) - format mini-language spec. Defaults to `""`.

**Returns:** `string`

**Description:** Equivalent to using the value as the operand of an f-string interpolation, e.g.
`f"{x:<spec>}"` - except the spec is parsed on each call rather than at compile time. With no argument or with an empty
string the type's default rendering is returned. The set of accepted verbs and modifiers is type-specific;
see [Format Mini-Language](../format-mini-language.md) for the full grammar.

```go
error("boom").format()       // "boom"
error("boom").format("v")    // 'error("boom")'
```

### Accessor Functions

#### `value()`

Returns the payload attached to the error.

**Arguments:** None

**Returns:** `any`

**Description:** Returns the value that was passed to `error(...)`. For runtime errors (caught via `recover()`)
this is the message body as a string. For user errors it is whatever the script passed.

```go
e = error("something went wrong")
e.value()    // "something went wrong"

// Error with structured payload
details = {code: 404, message: "Not found"}
e = error(details)
e.value()    // {code: 404, message: "Not found"}
```

#### `kind()`

Returns a stable string tag identifying the kind of error. For runtime errors this is the failure category
(e.g. `"division_by_zero"`, `"index_out_of_bounds"`, `"invalid_argument_type"`). For errors created by user
code via `error(...)`, `kind()` returns `"user"`.

Use `kind()` to branch on the type of failure inside a deferred `recover()`:

```go
defer func() {
    e := recover()
    if e != undefined {
        if e.kind() == "division_by_zero" { /* ... */ }
    }
}()
```

#### `is_runtime()`

Convenience predicate. Returns `true` if the error was raised by the runtime
(i.e. `kind() != "user"`), `false` if the error was created by user code via
`error(...)`.

```go
error("oops").is_runtime()    // false
// inside a deferred recover():
//   recover().is_runtime()   // true when caught a runtime error
```

### Conversion Functions

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the error message as a string. If the error payload is not a string, it attempts to convert it
to string format.

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

### Raising

#### `raise(err)`

Raises a Kavun error so it propagates up the call stack until caught by a `recover()` inside a deferred function. If
`err` is not already an error value, it is wrapped automatically.

**Arguments:**

- `err` (any): error value to raise (or any value to wrap as an error)

**Returns:** does not return — the surrounding instruction unwinds.

```go
divide := func(a, b) {
    if b == 0 { raise(error("division by zero")) }
    return a / b
}

safe := func() result {
    defer func() {
        e := recover()
        if e != undefined { result = e }
    }()
    divide(10, 0)
}
```

See `docs/language.md` for the full `defer` / `recover` semantics, including recoverable vs fatal error severity.

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

// Function that returns error on failure with structured details
validate_user = func(data) {
    if data.name == undefined || data.name == "" {
        return error({code: "INVALID_NAME", message: "Name is required", field: "name"})
    }

    if data.age == undefined || data.age < 0 {
        return error({code: "INVALID_AGE", message: "Age must be non-negative", field: "age"})
    }

    return data
}

user = {name: "", age: 25}
result = validate_user(user)

if is_error(result) {
    details = result.value()
    fmt.println("Validation failed: " + details.message)
    fmt.println("Code: " + details.code)
    fmt.println("Field: " + details.field)
}
```

## Design Notes

- Errors are values, not exceptions - they don't interrupt execution
- Use conditional checks with `is_error()` to handle errors
- Errors can be returned from functions or stored in data structures
- The payload of an error can be any type, allowing flexible error representation

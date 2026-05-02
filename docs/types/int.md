# int

Signed integer type for whole numbers.

## Overview

The `int` type represents 64-bit signed integers (-9,223,372,036,854,775,808 to 9,223,372,036,854,775,807).

## Declaration and Usage

### Decimal Literals

```go
i = 42
j = -100
k = 0
```

### Hexadecimal Literals

```go
hex = 0x2a        // 42
color = 0xFF00FF  // 16711935
```

### Octal Literals

```go
perms = 0o755     // 493
```

### Binary Literals

```go
bits = 0b1010     // 10
mask = 0b11111111 // 255
```

## Arithmetic Operations

```go
a = 10
b = 3

a + b      // 13
a - b      // 7
a * b      // 30
a / b      // 3 (integer division)
a % b      // 1 (modulo)
a ** 2     // 100 (exponentiation)
```

## Comparison and Logical Operations

```go
5 > 3       // true
5 < 3       // false
5 == 5      // true
5 != 3      // true
5 >= 5      // true
```

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `int`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `int` is immutable, this method
returns the receiver unchanged.

```go
(42).copy()    // 42
```

### Conversion Functions

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the same integer value.

```go
(42).int()   // 42
```

#### `float()`

Converts to floating-point.

**Arguments:** None

**Returns:** `float`

**Description:** Converts the integer to a float with no precision loss for smaller values.

```go
(42).float()       // 42.0
(1000000).float()  // 1000000.0
```

#### `decimal()`

Converts to decimal (exact decimal type).

**Arguments:** None

**Returns:** `decimal`

**Description:** Converts the integer to a decimal for exact arithmetic.

```go
(42).decimal()    // decimal(42)
(999).decimal()   // decimal(999)
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `false` for `0`, `true` for all other values.

```go
(0).bool()     // false
(42).bool()    // true
(-1).bool()    // true
```

#### `rune()`

Converts to rune (Unicode code point).

**Arguments:** None

**Returns:** `rune`

**Description:** Converts the integer to a Unicode code point. The value must be a valid Unicode code point
(0 to 0x10FFFF).

```go
(65).rune()           // 'A'
(0x1F600).rune()      // '😀'
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts the integer to its string representation in base 10.

```go
(42).string()      // "42"
(-100).string()    // "-100"
```

#### `time()`

Converts to time (Unix timestamp).

**Arguments:** None

**Returns:** `time`

**Description:** Interprets the integer as Unix time (seconds since epoch).

```go
(0).time()                  // 1970-01-01T00:00:00Z
(1704067200).time()         // 2024-01-01T00:00:00Z
```

### Numeric Utility Functions

#### `sign()`

Determines the sign of the integer.

**Arguments:** None

**Returns:** `int`

**Description:** Returns `-1` for negative, `0` for zero, `1` for positive.

```go
(42).sign()      // 1
(-42).sign()     // -1
(0).sign()       // 0
```

#### `abs()`

Returns the absolute value.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the absolute (non-negative) value.

```go
(42).abs()       // 42
(-42).abs()      // 42
(0).abs()        // 0
```

## Examples

### Working with Ranges

```go
fmt = import("fmt")

// Generate sequence of integers
numbers = range(1, 11).array()    // [1, 2, 3, ..., 10]

// Iterate and process
for i in range(0, 5) {
    fmt.println(i)
}
```

### Numeric Operations

```go
// Calculate statistics
values = [10, 20, 30, 40, 50]
total = values.sum()              // 150
average = total / values.len()    // 30

// Process with transformations
doubled = values.map(x => x * 2)  // [20, 40, 60, 80, 100]
evens = values.filter(x => (x % 2) == 0)
```

### Sign and Absolute Value

```go
// Determine direction
velocity = -15
direction = velocity.sign()       // -1 (moving backwards)
speed = velocity.abs()            // 15

// Normalize values
values = [-5, 3, -8, 2]
absolute_values = values.map(v => v.abs())  // [5, 3, 8, 2]
```

### Type Conversions

```go
// Mixed type arithmetic
count = 5
total = (count).decimal() + decimal("10.5")  // decimal(15.5)

// String formatting with int
id = 12345
message = "User ID: " + id.string()    // "User ID: 12345"

// Time operations
timestamp = 1704067200
event_time = timestamp.time()  // Parse as Unix timestamp
```

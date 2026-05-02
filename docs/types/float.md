# float

Floating-point type for decimal numbers with limited precision.

## Overview

The `float` type represents IEEE 754 double-precision floating-point numbers. Use `float` when you need fast arithmetic
with acceptable precision loss, or use `decimal` when you need exact decimal arithmetic.

## Declaration and Usage

### Decimal Literals

```go
f = 3.14
pi = 3.14159265
g = -42.5
```

### Scientific Notation

```go
large = 1e3        // 1000.0
small = 1e-3       // 0.001
scientific = 2.5e2 // 250.0
```

### Float Suffix

```go
h = 1f             // 1.0 (explicit float literal)
inf = 1e308f * 2   // Infinity
```

### Special Values

```go
inf = 1.0 / 0.0    // Infinity
nan = 0.0 / 0.0    // NaN (Not a Number)
```

## Arithmetic Operations

```go
a = 10.5
b = 2.5

a + b      // 13.0
a - b      // 8.0
a * b      // 26.25
a / b      // 4.2
a % b      // 0.5
a ** 2     // 110.25
```

## Comparison and Special Cases

```go
3.14 > 3.0         // true
3.14 == 3.14       // true (may have precision issues)

inf = 1.0 / 0.0
inf > 999999       // true
inf == inf         // true

nan = 0.0 / 0.0
nan == nan         // false (NaN never equals anything, including itself)
```

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `float`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `float` is immutable, this method
returns the receiver unchanged.

```go
(3.14).copy()    // 3.14
```

### Conversion Functions

#### `float()`

Converts to float.

**Arguments:** None

**Returns:** `float`

**Description:** Returns the same float value.

```go
(3.14).float()    // 3.14
```

#### `decimal()`

Converts to decimal (exact decimal type).

**Arguments:** None

**Returns:** `decimal`

**Description:** Converts the float to a decimal. Note that `NaN` and infinities convert to decimal `NaN`.

```go
(3.14).decimal()      // decimal(3.14)
(1e308 * 2).decimal() // decimal(NaN)
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Truncates toward zero. Special values like `NaN` and infinities return implementation-defined values.

```go
(3.14).int()      // 3
(3.99).int()      // 3
(-3.14).int()     // -3
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts the float to its string representation. Special values are represented as `"Inf"`, `"-Inf"`,
and `"NaN"`.

```go
(3.14).string()        // "3.14"
(1e3).string()         // "1000"
(1.0 / 0.0).string()   // "Inf"
(0.0 / 0.0).string()   // "NaN"
```

### Numeric Utility Functions

#### `sign()`

Determines the sign of the float.

**Arguments:** None

**Returns:** `int`

**Description:** Returns `-1` for negative, `0` for zero, `1` for positive. Special handling for special values.

```go
(3.14).sign()      // 1
(-3.14).sign()     // -1
(0.0).sign()       // 0
```

## Examples

### Basic Calculations

```go
fmt = import("fmt")

// Calculate area and perimeter
radius = 5.0
area = 3.14159 * radius * radius      // 78.53975
circumference = 2.0 * 3.14159 * radius  // 31.4159

fmt.println("Area:", area)
```

### Working with Collections

```go
// Average calculation
scores = [85.5, 92.0, 78.5, 95.5]
average = scores.sum() / scores.len().float()  // 87.875

// Temperature conversion
celsius = [0.0, 10.0, 20.0, 30.0]
fahrenheit = celsius.map(c => (c * 9.0 / 5.0) + 32.0)
// [32.0, 50.0, 68.0, 86.0]
```

### Precision Considerations

```go
fmt = import("fmt")

// Float precision limitations
a = 0.1 + 0.2
b = 0.3
fmt.println(a == b)   // false (due to floating-point rounding)

// For exact decimal arithmetic, use decimal type
exact_a = decimal("0.1") + decimal("0.2")
exact_b = decimal("0.3")
fmt.println(exact_a == exact_b)    // true
```

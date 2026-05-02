# decimal

Exact decimal type for precise arithmetic.

## Overview

The `decimal` type provides exact decimal arithmetic for cases where precision is critical, such as financial
calculations. Unlike `float`, decimals maintain exact values without rounding errors inherent to binary
floating-point representation.

## Declaration and Construction

### Decimal Literals

```go
a = 1.23d
b = 123d
zero = 0d
```

### Construction via Function

```go
// From integer
a2 = decimal(123)           // decimal(123)

// From float
b2 = decimal(1.23)          // decimal(1.23)

// From string
c2 = decimal("1.23")        // decimal(1.23)
d2 = decimal("1.23e2")      // decimal(123)

// From existing decimal
e2 = decimal(a)             // decimal(1.23)
```

### Member-based Construction

```go
d = (123).decimal()
e = (1.23).decimal()
f = "1.23".decimal()
```

## Conversion Rules

The `decimal(x)` function follows these rules:

- `decimal()` with no arguments returns `decimal(0)`
- `decimal(decimalValue)` returns the same decimal value
- `decimal(x)` attempts runtime conversion via the type system's `AsDecimal` handler
- `decimal(x, fallback)` returns `fallback` when conversion fails

**Fallback Behavior:**

```go
decimal("invalid")              // undefined
decimal("invalid", 0d)          // decimal(0)
decimal(undefined, 1.5d)        // decimal(1.5)
```

**Member Methods:**

```go
// All of these create decimal values
i = 42
f = 3.14
s = "2.71"

i_decimal = i.decimal()        // decimal(42)
f_decimal = f.decimal()        // decimal(3.14)
s_decimal = s.decimal()        // decimal(2.71)
```

## Mixed Arithmetic Operations

When decimals participate in operations with other numeric types:

- `decimal op x` converts `x` to decimal (if possible); result is `decimal`
- `int op decimal` promotes `int` to decimal; result is `decimal`
- `float op decimal` uses float semantics (decimal is converted to float); result is `float`
- `string + decimal` is valid only with string on left; decimal is converted to string

**Examples:**

```go
decimal(1) + 2         // decimal(3)
1 + decimal(2)         // decimal(3)
decimal(1) + 2.0       // float 3.0
1.0 + decimal(2)       // float 3.0 (float semantics)
"value=" + decimal(2)  // "value=2"
```

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `decimal`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `decimal` is immutable, this method
returns the receiver unchanged.

```go
decimal(2).copy()    // decimal(2)
```

### Conversion Functions

#### `decimal()`

Converts to decimal.

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the same decimal value.

```go
decimal(1.23).decimal()    // decimal(1.23)
```

#### `float()`

Converts to floating-point.

**Arguments:** None

**Returns:** `float`

**Description:** Converts the decimal to a float. May lose precision for very large or very precise decimals.

```go
decimal(1.23).float()      // 1.23
decimal("0.1").float()     // 0.1
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Truncates toward zero, discarding the fractional part.

```go
decimal(3.99).int()       // 3
decimal(3.14).int()       // 3
decimal(-3.99).int()      // -3
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts the decimal to its string representation. Includes all significant digits.

```go
decimal(1.23).string()     // "1.23"
decimal("0.1").string()    // "0.1"
decimal(100).string()      // "100"
```

### Classification Functions

#### `is_zero()`

Checks if decimal is zero.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the decimal value is exactly zero.

```go
decimal(0).is_zero()       // true
decimal("0.00").is_zero()  // true
decimal(1).is_zero()       // false
```

#### `is_negative()`

Checks if decimal is negative.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the decimal value is less than zero.

```go
decimal(-5).is_negative()     // true
decimal(5).is_negative()      // false
decimal(0).is_negative()      // false
```

#### `is_positive()`

Checks if decimal is positive.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the decimal value is greater than zero.

```go
decimal(5).is_positive()      // true
decimal(-5).is_positive()     // false
decimal(0).is_positive()      // false
```

#### `is_nan()`

Checks if decimal is NaN.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the decimal is NaN (Not a Number), which occurs in certain conversion scenarios.

```go
decimal("invalid").is_nan()   // true (conversion failure)
float("nan").decimal().is_nan()  // true
decimal(1.23).is_nan()        // false
```

### Metadata Functions

#### `sign()`

Determines the sign of the decimal.

**Arguments:** None

**Returns:** `int`

**Description:** Returns `-1` for negative, `0` for zero, `1` for positive.

```go
decimal(42).sign()         // 1
decimal(-42).sign()        // -1
decimal(0).sign()          // 0
```

#### `scale()`

Gets the scale (number of decimal places).

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of significant digits after the decimal point.

```go
decimal(1.23).scale()      // 2
decimal("0.1").scale()     // 1
decimal(100).scale()       // 0
decimal("1.230").scale()   // 3
```

#### `error_details()`

Gets error information for failed conversions.

**Arguments:** None

**Returns:** `record`

**Description:** Returns details about conversion errors for NaN decimals. Returns an error record if the decimal represents a conversion failure.

```go
fmt = import("fmt")
details = ""
result = decimal("invalid")
if result.is_nan() {
    details = result.error_details()
}
fmt.println(details)
```

### Scale and Normalization Functions

#### `rescale(scale)`

Changes the scale (number of decimal places).

**Arguments:**

- `scale` (int): Target number of decimal places

**Returns:** `decimal`

**Description:** Rescales to the specified number of decimal places. The scale argument must be within the
implementation-defined range; otherwise raises a runtime error.

```go
decimal("1.234").rescale(2)   // decimal(1.23) or decimal(1.24) depending on rounding
decimal(100).rescale(2)       // decimal(100.00)
decimal("1.2").rescale(3)     // decimal(1.200)
```

#### `canonical()`

Returns the canonical form.

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the decimal in canonical form with trailing zeros removed.

```go
decimal("1.230").canonical()  // decimal(1.23)
decimal("100.00").canonical() // decimal(100)
decimal("0.0").canonical()    // decimal(0)
```

#### `trunc(scale)`

Truncates to specified scale.

**Arguments:**

- `scale` (int): Number of decimal places to keep

**Returns:** `decimal`

**Description:** Truncates toward zero to the specified number of decimal places, without rounding.

```go
decimal("1.987").trunc(2)     // decimal(1.98)
decimal("1.234").trunc(1)     // decimal(1.2)
decimal("1.999").trunc(0)     // decimal(1)
```

### Neighbor and Transform Functions

#### `next_up()`

Gets the next representable decimal (toward positive infinity).

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the next decimal value in the direction of positive infinity.

```go
decimal(1).next_up()          // decimal(1.000...001)
decimal(0).next_up()          // smallest positive decimal
```

#### `next_down()`

Gets the previous representable decimal (toward negative infinity).

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the next decimal value in the direction of negative infinity.

```go
decimal(1).next_down()        // decimal(0.999...999)
decimal(0).next_down()        // smallest negative decimal
```

#### `abs()`

Returns absolute value.

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the non-negative value.

```go
decimal(-5.23).abs()          // decimal(5.23)
decimal(5.23).abs()           // decimal(5.23)
```

#### `negate()`

Returns negated value.

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the value with sign reversed.

```go
decimal(5.23).negate()        // decimal(-5.23)
decimal(-5.23).negate()       // decimal(5.23)
```

#### `sqrt()`

Returns the square root.

**Arguments:** None

**Returns:** `decimal`

**Description:** Returns the non-negative square root. Returns NaN for negative decimals.

```go
decimal(4).sqrt()             // decimal(2)
decimal("2.25").sqrt()        // decimal(1.5)
decimal(-1).sqrt()            // NaN
```

### Rounding Functions

The following rounding functions accept a `scale` argument (number of decimal places to round to). The scale must be
within the implementation-defined range; otherwise raises a runtime error.

#### `round_down(scale)`

Rounds toward negative infinity.

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.987").round_down(2)   // decimal(1.98)
decimal("1.234").round_down(0)   // decimal(1)
```

#### `round_up(scale)`

Rounds toward positive infinity.

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.234").round_up(2)     // decimal(1.24)
decimal("1.201").round_up(1)     // decimal(1.3)
```

#### `round_toward_zero(scale)`

Rounds toward zero (truncation).

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.987").round_toward_zero(2)   // decimal(1.98)
decimal("-1.987").round_toward_zero(2)  // decimal(-1.98)
```

#### `round_away_from_zero(scale)`

Rounds away from zero.

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.234").round_away_from_zero(2)   // decimal(1.24)
decimal("-1.234").round_away_from_zero(2)  // decimal(-1.24)
```

#### `round_half_toward_zero(scale)`

Rounds half values toward zero (banker's rounding variant).

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.235").round_half_toward_zero(2)  // decimal(1.23)
decimal("1.245").round_half_toward_zero(2)  // decimal(1.24)
```

#### `round_half_away_from_zero(scale)`

Rounds half values away from zero (standard rounding).

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.235").round_half_away_from_zero(2)  // decimal(1.24)
decimal("1.245").round_half_away_from_zero(2)  // decimal(1.25)
```

#### `round_bank(scale)`

Rounds half values to nearest even (banker's rounding).

**Arguments:**

- `scale` (int): Number of decimal places

**Returns:** `decimal`

```go
decimal("1.235").round_bank(2)   // decimal(1.24) (4 is even)
decimal("1.225").round_bank(2)   // decimal(1.22) (2 is even)
```

## Examples

### Financial Calculations

```go
fmt = import("fmt")

// Price calculation with tax
price = decimal("100.00")
tax_rate = decimal("0.0825")      // 8.25%
tax = (price * tax_rate).round_half_away_from_zero(2)
total = price + tax

fmt.println("Price:", price)
fmt.println("Tax:", tax)
fmt.println("Total:", total)
```

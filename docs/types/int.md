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

## Mixed-Type Arithmetic and Comparison

`int` widens losslessly into `float` and `decimal` — on either operand order, the result is
`float`/`decimal`, not `int`:

```go
1 + 2.5           // 3.5, a float
2.5 + 1           // 3.5, a float
1 + decimal(2)    // decimal(3)
decimal(2) + 1    // decimal(3)

1 < 2.5           // true
2.5 > 1           // true
1 < decimal(2)    // true
decimal(2) > 1    // true
```

`int` has no relationship at all with `byte`/`rune` arithmetic beyond what those types themselves define — see
[byte](byte.md) and [rune](rune.md) for their own `± int` pairings (`byte` wraps mod 256, `rune` offsets and stays
`rune`; neither widens into a plain `int` result the way `float`/`decimal` do). `int` also has no arithmetic
relationship with `bool` (`1 + true` is a runtime error — `bool` arithmetic is out of scope, not just undecided) or
with `string`/`bytes`/`runes`/`time`/`array`/`dict`/`record` (no implicit conversion in either direction).

### Equality and ordering — a wider set than arithmetic

Unlike arithmetic, equality and ordering extend to `bool`/`byte`/`rune` too (all three widen to `int` exactly —
`bool` via `0`/`1`, `byte`/`rune` via their existing value, flattened, not chained):

```go
1 == true          // true -- widens to 1
true < 5           // true
1 == byte(1)       // true
'A' == 65          // true
```

Ordering against `float` is exact, not the lossy `float64(int)` conversion arithmetic uses — this matters once
`int` values get large enough that adjacent integers stop being distinguishable as `float64`:

```go
9007199254740993 == float(9007199254740993)    // false -- would silently collapse with a lossy conversion
9007199254740992 == float(9007199254740992)    // true -- this one IS exactly representable
9007199254740993 > float(9007199254740992)     // true -- still correctly ordered even though not equal
```

Ordering against a `NaN` `float` is always `false`, in both directions, for every one of `< > <= >=` — `int` is
never itself `NaN`, so there's nothing to compare:

```go
nan = 0.0 / 0.0
5 < nan            // false
5 > nan            // false
```

`int` also joins the text tier for equality, comparing against its own canonical decimal-digit text form:

```go
5 == "5"           // true
5 != "6"           // true
```

Numeric-vs-text **ordering** stays undefined — `5 < "5"` is a runtime error, not a lexicographic-vs-numeric
guess.

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
(42).format()                // "42"
(42).format("5d")            // "   42"
(42).format("05d")           // "00042"
(42).format("b")             // "0b101010"
(42).format("06x")           // "0x002a"
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

**Description:** Interprets the integer as a Unix timestamp in **seconds** and returns the instant in UTC.

In a conversion an `int` is always a timestamp, never a duration — that reading belongs to operator
position, where `t + n` adds `n` *nanoseconds*. See
[time: what an `int` means next to a `time`](time.md#what-an-int-means-next-to-a-time) for the full rule.

```go
(0).time()                  // 1970-01-01T00:00:00Z
(1704067200).time()         // 2024-01-01T00:00:00Z
```

#### `time_ms()`

Converts to time from a millisecond Unix timestamp.

**Arguments:** None

**Returns:** `time`

**Description:** Interprets the integer as a Unix timestamp in **milliseconds** and returns the instant in
UTC. This is the encoding JavaScript's `Date.now()` and Java's `System.currentTimeMillis()` produce; passing
such a value to `time()`/`time_*()` with the wrong suffix yields a silently wrong instant, so name the
encoding you actually have. The inverse of `time.unix_ms()`.

```go
(1704067200123).time_ms()      // 2024-01-01T00:00:00.123Z
(1704067200123).time()         // 55969-09-28T00:00:00Z -- seconds reading of a millisecond timestamp
```

#### `time_micro()`

Converts to time from a microsecond Unix timestamp.

**Arguments:** None

**Returns:** `time`

**Description:** Interprets the integer as a Unix timestamp in **microseconds** and returns the instant in
UTC. The inverse of `time.unix_micro()`.

```go
(1704067200123456).time_micro()    // 2024-01-01T00:00:00.123456Z
```

#### `time_nano()`

Converts to time from a nanosecond Unix timestamp.

**Arguments:** None

**Returns:** `time`

**Description:** Interprets the integer as a Unix timestamp in **nanoseconds** and returns the instant in
UTC. The inverse of `time.unix_nano()`, and the only pair that round-trips a sub-second instant exactly.

```go
(1704067200123456789).time_nano()                    // 2024-01-01T00:00:00.123456789Z
t"2024-01-01T00:00:00.123456789Z".unix_nano().time_nano()   // the same instant, unchanged
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

### Sequence Functions

#### `repeat(n)`

Repeats the integer `n` times into an array.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `array`

**Description:** Returns a new array of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
x := 7
x.repeat(3)      // [7, 7, 7]
x.repeat(0)      // []
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

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
nan == nan         // true (deliberately reflexive -- see "NaN and Inf ordering" below)
```

### `NaN` and `Inf` ordering

`float`'s `NaN` is a **total order's unique minimum**, not IEEE-754's "incomparable with everything, including
itself" — `NaN == NaN` is `true`, and `NaN` sorts below every other value, including `-Inf`:

```go
nan = 0.0 / 0.0
nan == nan          // true -- reflexive, unlike raw IEEE-754
nan <= nan           // true
nan < nan            // false -- equal, not "less than," when both are NaN
5.0 > nan            // true -- NaN is always the smallest
nan < -1.0e300       // true -- sorts below even large negative numbers
```

This is a deliberate departure from `float64`'s native comparison operators, and matches `decimal`'s own `NaN`
convention exactly (`decimal("NaN")` and any `NaN` `float` are considered the same value — see
[decimal](decimal.md)). The reason: `array.sort()` depends on `<`/`==` forming a valid order to behave
deterministically, and raw IEEE-754 semantics — where `NaN` compares `false` in both directions against
everything, even itself — made sorting a `float` array containing `NaN` silently non-deterministic. `Inf` needed
no change: it was already a well-ordered, definite value under IEEE-754 (`Inf == Inf`, `5.0 < Inf`, etc. already
worked correctly).

## Mixed-Type Arithmetic and Comparison

`float` accepts `int` on either side, widening it to `float` — result is always `float`:

```go
1 + 2.5           // 3.5
2.5 + 1           // 3.5
1 < 2.5           // true
2.5 > 1           // true
```

`float` and `decimal` deliberately **do not mix for arithmetic**, in either direction — a runtime error, not a
silently-computed answer, since neither representation is an automatic winner over the other:

```go
0.1 + 2.5d        // runtime error: float + decimal
2.5d + 0.1        // runtime error: decimal + float
```

Convert explicitly first if you need to mix them for arithmetic (`(0.1).decimal() + 2.5d`, or
`2.5d.float() + 0.1`).

### Equality and ordering — a wider set than arithmetic

Unlike arithmetic, equality and ordering extend to `bool`/`byte`/`rune` (always exact — their entire ranges sit
inside `float64`'s 53-bit exact-integer mantissa) and, critically, to `decimal` too — the one pairing arithmetic
flatly refuses:

```go
true < 2.5           // true -- widens to 1
byte(5) == 5.0       // true
'A' == 65.0          // true

1.0 < decimal(2)         // true -- ordering works, even though arithmetic doesn't
decimal(2) < 1.0         // false
decimal("0.5") == 0.5    // true -- 0.5 has an exact binary form
decimal("0.1") == 0.1    // false -- float 0.1 is actually
                          //         ~0.1000000000000000055511151231257827021181583404541015625, not exactly 1/10
```

`decimal`/`float` comparisons work by comparing the two operands' true exact mathematical values via
arbitrary-precision rational arithmetic, never by rounding one side into the other's representation first —
rounding either direction would produce exactly the kind of false positive `decimal("0.1") == 0.1` would otherwise
be. `int` gets the same exact treatment against `float` — see [int](int.md) for the large-integer case this
protects against (`9007199254740993 == float(9007199254740993)` stays `false`, never silently collapsing to the
adjacent representable value). `float` also joins the text tier for equality (`3.14 == "3.14"` is `true`) — see
[decimal](decimal.md) for the parallel `NaN`-as-total-order behavior this section's ordering examples build on.
`float` still has no relationship with `string`/`bytes`/`runes`/`time`/`array`/`dict`/`record` beyond that text-tier
equality, and no ordering against them at all (numeric-vs-text ordering stays undefined everywhere).

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
(3.14159).format()           // "3.14159"
(3.14159).format(".2f")      // "3.14"
(0.5).format(".0%")          // "50%"
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

#### `time()`

Converts to time.

**Arguments:** None

**Returns:** `time`

**Description:** Interprets the float as a Unix timestamp read as **sec.frac** — the integer part is seconds
since epoch, the fraction is the sub-second part (the encoding Python's `time.time()` produces). The result is
UTC.

**Lossy by nature:** `float64` carries ~15–16 significant digits and a present-day timestamp already spends
10 of them on the seconds, so the fraction survives to roughly microsecond precision and not exactly. Use
[`decimal`](decimal.md#time) — exact to nanoseconds — or [`int.time_nano()`](int.md#time_nano) when the
sub-second part has to be right. `NaN`, infinities, and values beyond `int64` seconds return `undefined`
(or the `time(x, fallback)` default).

```go
(1704067200.5).time()        // 2024-01-01T00:00:00.5Z
(1704067200.123).time()      // 2024-01-01T00:00:00.122999907Z -- float64 cannot spell .123 here
(-0.5).time()                // 1969-12-31T23:59:59.5Z
(0.0/0.0).time()             // undefined
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

### Sequence Functions

#### `repeat(n)`

Repeats the float `n` times into an array.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `array`

**Description:** Returns a new array of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
f := 1.5
f.repeat(3)      // [1.5, 1.5, 1.5]
f.repeat(0)      // []
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

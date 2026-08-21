# byte

Unsigned 8-bit integer type. Represents values from 0 to 255.

## Declaration and Usage

```go
b = b'A'
b0 = b'\x00'
b = byte(42)
```

`b'...'` literals must resolve to exactly one byte. They accept byte-sized escapes such as `b'\xFF'` and `b'\n'`.
Empty contents, multi-character contents, and Unicode code points above 255 are invalid.

## Arithmetic Operations

`byte` is a genuine ring — `Z/256`, matching Go/Rust's `uint8` bit-for-bit. Every `+`/`-`, same-type or mixed with
a plain `int`, wraps modulo 256 for *any* magnitude of the `int` operand, not just small offsets — and `int` on
either side of the operator gives the ring-correct result, since `byte` is what integers reduce into:

```go
byte(10) + byte(3)    // 13
byte(10) - byte(3)    // 7
byte(255) + byte(1)   // 0 (wraps around)
byte(0) - byte(1)     // 255 (wraps around)

byte(255) + 1         // 0 -- mixes with plain int, same wraparound
1 + byte(255)         // 0 -- same result either order
byte(0) - 300         // 212 -- wraps for any magnitude, not just +/-255
300 - byte(0)         // 44
```

Unary `-` is the ring's additive inverse (`256 - x` mod 256, with `-0 = 0`):

```go
-byte(1)      // 255
-byte(0)      // 0
```

`byte` does **not** mix with `float`/`decimal` — that combination is a runtime error (no automatic
integer-like-type widening is defined for `byte`).

## Bitwise Operations

`& | ^ &^` are same-type only — mixing widths with `int` has no clean "offset" meaning the way arithmetic does, so
`byte(1) & 5` is a runtime error, even though `byte(1) & byte(5)` works. Unary `^` is the same-type bitwise
complement. The one exception is the shift count: `<<`/`>>` accept a plain `int` count in addition to a same-type
`byte` count, matching the universal shift-count convention in other languages.

```go
byte(0b1010) & byte(0b0110)    // byte(0b0010)
byte(0b1010) | byte(0b0110)    // byte(0b1110)
byte(0b1010) ^ byte(0b0110)    // byte(0b1100)
^byte(0)                       // 255 (bitwise complement)
byte(1) << 4                   // 16 -- shifted by a plain int count
byte(1) & 5                    // runtime error -- bitwise stays same-type only
```

## Comparison and Logical Operations

```go
byte(5) > byte(3)     // true
byte(5) < byte(3)     // false
byte(5) == byte(5)    // true
byte(5) != byte(3)    // true
byte(5) >= byte(5)    // true
```

`byte` also orders directly against `int` (converts itself to `int`, compares — so a `byte` is never made to look
"less than" an `int` outside its own 0-255 range just because it wraps for arithmetic):

```go
byte(200) < 300       // true
300 > byte(200)       // true
```

### Equality and ordering against `bool`, `decimal`, and `float`

`byte` widens to `0`/`1` and compares exactly against `bool`, both for equality and ordering — either operand
order:

```go
byte(1) == true       // true
byte(0) == false      // true
true < byte(5)        // true

byte(1) == byte(0)    // false
```

Against `decimal` and `float`, `byte`'s entire range (0-255) sits well inside both types' range of exact
representation, so equality and ordering are always exact, never approximate:

```go
byte(5) == decimal("5")   // true
byte(5) < decimal("6")    // true
byte(5) == 5.0            // true
byte(5) < 5.5              // true
```

`byte` also joins the text tier for equality — comparing against its own canonical decimal-digit text form, not
against the ASCII character it might also represent:

```go
byte(53) == "53"      // true -- decimal digit text, canonical
byte(53) == "5"       // false -- '5' is code point 53, but that's not byte's canonical text form
```

Numeric-vs-text **ordering** stays undefined regardless — `byte(1) < "1"` is a runtime error, the same as every
other numeric type against text.

### `byte` vs. `rune`

Every `byte` value (0-255) widens losslessly into its equivalent Unicode code point (the Latin-1 block), so `byte`
combines directly with `rune` for `-`, ordering, and equality — using whichever of `rune`'s own behaviors applies
once widened (see [rune](rune.md) for the full reasoning). `byte` does **not** combine with `rune` via `+` — that
inherits `rune + rune`'s own rejection (see below), so it's a runtime error, not a new exception to remember.

```go
byte(65) - 'A'    // 0 -- widens to rune, then rune - rune -> the code-point distance
byte(65) < 'B'    // true -- widens, then rune's own ordering
byte(65) + 'B'    // runtime error -- widens to rune + rune, which is itself undefined
```

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `byte`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `byte` is immutable, this method
returns the receiver unchanged.

```go
byte(5).copy()    // byte(5)
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
(65).byte().format()         // "65"
(65).byte().format("x")      // "0x41"
```

### Conversion Functions

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Converts the byte to an integer. The value is preserved as it fits within the range of an integer.

```go
byte(42).int()   // 42
```

#### `float()`

Converts to floating-point.

**Arguments:** None

**Returns:** `float`

**Description:** Converts the byte to a float.

```go
byte(42).float()       // 42.0
```

#### `decimal()`

Converts to decimal (exact decimal type).

**Arguments:** None

**Returns:** `decimal`

**Description:** Converts the byte to a decimal for exact arithmetic.

```go
byte(42).decimal()    // decimal(42)
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `false` for `0`, `true` for all other values.

```go
byte(0).bool()     // false
byte(42).bool()    // true
```

#### `rune()`

Converts to rune (Unicode code point).

**Arguments:** None

**Returns:** `rune`

**Description:** Converts the byte to a Unicode code point.

```go
byte(65).rune()    // 'A'
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts the byte to its string representation in base 10.

```go
byte(42).string()  // "42"
```

### Sequence Functions

#### `repeat(n)`

Repeats the byte `n` times into a `bytes` value.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `bytes`

**Description:** Returns new `bytes` of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
byte(65).repeat(3)   // bytes([65, 65, 65])
byte(0).repeat(0)    // empty bytes
```

#### `join(seq)`

Stringifies each element of `seq` and joins them using the receiver byte as separator.

**Arguments:**

- `seq` (array | range): Sequence of values to join.

**Returns:** `bytes`

**Description:** Each element is stringified, the final result is encoded as bytes. Empty `seq` yields empty bytes.
Single-element `seq` yields just the stringified element. A `range` is treated as if `.array()` were called first.

```go
byte(0x2C).join([1, 2, 3])     // bytes for "1,2,3"
byte(0x2C).join(range(1, 4))   // bytes for "1,2,3"
```

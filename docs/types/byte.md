# byte

Unsigned 8-bit integer type. Represents values from 0 to 255.

## Declaration and Usage

```go
b = byte(42)
```

## Arithmetic Operations

```go
byte(10) + byte(3)    // 13
byte(10) - byte(3)    // 7
byte(255) + byte(1)   // 0 (wraps around)
byte(0) - byte(1)     // 255 (wraps around)
```

## Comparison and Logical Operations

```go
byte(5) > byte(3)     // true
byte(5) < byte(3)     // false
byte(5) == byte(5)    // true
byte(5) != byte(3)    // true
byte(5) >= byte(5)    // true
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

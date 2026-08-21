# rune

Single Unicode code point.

## Overview

The `rune` type represents a single Unicode character. Runes are useful for character-level operations and Unicode
manipulation. Each rune holds a code point from 0 to 0x10FFFF.

## Declaration and Usage

### Rune Literals

```go
c = 'A'
quote = '"'
emoji = '😀'
unicode_char = '\u03B1'  // Greek alpha (α)
```

### Escape Sequences

```go
newline = '\n'
tab = '\t'
backslash = '\\'
quote = '\''
unicode = '\u0041'  // 'A'
```

## Arithmetic Operations

`rune` is a position/symbol type, not a ring like `byte` — it doesn't wrap, and not every combination below is
symmetric. `rune + int` and `int + rune` both offset forward and stay `rune`, either order; `rune - int` offsets
backward and also stays `rune`, but **`int - rune` has no defined meaning** ("a plain number minus a specific
character") and is a runtime error — unlike `byte`, which is a true ring and accepts subtraction from either side.

```go
'A' + 1       // 'B' (next code point, stays a rune)
1 + 'A'       // 'B' (same result either order)
'A' - 1       // '@' (offset backward, stays a rune)
1 - 'A'       // runtime error -- int - rune is not defined
```

`rune - rune` is the one case that escapes to `int` — a genuine distance between two code points, not the same
kind of result as the offsets above:

```go
'9' - '0'     // 9 (digit to value)
'Z' - 'A'     // 25 (alphabet position)
```

`rune + rune` is deliberately undefined — "the sum of two code points" has no meaning the way a distance does:

```go
'A' + 'B'     // runtime error
```

`rune` has no ring structure at all: unary `-` and every bitwise operator (`& | ^ &^ << >>`, including unary `^`)
are runtime errors for `rune`, even same-type — unlike `byte`, which supports all of these.

```go
-'A'          // runtime error
'A' & 'B'     // runtime error
```

### `rune` vs. `byte`

A `byte` widens losslessly into its equivalent code point (the Latin-1 block, U+0000-U+00FF), so combining a
`rune` with a `byte` uses whichever of the behaviors above applies once widened — see [byte](byte.md) for the
full reasoning from the other side:

```go
'A' - byte(65)    // 0 -- widens byte to rune, then the rune - rune distance
byte(65) < 'B'    // true -- widens, then ordering
```

## Comparison Operations

```go
'A' < 'B'     // true (by code point)
'a' > 'A'     // true (lowercase comes after uppercase)
'0' == '0'    // true
```

`rune` also orders directly against plain `int` (either order, by code-point value):

```go
'A' < 66      // true
65 < 'B'      // true
```

### Equality and ordering against `bool`, `decimal`, and `float`

`rune` widens to `0`/`1` and compares exactly against `bool`, both for equality and ordering — either operand
order:

```go
rune(1) == true       // true
rune(0) == false      // true
true < 'z'            // true (1 < 122)
```

Against `decimal` and `float`, `rune`'s entire code-point range sits well inside both types' range of exact
representation, so equality and ordering are always exact:

```go
'A' == decimal("65")  // true
'A' < decimal("66")   // true
'A' == 65.0            // true
'A' < 65.5             // true
```

`rune` also joins the text tier for equality — comparing against its own canonical single-character text form,
**not** against the decimal text of its code point:

```go
'A' == "A"        // true -- rune's canonical text form is the character itself
'A' == "65"       // false -- not the code point's decimal text
65 == 'A'         // true -- but int's canonical text form IS decimal digits, so int == rune still
                  //         resolves through the exact chain above, not through text at all
```

Numeric-vs-text **ordering** stays undefined regardless — `'A' < "A"` is a runtime error, the same as every
other numeric type against text.

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `rune`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `rune` is immutable, this method
returns the receiver unchanged.

```go
'a'.copy()    // 'a'
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
'A'.format()                 // "A"
'A'.format("d")              // "65"
'A'.format("U")              // "U+0041"
```

### Conversion Functions

#### `rune()`

Converts to rune.

**Arguments:** None

**Returns:** `rune`

**Description:** Returns the same rune value.

```go
'A'.rune()    // 'A'
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` for any non-zero code point, `false` for the null character.

```go
'A'.bool()    // true
'\x00'.bool() // false
'0'.bool()    // true (character '0', not the zero code point)
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unicode code point as an integer.

```go
'A'.int()           // 65
'0'.int()           // 48
'😀'.int()          // 128512 (0x1F600)
'\n'.int()          // 10
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts the rune to a single-character string.

```go
'A'.string()        // "A"
'😀'.string()       // "😀"
'\n'.string()       // "\n" (newline as string)
```

### Sequence Functions

#### `repeat(n)`

Repeats the rune `n` times into a `runes` value.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `runes`

**Description:** Returns new `runes` of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
'a'.repeat(3)        // u"aaa"
'こ'.repeat(2)       // u"ここ"
```

#### `join(seq)`

Stringifies each element of `seq` and joins them using the receiver rune as separator.

**Arguments:**

- `seq` (array | range): Sequence of values to join.

**Returns:** `runes`

**Description:** Each element is stringified. The final result is encoded as runes. Empty `seq` yields empty runes.
Single-element `seq` yields just the stringified element. A `range` is treated as if `.array()` were called first.

```go
','.join([1, 2, 3])            // u"1,2,3"
','.join(range(1, 4))          // u"1,2,3"
```

## Examples

### Character Classification

```go
fmt = import("fmt")

// Check character types
is_digit = func(r) {
    return r >= '0' && r <= '9'
}

is_letter = func(r) {
    return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

fmt.println(is_digit('5'))       // true
fmt.println(is_digit('A'))       // false
fmt.println(is_letter('A'))      // true
```

### Character Conversion

```go
// Convert character to digit
digit_char = '7'
digit_value = digit_char.int() - '0'.int()  // 7

// Convert digit to character
value = 3
digit_char = (value + '0'.int()).rune()     // '3'
```

### Working with Unicode

```go
// Unicode string processing
greek_alpha = 'α'  // Unicode character
code_point = greek_alpha.int()  // 945

// Build strings from runes
greeting = 'H'.string() + 'i'.string()  // "Hi"
```

## Unicode Considerations

- Runes represent single Unicode code points, not grapheme clusters
- Some characters appear as multiple code points when combined (combining marks)
- Use `runes` type for string-level operations when Unicode awareness is required
- The `rune` type properly handles all valid Unicode code points (U+0000 to U+10FFFF)
- Invalid code points will raise errors when converting to `rune` from integers

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

Runes participate in arithmetic operations by their numeric code point value:

```go
'A' + 1       // 66 (next code point)
'9' - '0'     // 9 (digit to value)
'Z' - 'A'     // 25 (alphabet position)
```

## Comparison Operations

```go
'A' < 'B'     // true (by code point)
'a' > 'A'     // true (lowercase comes after uppercase)
'0' == '0'    // true
```

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

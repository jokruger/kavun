# string

Immutable UTF-8 encoded text values.

## Overview

Strings are immutable sequences of UTF-8 encoded bytes optimized for compact text data storage. Use `string` for string
keys, printing and formatting, protocol fields, identity-like values, and basic text manipulation. Strings are
reference-typed in terms of performance optimizations but remain logically immutable.

Strings deliberately expose fewer sequence/container helpers than collection-oriented types. They do not provide
accessors such as `first()` or `last()`, or aggregations such as `min()` and `max()`. Their indexing and slicing support
is limited and operates on bytes, not Unicode code points. For correct Unicode indexing and slicing, prefer the
[`runes`](runes.md) type.

**Important:** This type splits operations between byte-level and rune-level:

- `len()`, indexing (`s[i]`), and slicing (`s[a:b]`) operate on **bytes**
- `lower()`, `upper()`, `filter(fn)`, `for_each(fn)`, `all(fn)`, `any(fn)`, and `count(fn)` operate on **runes** (Unicode characters)

This design keeps common string storage and byte-level access compact while providing a small set of Unicode-aware text
operations.

## Declaration and Usage

### String Literals

```go
s = "hello"
message = "Hello, World!"
empty = ""
```

### Escape Sequences

```go
newline = "line1\nline2"
tab = "col1\tcol2"
quote = "He said \"hi\""
backslash = "C:\\Users\\Bob"
```

### Unicode in Strings

```go
greeting = "Bonjour"           // French
wave = "👋"                    // Emoji
japanese = "こんにちは"         // Japanese
```

### Raw Strings

Raw strings preserve escape sequences literally, which is useful for regular expressions and file paths:

```go
pattern = r"\d+\w*"           // raw: literally \d+\w*
path = r"C:\Users\Bob"        // raw: backslashes are literal, no escape processing
regex = r"[a-zA-Z0-9]+"       // raw: patterns stay readable
```

## String Operations

### Concatenation

```go
greeting = "Hello, " + "World"   // "Hello, World"
```

### Comparison

```go
"abc" < "abd"      // true (lexicographic)
"abc" == "abc"     // true
"ABC" != "abc"     // true (case-sensitive)
```

### Indexing and Slicing (Byte-level)

All indexing and slicing operates on **bytes**:

```go
s = "héllo"        // é is 2 bytes in UTF-8
s[0]               // byte(104)
s[-1]              // byte(111) - last byte
s[0:2]             // "hé" (first 2 bytes)
s[:-1]             // "héll"
s[1:5:2]           // "él"
s[5:1:-1]          // "ollé"
s[::-1]            // reversed bytes
len(s)             // 6 (byte length, not character count)
```

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count
from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty
result. Strings also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot
be zero. Out-of-bounds index access raises `index out of bounds`.

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `string`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `string` is immutable, this method
returns the receiver unchanged.

```go
"hello".copy()    // "hello"
```

### Conversion Functions

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the same string.

```go
"hello".string()   // "hello"
```

#### `array()`

Converts to array of code points.

**Arguments:** None

**Returns:** `array`

**Description:** Returns an array where each element is a rune (Unicode character) converted to int.

```go
"ABC".array()      // [65, 66, 67]
"hi".array()       // [104, 105]
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `false` for empty string, `true` otherwise.

```go
"".bool()          // false
"hello".bool()     // true
" ".bool()         // true (space is not empty)
```

#### `bytes()`

Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns a bytes value containing the UTF-8 encoding.

```go
"ABC".bytes()      // bytes with [65, 66, 67]
"".bytes()         // empty bytes
```

#### `float()`

Converts to float.

**Arguments:** None

**Returns:** `float`

**Description:** Parses the string as a floating-point number. Returns `0` when parsing fails.

```go
"3.14".float()     // 3.14
"1e3".float()      // 1000.0
"invalid".float()  // 0
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Parses the string as an integer. Returns `0` when parsing fails.

```go
"42".int()         // 42
"-100".int()       // -100
"invalid".int()    // 0
```

#### `decimal()`

Converts to decimal.

**Arguments:** None

**Returns:** `decimal`

**Description:** Parses the string as a decimal number. Invalid input results in `decimal(NaN)`.

```go
"1.23".decimal()       // decimal(1.23)
"1e2".decimal()        // decimal(100)
"invalid".decimal()    // decimal(NaN)
```

#### `time()`

Converts to time.

**Arguments:** None

**Returns:** `time`

**Description:** Parses the string as a date/time value. Invalid input results in the zero time value.

```go
"2024-01-01".time()             // time at midnight Jan 1, 2024
"2024-01-01T12:30:00Z".time()   // specific time in UTC
"invalid".time()                // zero time value
```

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts the string into a record where keys are string indices (`"0"`, `"1"`, ...), and values are
runes.

```go
"abc".record()    // {"0": 'a', "1": 'b', "2": 'c'}
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts the string into a dict where keys are string indices (`"0"`, `"1"`, ...), and values are
runes.

```go
"abc".dict()       // dict({"0": 'a', "1": 'b', "2": 'c'})
```

### Transformation and Filtering Functions

#### `lower()`

Converts to lowercase.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the string with all letters converted to lowercase. Operates on **runes** (Unicode-aware).

```go
"HELLO".lower()        // "hello"
"HeLLo WoRLd".lower()  // "hello world"
"Café".lower()         // "café"
```

#### `upper()`

Converts to uppercase.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the string with all letters converted to uppercase. Operates on **runes** (Unicode-aware).

```go
"hello".upper()        // "HELLO"
"HeLLo WoRLd".upper()  // "HELLO WORLD"
"café".upper()         // "CAFÉ"
```

#### `trim([cutset])`

Removes leading and trailing characters.

**Arguments:**

- `cutset` (string, optional): Characters to remove. Default is whitespace.

**Returns:** `string`

**Description:** Returns the string with specified characters removed from both ends.

```go
"  hello  ".trim()              // "hello"
"xxhelloxx".trim("x")           // "hello"
"\n\t  text  \n".trim()         // "text"
"---text---".trim("-")          // "text"
```

#### `reverse()`

Reverses the string by Unicode code points.

**Arguments:** None

**Returns:** `string`

**Description:** Returns a new string with its Unicode code points in reverse order. Multi-byte UTF-8 characters are
preserved as whole code points (not reversed byte-by-byte).

```go
"hello".reverse()               // "olleh"
"їЇґҐ".reverse()                // "ҐґЇї"
"こんにちは".reverse()            // "はちにんこ"
```

#### `filter(fn)`

Filters by predicate on runes.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `string`

**Description:** Returns a string with only runes where the predicate returns `true`. Operates on **runes**.

```go
"hello123".filter(r => r >= 'a'.int() && r <= 'z'.int())  // "hello"
"a1b2c3".filter(r => r >= '0'.int() && r <= '9'.int())    // "123"
```

#### `for_each(fn)`

Executes a callback for each rune.

**Arguments:**

- `fn` (function): Callback that takes one argument `(rune)` or two arguments `(index, rune)`.

**Returns:** `undefined`

**Description:** Calls `fn` for each rune and ignores callback results except for control flow. Iteration stops when
`fn` returns falsy value. Operates on **runes**.

```go
text = ""
"abc".for_each(r => {
    text += r.string()
    return true
})
```

### Predicate Functions

#### `all(fn)`

Tests if all runes match predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if all runes satisfy the predicate. Operates on **runes**.

```go
"abc".all(r => r >= 'a'.int() && r <= 'z'.int())   // true
"abc123".all(r => r >= 'a'.int() && r <= 'z'.int()) // false
```

#### `any(fn)`

Tests if any rune matches predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if any rune satisfies the predicate. Operates on **runes**.

```go
"abc".any(r => r >= '0'.int() && r <= '9'.int())      // false
"abc123".any(r => r >= '0'.int() && r <= '9'.int())   // true
```

#### `find(fn)`

Finds byte index of first rune matching predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)`

**Returns:** `int` or `undefined`

**Description:** Returns the **byte index** of the first rune for which the predicate returns `true`. Iteration stops
on the first match. Returns `undefined` if no rune matches. Operates on **runes**, with the index reported as a byte
offset (matching the iteration index seen by `filter`, `count`, and `for_each`).

```go
"hello".find(r => r == 'l')         // 2
"hello".find(r => r == 'z')         // undefined
"hello".find((i, r) => i == 3)      // 3
```

### Aggregation Functions

#### `count(fn)`

Counts runes matching predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `int`

**Description:** Returns the number of runes where the predicate returns `true`. Operates on **runes**.

```go
"hello world".count(r => r == ' '.int())    // 1
"abc123xyz".count(r => r >= '0'.int() && r <= '9'.int())  // 3
```

### Query and Accessor Functions

#### `is_empty()`

Checks if string is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the string has zero bytes.

```go
"".is_empty()      // true
"hello".is_empty() // false
" ".is_empty()     // false
```

#### `len()`

Gets byte length.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of bytes. Note: For Unicode strings, this may be different from the rune count.

```go
"hello".len()      // 5
"hi".len()         // 2
"café".len()       // 5 (é is 2 bytes in UTF-8)
```

#### `contains(x)`

Checks if string contains substring.

**Arguments:**

- `x` (string): Substring to search for

**Returns:** `bool`

**Description:** Returns `true` if the substring is found.

```go
"hello world".contains("world")    // true
"hello world".contains("xyz")      // false
"hello".contains("")               // true (empty string in any string)
```

## Examples

### Text Processing

```go
fmt = import("fmt")
// Clean and validate input
process_name = func(name) {
    trimmed = name.trim()

    if trimmed.is_empty() {
        return error("Name cannot be empty")
    }

    if trimmed.len() > 100 {
        return error("Name too long")
    }

    return trimmed.lower()
}

result = process_name("  JOHN DOE  ")  // "john doe"
fmt.println(result)
```

### Character Filtering

```go
fmt = import("fmt")
// Remove non-alphabetic characters
alpha_only = func(s) {
    return s.filter(r =>
        (r >= 'a'.int() && r <= 'z'.int()) ||
        (r >= 'A'.int() && r <= 'Z'.int())
    )
}

clean = alpha_only("abc123def")  // "abcdef"
fmt.println(clean)
```

## Byte vs. Rune Operations

- **Byte operations** (`len`, indexing, slicing): Work with raw bytes
- **Rune operations** (`lower`, `upper`, `filter`, `all`, `any`, `count`): Aware of Unicode and work with code points

This design ensures:

- Compact UTF-8 storage for text values, keys, and printing
- Efficient byte-level access when needed
- Unicode-aware handling for the supported text operations
- Clear guidance to use `runes` when Unicode indexing, slicing, or richer sequence operations are required

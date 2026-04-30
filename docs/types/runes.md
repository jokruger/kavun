# runes

Unicode strings indexed by rune code points (not bytes).

## Overview

The `runes` type is similar to `string` but is indexed and operated on by **rune** (Unicode code point) rather than
byte. Use `runes` for Unicode-first applications where code-point-based indexing/slicing and rune-aware operations are
required across all operations.

**Key Difference from `string`:**

- `string`: Operates on both bytes (indexing, slicing) and runes (text operations)
- `runes`: All operations work on runes (code points), not bytes

## Declaration and Usage

### Unicode String Literals

Unicode string literals use the `u"..."` syntax with full escape sequence processing:

```go
s = u"ウクライナ"          // Unicode string
s2 = u"Привіт"           // Ukrainian
s3 = u"🚀🌍🎉"          // Emoji
```

### Construction via Function

```go
s2 = runes("ウクライナ")   // builtin function
s3 = runes("Hello")      // from ASCII string
```

### Indexing and Slicing (Rune-level)

All indexing and slicing operates on **runes** (code points):

```go
s = u"ウクライナ"
s[0]                     // 'ウ' (first rune)
s[-1]                    // 'ナ' (last rune)
s[0:2]                   // u"ウク" (first 2 runes)
s[:-1]                   // u"ウクライ"
s[1:5:2]                 // u"クラ"
s[4:1:-1]                // u"ナイラク"
s[::-1]                  // reversed runes
len(s)                   // 5 (rune count, not byte count)
```

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count
from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty
result. Runes also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot be
zero. Out-of-bounds index access raises `index out of bounds`.

## Member Functions

### Conversion Functions

#### `runes()`

Converts to runes.

**Arguments:** None

**Returns:** `runes`

**Description:** Returns the same runes value.

```go
u"hello".runes()   // u"hello"
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the runes as a byte-indexed UTF-8 string.

```go
u"hello".string()         // "hello"
u"ウクライナ".string()     // "ウクライナ" (as string)
```

#### `array()`

Converts to array of code points.

**Arguments:** None

**Returns:** `array`

**Description:** Returns an array where each element is a rune converted to int (code point).

```go
u"ABC".array()      // [65, 66, 67]
u"hi".array()       // [104, 105]
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `false` for empty runes, `true` otherwise.

```go
u"".bool()          // false
u"hello".bool()     // true
u" ".bool()         // true
```

#### `bytes()`

Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns a bytes value containing the UTF-8 encoding.

```go
u"ABC".bytes()      // bytes with [65, 66, 67]
```

#### `float()`

Converts to float.

**Arguments:** None

**Returns:** `float`

**Description:** Parses the runes as a floating-point number. Returns `0` when parsing fails.

```go
u"3.14".float()     // 3.14
u"invalid".float()  // 0
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Parses the runes as an integer. Returns `0` when parsing fails.

```go
u"42".int()         // 42
u"invalid".int()    // 0
```

#### `decimal()`

Converts to decimal.

**Arguments:** None

**Returns:** `decimal`

**Description:** Parses the runes as a decimal number. Invalid input results in `decimal(NaN)`.

```go
u"1.23".decimal()   // decimal(1.23)
```

#### `time()`

Converts to time.

**Arguments:** None

**Returns:** `time`

**Description:** Parses the runes as a date/time value. Invalid input results in the zero time value.

```go
u"2024-01-01".time()    // time at midnight Jan 1, 2024
```

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts runes to a record where keys are string indices (`"0"`, `"1"`, ...), and values are runes.

```go
u"abc".record()    // {"0": 'a', "1": 'b', "2": 'c'}
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts runes to a dict where keys are string indices (`"0"`, `"1"`, ...), and values are runes.

```go
u"abc".dict()       // dict({"0": 'a', "1": 'b', "2": 'c'})
```

### Transformation and Filtering Functions

#### `lower()`

Converts to lowercase.

**Arguments:** None

**Returns:** `runes`

**Description:** Returns the runes with all letters converted to lowercase. Unicode-aware.

```go
u"HELLO".lower()        // u"hello"
u"ПРИВІТ".lower()       // u"привіт"
u"Café".lower()         // u"café"
```

#### `upper()`

Converts to uppercase.

**Arguments:** None

**Returns:** `runes`

**Description:** Returns the runes with all letters converted to uppercase. Unicode-aware.

```go
u"hello".upper()        // u"HELLO"
u"привіт".upper()       // u"ПРИВІТ"
u"café".upper()         // u"CAFÉ"
```

#### `trim([cutset])`

Removes leading and trailing characters.

**Arguments:**

- `cutset` (runes, optional): Characters to remove. Default is whitespace.

**Returns:** `runes`

**Description:** Returns the runes with specified characters removed from both ends.

```go
u"  hello  ".trim()              // u"hello"
u"xxhelloxx".trim(u"x")          // u"hello"
u"---text---".trim(u"-")         // u"text"
```

#### `sort()`

Sorts runes by code point.

**Arguments:** None

**Returns:** `runes`

**Description:** Returns the runes sorted in ascending order by code point value.

```go
u"dcba".sort()          // u"abcd"
u"hello".sort()         // u"ehllo"
```

#### `chunk(size[, copy])`

Splits runes into runes chunks of up to `size` runes.

**Arguments:**

- `size` (int): Positive chunk size
- `copy` (bool, optional): When `true`, each chunk owns copied runes. Defaults to `false`.

**Returns:** `array`

**Description:** Returns an array of `runes`. The final chunk contains the remaining runes when the length is not evenly
divisible by `size`. By default, chunks are reference slices of the original runes for performance; pass `true` as the
second argument for independent chunk runes.

```go
u"hello".chunk(2)       // [u"he", u"ll", u"o"]
u"abc".chunk(10)        // [u"abc"]
u"abc".chunk(2, true)   // copied chunks
```

#### `for_each(fn)`

Executes a callback for each rune.

**Arguments:**

- `fn` (function): Callback that takes one argument `(rune)` or two arguments `(index, rune)`, and must return `bool`.

**Returns:** `undefined`

**Description:** Calls `fn` for each rune and ignores callback results except for control flow. Iteration stops when
`fn` returns `false`.

```go
text = ""
u"abc".for_each(r => {
    text += r.string()
    return true
})
```

#### `filter(fn)`

Filters by predicate on runes.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `runes`

**Description:** Returns runes where the predicate returns `true`.

```go
u"hello123".filter(r => r >= 'a'.int() && r <= 'z'.int())  // u"hello"
u"a1b2c3".filter(r => r >= '0'.int() && r <= '9'.int())    // u"123"
```

### Predicate Functions

#### `all(fn)`

Tests if all runes match predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if all runes satisfy the predicate.

```go
u"abc".all(r => r >= 'a'.int() && r <= 'z'.int())   // true
u"abc123".all(r => r >= 'a'.int() && r <= 'z'.int()) // false
```

#### `any(fn)`

Tests if any rune matches predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if any rune satisfies the predicate.

```go
u"abc".any(r => r >= '0'.int() && r <= '9'.int())      // false
u"abc123".any(r => r >= '0'.int() && r <= '9'.int())   // true
```

### Aggregation Functions

#### `count(fn)`

Counts runes matching predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(rune)` or two arguments `(index, rune)` and returns bool

**Returns:** `int`

**Description:** Returns the number of runes where the predicate returns `true`.

```go
u"hello world".count(r => r == ' '.int())    // 1
u"abc123xyz".count(r => r >= '0'.int() && r <= '9'.int())  // 3
```

#### `min()`

Finds minimum rune.

**Arguments:** None

**Returns:** `rune | undefined`

**Description:** Returns the rune with the smallest code point. Returns `undefined` for empty runes.

```go
u"hello".min()    // 'e'
u"".min()         // undefined
```

#### `max()`

Finds maximum rune.

**Arguments:** None

**Returns:** `rune | undefined`

**Description:** Returns the rune with the largest code point. Returns `undefined` for empty runes.

```go
u"hello".max()    // 'o'
u"".max()         // undefined
```

### Query and Accessor Functions

#### `is_empty()`

Checks if runes is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the runes has zero runes.

```go
u"".is_empty()      // true
u"hello".is_empty() // false
```

#### `len()`

Gets rune count.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of runes (code points).

```go
u"hello".len()      // 5
u"ウクライナ".len()  // 5
u"👋".len()        // 1 (single emoji)
```

#### `first()`

Gets first rune.

**Arguments:** None

**Returns:** `rune | undefined`

**Description:** Returns the first rune. Returns `undefined` for empty runes.

```go
u"hello".first()    // 'h'
u"ウクライナ".first() // 'ウ'
u"".first()         // undefined
```

#### `last()`

Gets last rune.

**Arguments:** None

**Returns:** `rune | undefined`

**Description:** Returns the last rune. Returns `undefined` for empty runes.

```go
u"hello".last()     // 'o'
u"".last()          // undefined
```

#### `contains(x)`

Checks if runes contains substring.

**Arguments:**

- `x` (runes): Substring to search for

**Returns:** `bool`

**Description:** Returns `true` if the substring is found.

```go
u"hello world".contains(u"world")    // true
u"hello world".contains(u"xyz")      // false
```

## Examples

### Unicode Text Processing

```go
// Process multilingual text
languages = [
    u"English",
    u"日本語",
    u"українська",
    u"العربية",
    u"Español"
]

// Count characters in each language
for lang in languages {
    println(lang.string() + ": " + lang.len().string() + " characters")
}
```

### Unicode Sorting

```go
// Sort Unicode characters
random_text = u"café"
sorted = random_text.sort()  // u"acéf"

// Sort with custom rules (if needed through filters)
text = u"aBcD"
sorted = text.sort()         // u"Bac" (by code point)
```

### Unicode Filtering

```go
// Extract only Greek letters
greek_text = u"Α Β Γ Δ"
// Filter to keep only uppercase Greek (code points in specific range)
filtered = greek_text.filter(r => r >= 0x0391 && r <= 0x03A9)
```

### Case-Insensitive Comparison

```go
// Normalize text for comparison
function normalize_for_comparison(text) {
    return text.lower().string()  // Convert to string for byte comparison
}

text1 = u"HELLO"
text2 = u"hello"

norm1 = normalize_for_comparison(text1)
norm2 = normalize_for_comparison(text2)

if norm1 == norm2 {
    println("Texts match (case-insensitive)")
}
```

### Rune-by-Rune Processing

```go
// Process each character
function process_string(s) {
    result = u""
    for i = 0; i < s.len(); i = i + 1 {
        r = s[i]  // Get i-th rune
        processed = (r.int() * 2).rune()  // Transform
        result = result + processed.string()
    }
    return result
}

input = u"abc"
output = process_string(input)
```

## Comparison with `string`

| Operation            | `string`        | `runes`               |
| -------------------- | --------------- | --------------------- |
| Indexing             | Byte-based      | Rune-based            |
| Slicing              | Byte-based      | Rune-based            |
| `len()`              | Byte count      | Rune count            |
| `first()` / `last()` | Bytes as `byte` | Rune characters       |
| Unicode operations   | Unicode-aware   | Native rune semantics |
| Raw literals         | Regular `"..."` | Unicode `u"..."`      |

Choose `string` for performance-critical byte operations, or `runes` when you need Unicode-first semantics throughout.

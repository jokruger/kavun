# bytes

Byte sequences.

## Overview

The `bytes` type represents a sequence of byte values (0-255). Use `bytes` when you need to manipulate raw byte data. Each index holds a `byte`.

## Declaration and Usage

### Construction

```go
b = bytes("abc")              // from string
b2 = [97, 98, 99].bytes()     // from array
empty = bytes()               // empty bytes
```

### Indexing and Slicing

```go
b = bytes("abc")
b[0]                          // byte(97)
b[-1]                         // byte(99)
b[0:2]                        // bytes slice
b[:-1]                        // bytes("ab")
b[1:5:2]                      // bytes("bd")
b[4:0:-1]                     // bytes("edcb")
b[::-1]                       // bytes reversed
```

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty result. Bytes also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot be zero. Out-of-bounds index access raises `index out of bounds`.

### Operations

```go
b1 = bytes("ab")
b2 = bytes("cd")
result = b1 + b2              // bytes with [97, 98, 99, 100]
```

## Member Functions

### Conversion Functions

#### `bytes()`
Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns the same bytes value.

```go
bytes("hello").bytes()    // bytes("hello")
```

#### `array()`
Converts to array of bytes.

**Arguments:** None

**Returns:** `array`

**Description:** Returns an array of `byte` values representing the bytes.

```go
bytes("ABC").array()      // [byte(65), byte(66), byte(67)]
```

#### `string()`
Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Interprets the bytes as UTF-8 and returns a string. May return invalid UTF-8 as-is.

```go
bytes("hello").string()   // "hello"
[72, 105].bytes().string()  // "Hi"
```

#### `record()`
Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts bytes to a record where keys are string indices (`"0"`, `"1"`, ...), and values are `byte` values.

```go
bytes("abc").record()   // {"0": byte(97), "1": byte(98), "2": byte(99)}
```

#### `dict()`
Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts bytes to a dict where keys are string indices (`"0"`, `"1"`, ...), and values are `byte` values.

```go
bytes("abc").dict()      // dict({"0": byte(97), "1": byte(98), "2": byte(99)})
```

### Transformation and Filtering Functions

#### `sort()`
Sorts bytes in ascending order.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns a new bytes with values sorted from smallest to largest.

```go
bytes("dcba").sort()     // bytes("abcd")
bytes([3, 1, 4, 1]).sort()  // bytes([1, 1, 3, 4])
```

#### `chunk(size[, copy])`
Splits bytes into bytes chunks of up to `size` bytes.

**Arguments:**
- `size` (int): Positive chunk size
- `copy` (bool, optional): When `true`, each chunk owns copied bytes. Defaults to `false`.

**Returns:** `array`

**Description:** Returns an array of `bytes`. The final chunk contains the remaining bytes when the length is not evenly divisible by `size`. By default, chunks are reference slices of the original bytes for performance; pass `true` as the second argument for independent chunk bytes.

```go
bytes("hello").chunk(2)   // [bytes("he"), bytes("ll"), bytes("o")]
bytes("abc").chunk(10)    // [bytes("abc")]
bytes("abc").chunk(2, true) // copied chunks
```

#### `filter(fn)`
Filters by predicate.

**Arguments:**
- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)` and returns bool

**Returns:** `bytes`

**Description:** Returns bytes containing only values where the predicate returns `true`.

```go
bytes("hello123").filter(b => b >= 'a'.int() && b <= 'z'.int())  
// bytes("hello")

bytes([1, 2, 3, 4, 5]).filter(b => b % 2 == 0)  // bytes([2, 4])
```

### Predicate Functions

#### `all(fn)`
Tests if all bytes match predicate.

**Arguments:**
- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if all bytes satisfy the predicate.

```go
bytes("abc").all(b => b >= 'a'.int() && b <= 'z'.int())   // true
bytes("abc123").all(b => b >= 'a'.int() && b <= 'z'.int()) // false
```

#### `any(fn)`
Tests if any byte matches predicate.

**Arguments:**
- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)` and returns bool

**Returns:** `bool`

**Description:** Returns `true` if any byte satisfies the predicate.

```go
bytes("abc").any(b => b >= '0'.int() && b <= '9'.int())      // false
bytes("abc123").any(b => b >= '0'.int() && b <= '9'.int())   // true
```

### Aggregation Functions

#### `count(fn)`
Counts bytes matching predicate.

**Arguments:**
- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)` and returns bool

**Returns:** `int`

**Description:** Returns the number of bytes where the predicate returns `true`.

```go
bytes("hello world").count(b => b == ' '.int())    // 1
bytes("a0b1c2").count(b => b >= '0'.int() && b <= '9'.int())  // 3
```

#### `min()`
Finds minimum byte.

**Arguments:** None

**Returns:** `byte | undefined`

**Description:** Returns the smallest byte value as a `byte`. Returns `undefined` for empty bytes.

```go
bytes("hello").min()    // byte(101)
bytes().min()           // undefined
```

#### `max()`
Finds maximum byte.

**Arguments:** None

**Returns:** `byte | undefined`

**Description:** Returns the largest byte value as a `byte`. Returns `undefined` for empty bytes.

```go
bytes("hello").max()    // byte(111)
bytes().max()           // undefined
```

### Query and Accessor Functions

#### `is_empty()`
Checks if bytes is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the bytes has zero bytes.

```go
bytes().is_empty()      // true
bytes("hello").is_empty() // false
```

#### `len()`
Gets byte count.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of bytes.

```go
bytes("hello").len()    // 5
bytes([1, 2, 3]).len()  // 3
```

#### `first()`
Gets first byte.

**Arguments:** None

**Returns:** `byte | undefined`

**Description:** Returns the first byte as a `byte`. Returns `undefined` for empty bytes.

```go
bytes("hello").first()  // byte(104)
bytes().first()         // undefined
```

#### `last()`
Gets last byte.

**Arguments:** None

**Returns:** `byte | undefined`

**Description:** Returns the last byte as a `byte`. Returns `undefined` for empty bytes.

```go
bytes("hello").last()   // byte(111)
bytes().last()          // undefined
```

#### `contains(x)`
Checks if bytes contains a value.

**Arguments:**
- `x` (int): Byte value to search for (0-255)

**Returns:** `bool`

**Description:** Returns `true` if the byte value is found.

```go
bytes("hello").contains('h'.int())    // true
bytes("hello").contains('x'.int())    // false
bytes([1, 2, 3]).contains(2)          // true
```

## Examples

### Binary Data Manipulation

```go
// Create and modify binary data
data = [0xFF, 0x00, 0x42]
data[1] = 0xAA           // Modify a byte
println(data.string())   // Print as string (may be non-printable)
```

### String Encoding/Decoding

```go
// Convert string to bytes and back
original = "Hello"
binary = original.bytes()  // Convert to bytes

// Modify
binary[0] = 'J'.int()      // Change 'H' to 'J'

result = binary.string()   // "Jello"
```

### Byte Filtering and Analysis

```go
// Filter ASCII text
text = bytes("Hello123!")
letters = text.filter(b => 
    (b >= 'A'.int() && b <= 'Z'.int()) ||
    (b >= 'a'.int() && b <= 'z'.int())
)
println(letters.string())   // "Hello"

// Extract digits
digits = text.filter(b => b >= '0'.int() && b <= '9'.int())
println(digits.string())    // "123"
```

### Data Statistics

```go
// Analyze byte distribution
data = bytes("programming")

min_byte = data.min()       // 97 ('a')
max_byte = data.max()       // 114 ('r')
total_bytes = data.len()    // 11

// Count specific bytes
letter_a_count = data.count(b => b == 'a'.int())  // 1
```

### JSON Processing

```go
// Simulate JSON manipulation
json_bytes = `{"name":"Alice","age":30}`.bytes()

// Convert to string for processing
json_str = json_bytes.string()
record = json_str.record()

// Verify data integrity
if record != undefined {
    println("Valid JSON")
}
```

## Performance Notes

- `bytes` values are reference-typed for efficiency
- `a = b` makes both variables refer to the same bytes value
- Use `copy()` to create independent copies
- Byte values must be in range 0-255; values outside this range raise errors

# Type Reference

Kavun has a comprehensive type system with scalar types (numbers, strings, runes), collections (arrays, records, dicts),
and specialized types (errors, time, ranges, and immutable wrappers).

For detailed documentation on each type including all member functions, arguments, descriptions, and examples, see the
individual type guides below.

## Scalar Types

### [undefined](types/undefined.md)

Represents the absence of a value. Returned from failed conversions and missing fields.

### [bool](types/bool.md)

Boolean values: `true` and `false`. Used in control flow and logical operations.

### [int](types/int.md)

Signed 64-bit integers. Supports decimal, hexadecimal, octal, and binary literals. Includes numeric utilities like
`sign()` and `abs()`.

### [float](types/float.md)

IEEE 754 double-precision floating-point numbers. Supports scientific notation. Note precision limitations; use
`decimal` for exact arithmetic.

### [decimal](types/decimal.md)

Exact decimal type for precise arithmetic, especially financial calculations. Includes extensive rounding and scaling
operations.

### [rune](types/rune.md)

Single Unicode code point. Useful for character operations and Unicode handling.

### [byte](types/byte.md)

Unsigned 8-bit integer (0-255). Ideal for binary data manipulation and byte-level operations.

## String Types

### [string](types/string.md)

Immutable UTF-8 encoded text optimized for compact storage, keys, printing, and basic text manipulation. Indexing and
slicing are byte-level; use `runes` for Unicode indexing. Raw strings supported via `r"..."` syntax.

### [runes](types/runes.md)

Unicode strings indexed by rune, not byte. Use `u"..."` syntax for Unicode literals. Ideal for Unicode-first operations
where rune indexing is required throughout.

### [bytes](types/bytes.md)

Byte sequences. Each element is a `byte` value (0-255). Useful for binary data manipulation.

## Collection Types

### [array](types/array.md)

Mutable, reference-typed ordered collections of heterogeneous values. Supports filtering, mapping, reduction, and
aggregation operations.

### [record](types/record.md)

Primary object type with string keys and heterogeneous values. Supports both dot notation (`r.field`) and index notation
(`r["field"]`). No member functions; fields only.

### [dict](types/dict.md)

Dictionary/map type similar to record but only supports index access for elements (`d["key"]`). Selector notation
reserved for member functions. Rich query and filtering operations.

### [range](types/range.md)

Lazy sequences of integers. Efficiently represents large sequences without memory allocation until materialized.

## Specialized Types

### [time](types/time.md)

Date and time values representing instants in time. Parse from ISO 8601 strings or Unix timestamps. Includes extensive
date/time field access and timezone handling.

### [error](types/error.md)

First-class error values carrying payloads. Errors don't interrupt execution; use conditional checks with `is_error()`
for handling.

### [immutable wrappers](types/immutable-wrappers.md)

Wrap containers (arrays, bytes, dicts, records, runes) to make them immutable at the container level. Use `immutable()`
to create and `copy()` to get mutable copies.

## Type Overview Quick Reference

| Type    | Mutability      | Indexed By | Primary Use           |
| ------- | --------------- | ---------- | --------------------- |
| int     | N/A             | N/A        | Whole numbers         |
| byte    | N/A             | N/A        | Binary data           |
| rune    | N/A             | N/A        | Unicode code points   |
| float   | N/A             | N/A        | Approximate decimals  |
| decimal | N/A             | N/A        | Exact decimals        |
| string  | Immutable       | Bytes      | Text, UTF-8 encoded   |
| runes   | Immutable       | Runes      | Text, rune indexed    |
| bytes   | Reference-typed | Bytes      | Binary data           |
| array   | Mutable         | Integers   | Ordered collections   |
| record  | Mutable         | Strings    | Object representation |
| dict    | Mutable         | Strings    | Dictionary operations |
| range   | Lazy            | Integers   | Integer sequences     |
| time    | N/A             | N/A        | Date/time values      |
| error   | N/A             | N/A        | Error handling        |

## Common Operations by Category

### Conversion Functions

Most types support conversion functions: `int()`, `float()`, `string()`, `array()`, `bool()`, `decimal()`, `time()`,
etc. Each type's documentation details its conversion capabilities.

### Text Operations

- **String byte-level**: `len()`, indexing, slicing
- **String rune-level**: `lower()`, `upper()`, `filter()`, `all()`, `any()`, `count()`
- **Runes**: rune-based `len()`, indexing, slicing, and collection-style helpers such as `first()`, `last()`, `min()`,
- and `max()`

### Collection Operations

- **Filtering**: `filter(fn)` (arrays, bytes, dicts, runes)
- **Mapping**: `map(fn)` (arrays only)
- **Chunking**: `chunk(size[, copy])` (arrays, bytes, runes)
- **Aggregation**: `sum()`, `avg()`, `min()`, `max()`, `count()`, `reduce()` (arrays, dicts)
- **Queries**: `is_empty()`, `len()`, `contains()`, `all()`, `any()`

### Lambda Callbacks

Most collection operations accept callbacks that can take one argument (value) or two (index, value):

```go
[1, 2, 3, 4].filter(x => x % 2 == 0)          // [2, 4]
[1, 2, 3].map((i, v) => i * v)                // [0, 2, 6]
[1, 2, 3].reduce(0, (acc, v) => acc + v)      // 6
```

## Reference Types

These types maintain reference semantics—assignments create references, not copies:

- **array**
- **bytes**
- **record**
- **dict**
- **immutable containers**

Use `copy()` to create independent copies of reference types.

## Value Types

These types are immutable and copied by value:

- **int**
- **float**
- **decimal**
- **rune**
- **byte**
- **string**
- **runes**
- **time**
- **error**
- **range** (lazy)
- **bool**
- **undefined**

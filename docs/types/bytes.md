# bytes

Mutable byte sequences.

## Overview

The `bytes` type represents a sequence of byte values (0-255). Use `bytes` when you need to manipulate raw byte data.
Each index holds a `byte`. Bytes are mutable and reference-typed: `a = b` makes both variables refer to the same
underlying buffer; use `copy()` to produce an independent value. Wrap with `immutable(...)` to obtain an
`immutable-bytes` value that rejects index assignment and `append_in_place` mutation.

## Declaration and Usage

### Construction

```go
b = b"abc"                   // bytes literal
b = bytes("abc")             // from string
b2 = [97, 98, 99].bytes()    // from array
empty = bytes()              // empty bytes
prealloc = bytes(3)          // bytes([0, 0, 0]) - n zero-filled bytes, n must be >= 0
same = bytes(b)              // already bytes, returned unchanged
```

`b"..."` uses the same escape rules as regular double-quoted strings and produces a `bytes` value directly.

`bytes(n)` with a single `int` argument preallocates an `n`-byte zero-filled buffer rather than attempting a
conversion (compare with [`string(n)`](string.md), where an int argument stringifies to its decimal text instead).
A negative `n` raises a recoverable `invalid_value` error. `bytes(x, fallback)` returns `fallback` when `x` isn't
convertible via `AsBytes` (a narrower set of source types than [`runes(x)`](runes.md), which additionally accepts
anything with a string representation). See [Built-in functions](../language.md#built-in-functions) for the full
constructor reference shared across `array`/`bytes`/`runes`/etc.

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

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count
from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty
result. Bytes also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot be
zero. Out-of-bounds index access raises `index out of bounds`.

### Operations

```go
b1 = bytes("ab")
b2 = bytes("cd")
result = b1 + b2              // bytes with [97, 98, 99, 100]
```

`bytes` also concatenates directly with a `byte` or `rune` scalar (either side, always producing `bytes`), and with
`string`/`runes` (see [Cross-type sequence operators](#cross-type-sequence-operators) below):

```go
b'A' + bytes("bc")            // bytes("Abc")
bytes("bc") + b'A'            // bytes("bcA")
'x' + bytes("bc")             // bytes("xbc") -- valid rune encodes as UTF-8
```

There is no implicit conversion of any other type — `bytes(...) + 5`, `bytes(...) + true`, etc. are all runtime
errors, the same as [`string`](string.md#concatenation)'s rejection of implicit stringification.

### Removal (`-`)

`-` removes every occurrence of the right-hand operand from the left-hand `bytes`, returning a new `bytes` (the
receiver is never mutated). It only ever reads "remove this from that" — there's no reversed form (`byte - bytes`
or similar is a runtime error, not a differently-shaped removal).

`bytes` owns every pairing it's in for removal too, same as it does for `+` and ordering — `byte`/`rune`/`string`/
`bytes`/`runes` are all accepted on the right, since every one of them already has an exact byte encoding (the
same conversions `+`/ordering already use):

```go
bytes("abcabc") - b'a'        // bytes("bcbc") -- drop every occurrence of that byte
bytes("abcabc") - bytes("bc") // bytes("aa")   -- drop every occurrence of that subsequence
bytes("abcabc") - b'z'        // bytes("abcabc") -- no occurrences, unchanged
bytes("banana") - "an"        // bytes("ba")   -- string encodes to bytes first, same as + does
bytes("banana") - runes("an") // bytes("ba")
```

### Cross-type sequence operators

`string`, `bytes`, and `runes` share a fixed precedence for `+` and ordering (`< > <= >=`) whenever two different
sequence types combine: **`bytes` > `runes` > `string`**, always — `bytes` is the highest rank, so it's always the
result type (for `+`) or comparison basis (for ordering) no matter which side of the operator it's written on:

```go
bytes("b") + "a"              // bytes, contents "ba"
"a" + bytes("b")              // bytes, contents "ab" -- same result type either order, content order still
                               // follows which operand was written first
bytes("b") + runes("a")       // bytes, contents "ba"
bytes("abc") < "abd"          // true -- compares as bytes
"abc" < bytes("abd")          // true -- same comparison, either order
```

The `byte`/`rune` scalars join a sequence but never carry rank themselves — they always produce whichever sequence
type they're joining, and each pairs with a fixed subset of the three sequence types:

| Scalar | Pairs with            | Does **not** pair with |
| ------ | ---------------------- | ------------------------ |
| `byte` | `bytes` only            | `string`, `runes` — an arbitrary byte isn't guaranteed valid UTF-8 |
| `rune` | `bytes`, `runes`, `string` | — (a valid rune is safe to join any of the three) |

```go
b'A' + "bc"                   // runtime error -- byte does not pair with string
b'A' + runes("bc")            // runtime error -- byte does not pair with runes
'x' + runes("bc")             // runes, contents "xbc"
'x' + "bc"                    // string, contents "xbc"
```

See [Extending types: operators](../extending-types.md) for the reasoning behind this fixed precedence (mutable
working buffers outrank the immutable literal type feeding them) and the general cross-type operator model.

### Equality

```go
bytes("abc") == bytes("abc")   // true
```

`bytes` recognizes more types for `==`/`!=` than for `+`/ordering — it's the top of the cross-type comparison
hierarchy, recognizing every other type this document's cross-type model covers, and never delegates:

```go
bytes("hello") == "hello"          // true -- string's own already-exact AsBytes() encoding
"hello" == bytes("hello")          // true -- same result either order
bytes("hello") == runes("hello")   // true

bytes("5") == 5              // true -- 5's canonical text form ("5") encoded down to bytes
bytes("true") == true        // true
```

Encoding always goes *into* `bytes`, never out of it — `string`/`runes` convert down via their own exact `AsBytes`,
and the exact-chain/`float` types convert to their own canonical text form first, then encode that down the same
way. This is deliberate: decoding arbitrary `bytes` *up* into text could fail outright, since `bytes` isn't
guaranteed to hold valid UTF-8 at all, but encoding text *down* into raw bytes always succeeds.

A `bytes` value is **not** accidentally equal to an unrelated `array` of matching integer values, even though
`array` can convert itself to `bytes` for other purposes (e.g. `[104, 101, 108, 108, 111].bytes()`):

```go
bytes("hello") == [104, 101, 108, 108, 111]   // false -- array is not one of the recognized types
```

### Mutation

Bytes support index assignment and the `append()`/`append_in_place()` member functions:

```go
b = bytes("hello")
b[0] = 'H'                    // bytes("Hello")
b[-2] = '!'                   // bytes("Hel!o")
b[0] = 65                     // numeric byte value (0-255)

b2 = b.append('X')            // b2 is a NEW, independent bytes value; b is unchanged
b2 = b.append('X', 'Y')       // append multiple bytes
b2 = b.append(bytes("!!"))    // append another bytes value

b.append_in_place('X')        // mutates b's own shared body in place; every alias into b sees the change
```

`append()` always returns an independent copy — the source is never mutated, even with zero items, and it works
regardless of the source's mutability. `append_in_place()` is the explicit mutating twin: it rejects an immutable
receiver and mutates the shared underlying buffer directly, so every other variable sharing that buffer (via plain
assignment, `b2 = b`) observes the change too. `array`/`bytes`/`runes` also support `splice()`/`splice_in_place()`
for insert/remove-by-position — same pure/mutating split as `append`/`append_in_place`. Index assignment requires
the right-hand side to fit in a byte (0-255); other types or out-of-range values raise an error. Out-of-bounds
indices raise `index out of bounds`.

Wrapping with `immutable(...)` prevents both index assignment and any other mutation; reads continue to work normally:

```go
ib = immutable(bytes("abc"))
type_name(ib)                 // "immutable-bytes"
ib[0] = 'X'                   // runtime error - immutable
copy(ib)                      // returns a mutable copy
```

## Member Functions

### General Functions

#### `copy()`

Returns a deep, mutable copy of the bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Equivalent to the builtin `copy(x)`. The result is an independent value; mutations to the copy do not
affect the original. When called on an `immutable-bytes`, the returned copy is mutable. See
[container semantics](container-semantics.md) for details.

```go
b = bytes("abc")
c = b.copy()
c[0] = 'X'
// b is still bytes("abc"), c is bytes("Xbc")
```

#### `copy_shallow()`

Returns a copy of the bytes value.

**Arguments:** None

**Returns:** `bytes`

**Description:** `bytes` elements are raw bytes, not nested `Value`s, so there's no depth for "shallow" vs.
"deep" to actually differ on — `copy_shallow()` behaves identically to `copy()`. Kept as a real, separately
callable spelling anyway for member-call-surface consistency with `array`/`dict`, where the two genuinely do
differ. When called on an `immutable-bytes`, the returned copy is mutable.

```go
bytes("abc").copy_shallow() == bytes("abc").copy()   // true
```

#### `freeze()`

Returns a fully independent, immutable copy of the bytes value.

**Arguments:** None

**Returns:** `bytes` (immutable)

**Description:** Equivalent to `copy()` followed by marking the fresh clone immutable. Always detaches first, so
the source and every existing alias into it are completely unaffected. For the explicit twin that skips the
detach, see `freeze_shallow()`.

```go
b = bytes("abc")
f = b.freeze()
is_immutable(f)    // true
b[0] = 'X'
f                  // bytes("abc") - unaffected
```

#### `freeze_shallow()`

Marks the bytes value's own header immutable without detaching.

**Arguments:** None

**Returns:** `bytes` (immutable)

**Description:** Genuinely pure — never mutates anything reachable, just returns a new header with the
immutable flag set, pointing at the *same* shared body. Requires reassignment to affect your own variable
(`b = b.freeze_shallow()`), and a pre-existing sibling binding into the same body stays independently mutable
and can still change what the "frozen" variable sees. See
[container semantics](container-semantics.md#interaction-with-freeze-freeze_shallow) for the full contract.

```go
b = bytes("abc")
b = b.freeze_shallow()
is_immutable(b)    // true
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
bytes([72, 73]).format()       // "HI"
bytes([1, 2, 3]).format("v")   // "bytes([1, 2, 3])"
```

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

#### `sort_in_place()`

Sorts bytes in place, in ascending order.

**Arguments:** None

**Returns:** `bytes` (the receiver)

**Description:** Sorts the receiver's own backing storage directly — visible through every existing alias into
the receiver without needing reassignment. Rejects an immutable receiver.

```go
b = bytes("dcba")
c = b                  // c shares b's body
b.sort_in_place()
b                       // bytes("abcd")
c                       // bytes("abcd") - c sees it too
immutable(bytes("ba")).sort_in_place()   // Error: not_sortable
```

#### `dedup()`

Removes consecutive duplicate bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns new bytes where each run of consecutive equal byte values is collapsed into a single byte.
Order is preserved. Pair with `sort()` to fully deduplicate.

```go
bytes("aabbccd").dedup()              // bytes("abcd")
bytes("hello").sort().dedup()         // bytes("ehlo")
bytes([1, 1, 2, 2, 3]).dedup()        // bytes([1, 2, 3])
```

#### `unique()`

Removes all duplicate bytes regardless of position.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns new bytes containing only the first occurrence of each byte value, preserving original order.

```go
bytes("hello").unique()               // bytes("helo")
bytes("abab").unique()                // bytes("ab")
bytes([3, 1, 2, 1, 3, 2]).unique()    // bytes([3, 1, 2])
```

#### `reverse()`

Reverses bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Returns a new bytes with byte values in reverse order.

```go
bytes("hello").reverse()        // bytes("olleh")
bytes([1, 2, 3]).reverse()      // bytes([3, 2, 1])
```

#### `reverse_in_place()`

Reverses bytes in place.

**Arguments:** None

**Returns:** `bytes` (the receiver)

**Description:** Reverses the receiver's own byte order directly — visible through every existing alias into
the receiver without needing reassignment. Rejects an immutable receiver.

```go
b = bytes("abc")
c = b
b.reverse_in_place()
b                       // bytes("cba")
c                       // bytes("cba") - c sees it too
immutable(bytes("ab")).reverse_in_place()   // Error: not_reversible
```

#### `slice(start, end)`

Returns a copy of a sub-range of the bytes.

**Arguments:**

- `start` (int, optional): Start index, inclusive. Defaults to `0`. Negative values count from the end.
- `end` (int, optional): End index, exclusive. Defaults to the bytes' length. Negative values count from the end.

**Returns:** `bytes`

**Description:** Member-function spelling of the `b[start:end]` operator. Always returns an independently-owned
copy, regardless of the receiver's mutability. For the explicit performance opt-in that shares backing storage
instead, see `slice_view(start, end)`.

```go
b = bytes("hello")
b.slice()          // bytes("hello")
b.slice(1)         // bytes("ello")
b.slice(1, 3)      // bytes("el")
```

#### `slice_view(start, end)`

Returns a view of a sub-range that shares backing storage with the source.

**Arguments:**

- `start` (int, optional): Start index, inclusive. Defaults to `0`. Negative values count from the end.
- `end` (int, optional): End index, exclusive. Defaults to the bytes' length. Negative values count from the end.

**Returns:** `bytes` (`is_view()` reports `true`)

**Description:** The explicit sharing twin of `slice()` — a raw re-slice that shares the source's underlying
storage instead of copying. Mutating the result mutates the source (and vice versa). See
[container semantics](container-semantics.md#slicing-and-chunking-views) for the full danger/idiom writeup.

```go
b = bytes("hello")
s = b.slice_view(1, 3)
s[0] = 'X'
b               // bytes("hXllo") - the source changed too
s.is_view()     // true
```

#### `is_view()`

Reports whether the bytes value shares backing storage with some other value.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` only for values actually produced by `slice_view()` or `chunk_view()` — not for
plain `bytes` literals, `copy()`/`copy_shallow()` results, or `chunk()`'s own default output.

```go
bytes("abc").is_view()                      // false
bytes("abc").slice_view(0, 1).is_view()     // true
```

#### `repeat(n)`

Repeats bytes `n` times by concatenation.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `bytes`

**Description:** Returns new bytes with the original bytes concatenated `n` times. Returns empty bytes when `n == 0` or
when the receiver is empty. Errors when `n < 0`.

```go
"AB".bytes().repeat(3)          // bytes([65, 66, 65, 66, 65, 66])
"".bytes().repeat(5)            // empty bytes
```

#### `append(...)`

Returns new bytes with the given items added.

**Arguments:**

- `...items` (byte | rune | bytes | runes, 0 or more): Values to append. A `bytes`/`runes` argument spreads
  element-by-element, matching this operation's own flattening convention — it's never nested as one element.

**Returns:** `bytes`

**Description:** Always returns fresh, independently-owned bytes — never touches the receiver's backing
storage, works regardless of the receiver's mutability, even with zero items. For amortized O(n) growth in a
loop, use `append_in_place()` instead; see [container semantics](container-semantics.md#append).

```go
b = bytes("ab")
b.append('c')             // bytes("abc")
b.append(bytes("cd"))     // bytes("abcd") - spreads, doesn't nest
b                          // bytes("ab") - source untouched
```

#### `append_in_place(...)`

Appends items to bytes in place.

**Arguments:** Same as `append()`.

**Returns:** `bytes` (the receiver)

**Description:** Mutates the receiver's own shared body directly — visible through every existing alias without
reassignment. Rejects an immutable receiver. Zero items is a legal no-op. This is genuinely new capability for
`bytes` (no sharing form of `append` existed before this), not a rename. See
[container semantics](container-semantics.md#append-in-place-aliasing) for the full aliasing contract.

```go
b = bytes("ab")
c = b
b.append_in_place('c')
b, c   // bytes("abc") bytes("abc") - c sees it too
immutable(bytes("a")).append_in_place('b')   // Error: not_appendable
```

#### `splice(start[, delete_count[, ...items]])`

Returns new bytes with a range removed and/or items inserted.

**Arguments:**

- `start` (int): Start index. Must be within `[0, len]`.
- `delete_count` (int, optional): Number of bytes to remove starting at `start`. Defaults to "everything from
  `start` to the end." Must be non-negative; clamped if it would run past the end.
- `...items` (byte | rune | bytes | runes, 0 or more): Values to insert at `start`, after the deletion — a
  `bytes`/`runes` argument spreads element-by-element, same convention as `append()`.

**Returns:** `bytes` (the value after the operation — not the deleted items)

**Description:** Always builds genuinely fresh bytes — never aliases the receiver — and works regardless of the
receiver's mutability. For the mutating twin that returns the deleted bytes instead, see `splice_in_place()`.

```go
b = bytes("abc")
b.splice(1, 0, bytes("xy"))   // bytes("axybc")
b                              // bytes("abc") - source untouched
```

#### `splice_in_place(start[, delete_count[, ...items]])`

Removes a range and/or inserts items into bytes in place.

**Arguments:** Same as `splice()`.

**Returns:** `bytes` of the deleted elements (not the modified value)

**Description:** Mutates the receiver's own shared body directly — visible through every existing alias without
reassignment. Rejects an immutable receiver.

```go
b = bytes("abc")
deleted = b.splice_in_place(0, 1)
deleted    // bytes("a")
b          // bytes("bc")
```

#### `split([sep[, n]])`

Splits the bytes into an array of bytes.

**Arguments:**

- `sep` (bytes | byte | string | rune, optional): Separator. If omitted, splits on runs of ASCII whitespace
  (`' '`, `'\t'`, `'\n'`, `'\r'`, `'\v'`, `'\f'`) and drops empty pieces.
- `n` (int, optional): Maximum number of splits. `0` performs no splits. Negative values mean unlimited.

**Returns:** `array of bytes`

**Description:** With a literal separator, leading/trailing/consecutive separators produce empty pieces. Empty receiver
returns an empty array. Separator must not be empty when provided.

```go
bytes("a,b,c").split(",")             // [bytes("a"), bytes("b"), bytes("c")]
bytes("a,b,c").split(byte(0x2C))      // same
bytes("a b c").split()                // [bytes("a"), bytes("b"), bytes("c")]
bytes("").split(",")                  // []
```

#### `split_lines()`

Splits on `\n`, `\r\n`, or `\r`. Trailing line terminator does not produce an extra empty trailing element.

**Returns:** `array of bytes`

```go
bytes("a\nb\nc").split_lines()        // [bytes("a"), bytes("b"), bytes("c")]
```

#### `partition(sep)`

Splits at the first occurrence of `sep` into 3 pieces.

**Arguments:**

- `sep` (bytes | byte | string | rune): Non-empty separator.

**Returns:** `array of bytes` of length 3: `[before, sep, after]`. If `sep` not found, returns
`[receiver, bytes(""), bytes("")]`.

```go
bytes("k=v").partition("=")           // [bytes("k"), bytes("="), bytes("v")]
bytes("abc").partition("x")           // [bytes("abc"), bytes(""), bytes("")]
```

#### `chunk(size)`

Splits bytes into bytes chunks of up to `size` bytes.

**Arguments:**

- `size` (int): Positive chunk size

**Returns:** `array`

**Description:** Returns an array of `bytes`. The final chunk contains the remaining bytes when the length is not evenly
divisible by `size`. Every chunk is an independent copy — mutating a chunk never affects the source. For the explicit
performance opt-in that shares backing storage instead, see `chunk_view(size)` in
[container semantics](container-semantics.md#slicing-and-chunking-views).

```go
bytes("hello").chunk(2)   // [bytes("he"), bytes("ll"), bytes("o")]
bytes("abc").chunk(10)    // [bytes("abc")]
```

#### `chunk_view(size)`

Splits bytes into chunks that share backing storage with the source.

**Arguments:**

- `size` (int): Positive chunk size.

**Returns:** `array` of `bytes` (each chunk's `is_view()` reports `true`)

**Description:** The explicit sharing twin of `chunk()` — each chunk is a raw re-slice into the source's own
backing storage rather than an independent copy. Mutating a chunk mutates the corresponding bytes of the
source. See [container semantics](container-semantics.md#slicing-and-chunking-views) for the full danger/idiom
writeup.

```go
b = bytes("hello")
chunks = b.chunk_view(2)
chunks[0][0] = 'X'
b   // bytes("Xello") - the source changed too
```

#### `for_each(fn)`

Executes a callback for each byte.

**Arguments:**

- `fn` (function): Callback that takes one argument `(byte)` or two arguments `(index, byte)`.

**Returns:** `undefined`

**Description:** Calls `fn` for each byte and ignores callback results except for control flow. Iteration stops when
`fn` returns falsy value.

```go
total = 0
bytes("abc").for_each(b => {
    total += b
    return true
})
```

#### `filter(fn)` / `filter()`

Filters by predicate, or filters out zero values when called without arguments.

**Arguments:**

- `fn` (function, optional): Predicate function. Accepts one argument (value) or two (index, value). When omitted, all
  zero elements are removed.

**Returns:** `bytes`

**Description:** Returns bytes containing only values where the predicate returns `true`. If called with no arguments,
returns a new bytes with all zero elements removed.

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

#### `find(fn)`

Finds index of first byte matching predicate.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)`

**Returns:** `int` or `undefined`

**Description:** Returns the index of the first byte for which the predicate returns `true`. Iteration stops on the
first match. Returns `undefined` if no byte matches.

```go
bytes("hello").find(b => b == 'l')         // 2
bytes("hello").find(b => b == 'z')         // undefined
bytes("hello").find((i, b) => i == 3)      // 3
```

### Aggregation Functions

#### `count(fn)` / `count()`

Counts bytes matching predicate or counts non-zero elements when called without arguments.

**Arguments:**

- `fn` (function): Predicate that takes one argument `(byte)` or two arguments `(index, byte)` and returns bool

**Returns:** `int`

**Description:** Returns the number of bytes where the predicate returns `true`. If called with no arguments, returns
the number of non-zero bytes.

```go
bytes("hello world").count(b => b == ' '.int())    // 1
bytes("a0b1c2").count(b => b >= '0'.int() && b <= '9'.int())  // 3
```

#### `sum()`

Sums the numeric byte values.

**Arguments:** None

**Returns:** `int | undefined`

**Description:** Returns the sum of all byte values as an `int`. Returns `undefined` for empty bytes.

```go
bytes([1, 2, 3]).sum()        // 6
bytes("abc").sum()            // 294
bytes().sum()                 // undefined
```

#### `avg()`

Computes the integer average of byte values.

**Arguments:** None

**Returns:** `int | undefined`

**Description:** Returns the integer average (floor division) of all byte values. Returns `undefined` for empty bytes.

```go
bytes([2, 4, 6]).avg()        // 4
bytes("abc").avg()            // 98
bytes().avg()                 // undefined
```

#### `map(fn)`

Maps each byte through a callback.

**Arguments:**

- `fn` (function): Callback that takes one argument `(byte)` or two arguments `(index, byte)`.

**Returns:** `array`

**Description:** Returns a new array where each element is the result of `fn` applied to the corresponding byte. The
result is an `array` because the callback may return values of any type.

```go
bytes("abc").map(b => b + 1)              // [98, 99, 100]
bytes("abc").map((i, b) => [i, b])        // [[0, byte(97)], [1, byte(98)], [2, byte(99)]]
```

#### `reduce(initial, fn)`

Reduces bytes to a single value.

**Arguments:**

- `initial`: The initial accumulator value
- `fn` (function): Reducer that takes `(acc, byte)` or `(acc, index, byte)` and returns the new accumulator.

**Returns:** Whatever the reducer returns

**Description:** Folds the bytes from left to right using `fn`, starting with `initial`.

```go
bytes("abc").reduce(0, (acc, b) => acc + b)               // 294
bytes("abc").reduce("", (acc, i, b) => acc + b.string())  // "abc"
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
data.string()            // Print as string (may be non-printable)
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
fmt := import("fmt")
// Filter ASCII text
text = bytes("Hello123!")
letters = text.filter(b =>
    (b >= 'A'.int() && b <= 'Z'.int()) ||
    (b >= 'a'.int() && b <= 'z'.int())
)
fmt.println(letters.string())   // "Hello"

// Extract digits
digits = text.filter(b => b >= '0'.int() && b <= '9'.int())
fmt.println(digits.string())    // "123"
```

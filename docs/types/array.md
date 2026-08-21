# array

Mutable collections of heterogeneous values.

## Overview

Arrays are ordered, mutable collections that can hold values of any type. Arrays are reference-typed, meaning `a = b`
makes both variables point to the same array; to get an independent copy, use `copy()`.

## Declaration and Usage

### Array Literals

```go
a = [1, 2, 3]
b = ["hello", "world"]
c = [1, "two", 3.0, true]  // mixed types
empty = []
```

### Construction

```go
// Via the array() builtin
empty = array()                       // []
prealloc = array(3)                   // [undefined, undefined, undefined] (n must be >= 0)
same = array([1, 2, 3])               // [1, 2, 3] (already an array, returned unchanged)
from_range = array(range(1, 4))       // [1, 2, 3] (converted via range's AsArray)
fallback = array(true, [9])           // [9] (bool doesn't convert, fallback used)

// From other types, via member function
from_range2 = range(1, 4).array()     // [1, 2, 3]
from_string = "abc".array()           // [97, 98, 99]
```

`array(n)` with a single `int` argument preallocates an `n`-element array of `undefined` values rather than
attempting a conversion — this is different from most other conversion builtins, where an int argument goes
through normal conversion (compare with [`string(n)`](string.md), which stringifies the number). A negative `n`
raises a recoverable `invalid_value` error. See [Built-in functions](../language.md#built-in-functions) for the
full constructor reference shared across `array`/`bytes`/`runes`/etc.

### Reference Semantics

```go
a = [1, 2, 3]
b = a
a[0] = 99
b[0]             // 99 (both point to same array)

c = copy(a)      // Independent copy
a[0] = 1
c[0]             // 99 (c is unchanged)
```

### Indexing and Slicing

```go
a = [10, 20, 30, 40, 50]
a[0]             // 10
a[2]             // 30
a[0:2]           // [10, 20]
a[2:4]           // [30, 40]
a[-1]            // 50 (last element)
a[:-1]           // [10, 20, 30, 40]
a[-3:-1]         // [30, 40]
a[4:2]           // []
a[1:5:2]         // [20, 40]
a[5:1:-1]        // [50, 40, 30, 20]
a[::-1]          // [50, 40, 30, 20, 10]
```

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count
from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty
result. Arrays also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot be
zero. Out-of-bounds index access raises `index out of bounds`.

### Mutation

```go
a = [1, 2, 3]
a[0] = 99        // Change element
a[5] = 100       // Error: index out of bounds
```

## Operators

`array` supports exactly one operator: same-type `+`, which concatenates.

```go
[1, 2] + [3, 4]        // [1, 2, 3, 4]
[[1], [2]] + [3]        // [[1], [2], 3] -- concatenates elements; [3] is not appended as one element
```

No other operator is defined for `array` — not appending a scalar, not prepending, not `-` in any form (including
`array - array`):

```go
[1, 2] + 3     // runtime error: invalid_binary_operator: array + int
3 + [1, 2]     // runtime error: invalid_binary_operator: int + array
[1, 2] - [1]   // runtime error: invalid_binary_operator: array - array
```

This is deliberate, not a missing feature. An array's elements can be any type, including another array, so
`arr + x` for a non-array `x` would be genuinely ambiguous — "append `x` as one new element" and "concatenate
`x`'s elements in" are two different operations that only diverge based on `x`'s incidental type, and a nested
array is an entirely ordinary thing to want to append as a single element. Rather than pick one reading silently,
`array` only defines `+` where there's no ambiguity: same-type concatenation. `-` has no single meaning either
(remove-by-value? set-difference? positional diff?), so it's not defined at all, for any right-hand type.

## Member Functions

### General Functions

#### `copy()`

Returns a deep, mutable copy of the array.

**Arguments:** None

**Returns:** `array`

**Description:** Equivalent to the builtin `copy(x)`. The result is an independent value; mutations to the copy do not
affect the original. When called on an `immutable-array`, the returned copy is mutable. See
[container semantics](container-semantics.md) for details.

```go
a = [1, 2, 3]
b = a.copy()
b[0] = 99
// a is still [1, 2, 3], b is [99, 2, 3]
```

#### `copy_shallow()`

Returns a shallow, mutable copy of the array.

**Arguments:** None

**Returns:** `array`

**Description:** Clones only the top-level array — a fresh, independently-owned element list — but nested
values are shared with the source, not recursively cloned. Use this when you only need to stop the outer array
from growing/aliasing the original and don't need nested content protected too; use `copy()` for a fully
independent deep clone. When called on an `immutable-array`, the returned copy is mutable. See
[container semantics](container-semantics.md) for the copy-vs-view contract.

```go
a = [[1, 2], [3, 4]]
b = a.copy_shallow()
b[0] = 99          // top level is independent
b[1][0] = 88       // but nested arrays are still shared
a[1][0]            // 88 - a sees the nested mutation
a[0]               // [1, 2] - a's own top level is untouched
```

#### `freeze()`

Returns a fully independent, deep-immutable copy of the array.

**Arguments:** None

**Returns:** `array` (immutable)

**Description:** Equivalent to `copy()` followed by recursively marking the fresh clone (and everything nested
inside it) immutable. Always detaches first, so the source and every existing alias into it are completely
unaffected — freezing a value never surprises whoever else still holds it. For the explicit twin that skips the
detach (and so does *not* protect against another live, still-mutable alias into the same body), see
`freeze_shallow()`.

```go
a = [1, 2, 3]
f = a.freeze()
is_immutable(f)    // true
is_immutable(a)    // false - the source is untouched
a[0] = 99
f                  // [1, 2, 3] - unaffected
```

#### `freeze_shallow()`

Marks the array's own header immutable without detaching.

**Arguments:** None

**Returns:** `array` (immutable)

**Description:** This is genuinely pure, like `copy_shallow()` — it never mutates anything reachable, it just
returns a new header with the immutable flag set pointing at the *same* shared body. You must reassign the
result to see the effect on your own variable (`a = a.freeze_shallow()`), and — because nothing was detached — a
pre-existing sibling binding into the same body stays independently mutable, and mutating through it remains
visible through the "frozen" variable too, since both still share the same underlying storage. This is
deliberately *not* the same shape as `append_in_place`/`splice_in_place`: those mutate the shared body and need
no reassignment; this only ever changes the header you already hold. See
[container semantics](container-semantics.md#interaction-with-freeze-freeze_shallow) for the full contract, and
`copy_shallow().freeze_shallow()` for how to get a "shallow freeze" (detach the top level, then protect only it).

```go
a = [1, 2, 3]
a = a.freeze_shallow()
is_immutable(a)    // true

b = [1, 2, 3]
c = b              // c shares b's body
b = b.freeze_shallow()
is_immutable(c)    // false - c was never reassigned, still independently mutable
c[0] = 99
b[0]               // 99 - the shared body changed; b was never protected from it
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
[1, 2, 3].format()           // "[1, 2, 3]"
[1, 2, 3].format("v")        // "[1, 2, 3]"
```

### Conversion Functions

#### `array()`

Converts to array.

**Arguments:** None

**Returns:** `array`

**Description:** Returns the same array.

```go
[1, 2, 3].array()    // [1, 2, 3]
```

#### `bytes()`

Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Converts array elements to bytes (elements must be 0-255).

```go
[72, 101, 108, 108, 111].bytes()  // bytes("Hello")
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts elements to runes and builds a string from them.

```go
[72, 101, 108, 108, 111].string()  // "Hello"
```

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts array to record where keys are string indices (`"0"`, `"1"`, ...).

```go
[48, 49, -1].record()   // {"0": 48, "1": 49, "2": -1}
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts array to dict where keys are string indices (`"0"`, `"1"`, ...).

```go
[48, 49, -1].dict()     // dict({"0": 48, "1": 49, "2": -1})
```

### Transformation and Filtering Functions

#### `sort()`

Sorts array elements.

**Arguments:** None

**Returns:** `array`

**Description:** Sorts the array in ascending order. Elements must be comparable.

```go
[3, 1, 4, 1, 5].sort()         // [1, 1, 3, 4, 5]
["c", "a", "b"].sort()         // ["a", "b", "c"]
```

#### `sort_in_place()`

Sorts array elements in place.

**Arguments:** None

**Returns:** `array` (the receiver)

**Description:** Sorts the receiver's own backing storage directly, in ascending order — the mutation is
visible through every existing alias into the receiver without needing reassignment, same shape as
`append_in_place`/`splice_in_place`. Rejects an immutable receiver. Elements must be comparable.

```go
a = [3, 1, 4, 1, 5]
b = a                  // b shares a's body
a.sort_in_place()
a                       // [1, 1, 3, 4, 5]
b                       // [1, 1, 3, 4, 5] - b sees it too, no reassignment needed
immutable([3, 1]).sort_in_place()   // Error: not_sortable
```

#### `dedup()`

Removes consecutive duplicate elements.

**Arguments:** None

**Returns:** `array`

**Description:** Returns a new array where each run of consecutive equal elements is collapsed into a single element.
Order is preserved. Pair with `sort()` to fully deduplicate a sequence in O(n log n).

```go
[1, 1, 2, 2, 3, 3, 3, 1].dedup()       // [1, 2, 3, 1]
[3, 1, 2, 1, 3, 2].sort().dedup()      // [1, 2, 3]
["a", "a", "b", "a"].dedup()           // ["a", "b", "a"]
```

#### `unique()`

Removes all duplicate elements regardless of position.

**Arguments:** None

**Returns:** `array`

**Description:** Returns a new array containing only the first occurrence of each element, preserving original order.
Equality is determined element by element, so this works on any element type but runs in O(n²) time.

```go
[1, 1, 2, 2, 3, 3, 3, 1].unique()      // [1, 2, 3]
[3, 1, 2, 1, 3, 2].unique()            // [3, 1, 2]
["a", "b", "a", "c", "b"].unique()     // ["a", "b", "c"]
```

#### `reverse()`

Reverses the array.

**Arguments:** None

**Returns:** `array`

**Description:** Returns a new array with elements in reverse order.

```go
[].reverse()                   // []
[1, 2, 3].reverse()            // [3, 2, 1]
["a", "b", "c"].reverse()      // ["c", "b", "a"]
```

#### `reverse_in_place()`

Reverses the array in place.

**Arguments:** None

**Returns:** `array` (the receiver)

**Description:** Reverses the receiver's own element order directly — visible through every existing alias into
the receiver without needing reassignment. Rejects an immutable receiver.

```go
a = [1, 2, 3]
b = a
a.reverse_in_place()
a                       // [3, 2, 1]
b                       // [3, 2, 1] - b sees it too
immutable([1, 2]).reverse_in_place()   // Error: not_reversible
```

#### `slice(start, end)`

Returns a copy of a sub-range of the array.

**Arguments:**

- `start` (int, optional): Start index, inclusive. Defaults to `0`. Negative values count from the end.
- `end` (int, optional): End index, exclusive. Defaults to the array's length. Negative values count from the end.

**Returns:** `array`

**Description:** Member-function spelling of the `a[start:end]` operator — the two are equivalent, `slice()`
just gives it a name usable in a call chain. Always returns an independently-owned copy, regardless of the
receiver's mutability; mutating the result never affects the source. For the explicit performance opt-in that
shares backing storage instead, see `slice_view(start, end)`.

```go
a = [10, 20, 30, 40, 50]
a.slice()          // [10, 20, 30, 40, 50] - full copy
a.slice(1)         // [20, 30, 40, 50]
a.slice(1, 3)      // [20, 30]
a.slice(1, 3) == a[1:3]   // true - same result as the operator
```

#### `slice_view(start, end)`

Returns a view of a sub-range that shares backing storage with the source.

**Arguments:**

- `start` (int, optional): Start index, inclusive. Defaults to `0`. Negative values count from the end.
- `end` (int, optional): End index, exclusive. Defaults to the array's length. Negative values count from the end.

**Returns:** `array` (`is_view()` reports `true`)

**Description:** The explicit sharing twin of `slice()` — a raw re-slice that shares the source's underlying
storage instead of copying. Mutating the result mutates the source (and vice versa), and growing it through
`append_in_place()` can silently reallocate and detach it from the source without warning. See
[container semantics](container-semantics.md#slicing-and-chunking-views) for the full danger/idiom writeup
before using this outside a tight, well-understood loop.

```go
a = [1, 2, 3, 4, 5]
b = a.slice_view(1, 3)
b[0] = 99
a               // [1, 99, 3, 4, 5] - the source changed too
b.is_view()     // true
```

#### `is_view()`

Reports whether the array shares backing storage with some other value.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` only for values actually produced by `slice_view()` or `chunk_view()` — not for
plain array literals, not for `copy()`/`copy_shallow()` results, and (deliberately, for now) not for `chunk()`'s
own default output either, even though nothing else has ever set it to `true` for a value that doesn't actually
share storage.

```go
[1, 2, 3].is_view()                        // false
[1, 2, 3].slice_view(1, 2).is_view()       // true
[1, 2, 3].slice_view(1, 2).copy().is_view() // false - copy() always detaches
```

#### `chunk(size)`

Splits an array into arrays of up to `size` elements.

**Arguments:**

- `size` (int): Positive chunk size

**Returns:** `array`

**Description:** Returns an array of arrays. The final chunk contains the remaining elements when the length is not
evenly divisible by `size`. Every chunk is an independent copy — mutating a chunk never affects the source array.
For the explicit performance opt-in that shares backing storage instead, see `chunk_view(size)` in
[container semantics](container-semantics.md#slicing-and-chunking-views).

```go
[1, 2, 3, 4, 5].chunk(2)       // [[1, 2], [3, 4], [5]]
[1, 2, 3].chunk(10)            // [[1, 2, 3]]
```

#### `chunk_view(size)`

Splits an array into views that share backing storage with the source.

**Arguments:**

- `size` (int): Positive chunk size.

**Returns:** `array` of `array` (each chunk's `is_view()` reports `true`)

**Description:** The explicit sharing twin of `chunk()` — each chunk is a raw re-slice into the source's own
backing array rather than an independent copy. Mutating a chunk mutates the corresponding elements of the
source. See [container semantics](container-semantics.md#slicing-and-chunking-views) for the full danger/idiom
writeup before using this outside a tight, well-understood loop.

```go
a = [1, 2, 3, 4, 5]
chunks = a.chunk_view(2)
chunks[0][0] = 99
a                        // [99, 2, 3, 4, 5] - the source changed too
chunks[0].is_view()      // true
```

#### `append(...)`

Returns a new array with the given items added.

**Arguments:**

- `...items` (any, 0 or more): Values to append.

**Returns:** `array`

**Description:** Always returns a fresh, independently-owned array — never touches the receiver's backing
storage, works regardless of the receiver's mutability, even with zero items (a legal no-op that still returns
an independent copy rather than the receiver itself). If an item is itself an `array`, it's appended as a single
element, not spread — pass `x.append(other...)` to spread. For amortized O(n) growth in a loop, use
`append_in_place()` instead; see [container semantics](container-semantics.md#append).

```go
x = [1, 2, 3]
v1 = x.append(100)
v2 = x.append(200)
x, v1, v2    // [1, 2, 3] [1, 2, 3, 100] [1, 2, 3, 200] - independent, no aliasing
```

#### `append_in_place(...)`

Appends items to the array in place.

**Arguments:**

- `...items` (any, 0 or more): Values to append.

**Returns:** `array` (the receiver)

**Description:** Mutates the receiver's own shared body directly and returns that same receiver — not a copy.
Every existing alias into the receiver sees the mutation, deterministically, with no reassignment needed.
Rejects an immutable receiver. Zero items is a legal no-op. See
[container semantics](container-semantics.md#append-in-place-aliasing) for the full aliasing contract, including
the one place the outcome is still implementation-defined (two independent containers that once shared a common
ancestor, e.g. two views derived from the same source).

```go
x = [1, 2, 3]
b = x                        // b shares x's body
x.append_in_place(4)
x, b                          // [1, 2, 3, 4] [1, 2, 3, 4] - b sees it too
immutable([1]).append_in_place(2)   // Error: not_appendable
```

#### `splice(start[, delete_count[, ...items]])`

Returns a new array with a range removed and/or items inserted.

**Arguments:**

- `start` (int): Start index. Must be within `[0, len]`.
- `delete_count` (int, optional): Number of elements to remove starting at `start`. Defaults to "everything from
  `start` to the end." Must be non-negative; clamped if it would run past the end.
- `...items` (any, 0 or more): Values to insert at `start`, after the deletion.

**Returns:** `array` (the array after the operation — not the deleted items)

**Description:** Always builds a genuinely fresh array — never aliases the receiver — and works regardless of
the receiver's mutability. For the mutating twin that returns the deleted items instead, see
`splice_in_place()`.

```go
a = [1, 2, 3, 4, 5]
a.splice(1)              // [1] - deletes from index 1 to the end
a.splice(1, 2)            // [1, 4, 5] - deletes 2 elements starting at 1
a.splice(1, 2, "x", "y")  // [1, "x", "y", 4, 5] - deletes 2, inserts 2
a.splice(1, 0, "z")       // [1, "z", 2, 3, 4, 5] - deletes 0 = pure insertion
a                          // [1, 2, 3, 4, 5] - source untouched
```

#### `splice_in_place(start[, delete_count[, ...items]])`

Removes a range and/or inserts items into the array in place.

**Arguments:** Same as `splice()`.

**Returns:** `array` of the deleted elements (not the modified array)

**Description:** Mutates the receiver's own shared body directly — visible through every existing alias without
reassignment. Rejects an immutable receiver. Note the return value is the *opposite* of `splice()`'s: this
returns what was removed, so you can inspect it, while the pure form returns the array after the change.

```go
a = [1, 2, 3, 4, 5]
deleted = a.splice_in_place(1, 2)
deleted    // [2, 3]
a          // [1, 4, 5]
immutable([1]).splice_in_place(0)   // Error: invalid_argument_type (expects a mutable array)
```

#### `repeat(n)`

Repeats the array `n` times by concatenation.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `array`

**Description:** Returns a new array containing the original array's elements concatenated `n` times. Element references
are not deep-copied — reference-type elements are shared, exactly as they would be in an array literal. Returns an empty
array when `n == 0`. Errors when `n < 0`.

```go
[1, 2].repeat(3)               // [1, 2, 1, 2, 1, 2]
[].repeat(5)                   // []
[1, 2, 3].repeat(0)            // []
```

#### `join(sep)` / `join()`

Stringifies each element and joins them with a separator.

**Arguments:**

- `sep` (string | runes | byte | rune, optional): Separator. Defaults to `""` (empty string).

**Returns:** Type matches `sep`: `string` for `string`/no arg, `runes` for `runes`/`rune`, `bytes` for `byte`.

**Description:** Each element is stringified using its `AsString` conversion (the same coercion used by the `+`
operator). An empty array produces an empty result. A single-element array produces just the stringified element with
no separator. The separator type controls the result type. Values that cannot be converted to a string (e.g.
`undefined`, functions) raise a runtime error.

```go
[1, 2, 3].join(", ")           // "1, 2, 3"
[1, "a", true].join(" | ")     // "1 | a | true"
[1, 2, 3].join()               // "123"
[].join(", ")                  // ""
[42].join(", ")                // "42"
[1, 2, 3].join(',')            // runes "1,2,3"
[1, 2, 3].join(byte(0x2C))     // bytes "1,2,3"
```

#### `flatten([depth])`

Flattens nested arrays into a new array.

**Arguments:**

- `depth` (int, optional): Maximum levels of nesting to unwrap. Defaults to `1` (one level). `0` returns a shallow copy
  with no unwrapping. Negative values mean unbounded (fully recursive).

**Returns:** `array`

**Description:** Returns a new top-level array. Only `array` elements are unwrapped — strings, runes, bytes, ranges,
records, dicts, and scalars are kept as-is. Element references are not deep-copied: reference-type elements are shared
with the original (same convention as `repeat` and array literals).

```go
[[1, 2], [3, 4]].flatten()              // [1, 2, 3, 4]
[1, [2, 3], [4, [5, 6]]].flatten()      // [1, 2, 3, 4, [5, 6]]
[1, [2, 3], [4, [5, 6]]].flatten(2)     // [1, 2, 3, 4, 5, 6]
[1, [[2, [[3]]]]].flatten(-1)           // [1, 2, 3]
[1, [2, [3]]].flatten(0)                // [1, [2, [3]]]   (shallow copy)
["ab", [1, 2]].flatten()                // ["ab", 1, 2]    (strings stay intact)
```

#### `filter(fn)` / `filter()`

Filters by predicate, or filters out `undefined` values when called without arguments.

**Arguments:**

- `fn` (function, optional): Predicate function. Accepts one argument (value) or two (index, value). When omitted, all
  `undefined` elements are removed.

**Returns:** `array`

**Description:** Returns a new array with only elements where the predicate returns `true`. If called with no arguments,
returns a new array with all `undefined` elements removed.

```go
[1, 2, 3, 4, 5].filter(x => x % 2 == 0)        // [2, 4]
[10, 20, 30].filter((i, v) => i > 0)           // [20, 30]
[1, undefined, 2, undefined, 3].filter()       // [1, 2, 3]
```

#### `map(fn)`

Transforms elements.

**Arguments:**

- `fn` (function): Transformation function. Accepts one argument (value) or two (index, value).

**Returns:** `array`

**Description:** Returns a new array with each element transformed by the function.

```go
[1, 2, 3].map(x => x * 2)                      // [2, 4, 6]
[1, 2, 3].map((i, v) => i * v)                 // [0, 2, 6]
```

#### `for_each(fn)`

Executes a callback for each element.

**Arguments:**

- `fn` (function): Callback function. Accepts one argument (value) or two (index, value).

**Returns:** `undefined`

**Description:** Calls `fn` for each element and ignores callback results except for control flow. Iteration stops when
`fn` returns falsy value.

```go
sum = 0
[1, 2, 3].for_each(v => {
    sum += v
    return true
})
```

### Predicate Functions

#### `all(fn)`

Tests if all elements match predicate.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `bool`

**Description:** Returns `true` if all elements satisfy the predicate.

```go
[2, 4, 6].all(x => x % 2 == 0)     // true
[1, 2, 3].all(x => x % 2 == 0)     // false
```

#### `any(fn)`

Tests if any element matches predicate.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `bool`

**Description:** Returns `true` if any element satisfies the predicate.

```go
[1, 3, 5].any(x => x % 2 == 0)     // false
[1, 2, 3].any(x => x % 2 == 0)     // true
```

#### `find(fn)`

Finds index of first matching element.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument (value) or two (index, value).

**Returns:** `int` or `undefined`

**Description:** Returns the index of the first element for which the predicate returns `true`. Iteration stops on the
first match. Returns `undefined` if no element matches.

```go
[10, 20, 30].find(x => x == 20)      // 1
[10, 20, 30].find(x => x == 99)      // undefined
[10, 20, 30].find((i, v) => i == 2)  // 2
```

#### `contains(x)`

Checks if array contains value.

**Arguments:**

- `x` (any): Value to search for

**Returns:** `bool`

**Description:** Returns `true` if the exact value is found.

```go
[1, 2, 3].contains(2)      // true
[1, 2, 3].contains(4)      // false
```

### Aggregation Functions

#### `count(fn)` / `count()`

Counts elements matching predicate or counts non-`undefined` elements when called without arguments.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `int`

**Description:** Returns the number of elements where the predicate returns `true`. If called with no arguments, returns
the number of non-`undefined` elements.

```go
[1, 2, 3, 4, 5].count(x => x > 2)    // 3
[1, 2, 3].count(x => x % 2 == 0)     // 1
```

#### `reduce(init, fn)`

Reduces array to single value.

**Arguments:**

- `init` (any): Initial accumulator value
- `fn` (function): Reducer function. Accepts two arguments (accumulator, value).

**Returns:** `any`

**Description:** Iteratively applies the reducer function to produce a single value.

```go
[1, 2, 3].reduce(0, (acc, v) => acc + v)         // 6
[1, 2, 3].reduce(1, (acc, v) => acc * v)         // 6
["a", "b", "c"].reduce("", (acc, v) => acc + v)  // "abc"
```

#### `min()`

Finds minimum element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the smallest element. Returns `undefined` for empty array. Elements must be comparable.

```go
[3, 1, 4, 1, 5].min()    // 1
["c", "a", "b"].min()    // "a"
[].min()                 // undefined
```

#### `max()`

Finds maximum element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the largest element. Returns `undefined` for empty array. Elements must be comparable.

```go
[3, 1, 4, 1, 5].max()    // 5
["c", "a", "b"].max()    // "c"
[].max()                 // undefined
```

#### `sum()`

Sums numeric elements.

**Arguments:** None

**Returns:** `number`

**Description:** Returns the sum of all numeric elements (int, float, decimal).

```go
[1, 2, 3, 4, 5].sum()          // 15
[1.5, 2.5, 3.0].sum()          // 7.0
[decimal(1), decimal(2)].sum() // decimal(3)
```

#### `avg()`

Calculates average of numeric elements.

**Arguments:** None

**Returns:** `number | undefined`

**Description:** Returns the arithmetic mean. Returns `undefined` for empty array.

```go
[1, 2, 3, 4, 5].avg()    // 3
[10, 20, 30].avg()       // 20
[].avg()                 // undefined
```

### Query and Accessor Functions

#### `is_empty()`

Checks if array is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the array has no elements.

```go
[].is_empty()      // true
[1, 2, 3].is_empty()  // false
```

#### `len()`

Gets array length.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of elements.

```go
[1, 2, 3].len()    // 3
[].len()           // 0
```

#### `first()`

Gets first element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the first element. Returns `undefined` for empty array.

```go
[1, 2, 3].first()  // 1
[].first()         // undefined
```

#### `last()`

Gets last element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the last element. Returns `undefined` for empty array.

```go
[1, 2, 3].last()   // 3
[].last()          // undefined
```

## Examples

### Data Transformation

```go
fmt = import("fmt")

// Parse and transform data
scores = [85, 92, 78, 95, 88]

// Convert to percentages and filter
results = scores
    .map(s => (s.float() / 100.0) * 100.0)
    .filter(p => p >= 80.0)

fmt.println("Passing scores: ", results)
```

### Accumulation with Reduce

```go
fmt = import("fmt")

// Calculate total price with tax
items = [
    {name: "Item A", price: 10.0},
    {name: "Item B", price: 20.0},
    {name: "Item C", price: 15.0}
]

total = items.reduce(0.0, (sum, item) => sum + item.price)
tax = total * 0.08
fmt.println("Total: $" + total)
fmt.println("Tax: $" + tax)
```

### Complex Filtering

```go
fmt = import("fmt")

// Multi-condition filtering
users = [
    {name: "Alice", age: 25, active: true},
    {name: "Bob", age: 17, active: true},
    {name: "Carol", age: 30, active: false},
    {name: "Dave", age: 28, active: true}
]

active_adults = users
    .filter(u => u.active)
    .filter(u => u.age >= 18)

fmt.println("Active adults: ", active_adults)
```

### Array Statistics

```go
fmt = import("fmt")

// Calculate statistics
data = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]

count = data.len()
minimum = data.min()
maximum = data.max()
average = data.avg()

fmt.println("Count:", count)
fmt.println("Min:", minimum)
fmt.println("Max:", maximum)
fmt.println("Avg:", average)
```

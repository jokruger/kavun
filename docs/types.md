# Type Reference

Kavun has a comprehensive type system with scalar types (numbers, strings, runes), collections (arrays, records, dicts),
and specialized types (errors, time, ranges) plus shared container semantics.

For detailed documentation on each type including all member functions, arguments, descriptions, and examples, see the
individual type guides below. **This page is the map, not the territory**: it exists to build the mental model —
what types exist, which *families* they belong to, and what capability a family membership implies — not to
duplicate what's already fully documented per type.

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
slicing are byte-level; use `runes` for Unicode indexing. Raw strings supported via `r"..."` syntax. A thinner member
of the Sequence-shaped family (see below) than `bytes`/`runes` — no `append`/`splice`/mutating twins, since `string`
never has a mutating operation at all.

### [runes](types/runes.md)

Unicode strings indexed by rune, not byte. Use `u"..."` syntax for Unicode literals. Ideal for Unicode-first operations
where rune indexing is required throughout. Full member of the Sequence-shaped family.

### [bytes](types/bytes.md)

Byte sequences. Each element is a `byte` value (0-255). Useful for binary data manipulation. Use `b"..."` for bytes
literals. Full member of the Sequence-shaped family.

## Collection Types

### [array](types/array.md)

Ordered collections of heterogeneous values, shared by default on assignment/passing (see
[the value model](language.md#builtin-types-overview)). Full member of the Sequence-shaped family — supports
filtering, mapping, reduction, and aggregation operations, plus the full `append`/`splice`/`sort`/`reverse` twin
set.

### [record](types/record.md)

Primary object type with string keys and heterogeneous values. Supports both dot notation (`r.field`) and index
notation (`r["field"]`). The one significant outlier in the whole type system: **zero member functions** — every
operation on it (`copy`, `copy_shallow`, `delete`, `delete_in_place`, `freeze`, `freeze_shallow`, `format`,
`dict`/`dict_view` conversion) goes through a free builtin instead, because `record`'s own selector syntax
(`r.foo`) is reserved for looking up a field, not dispatching a method.

### [dict](types/dict.md)

Dictionary/map type similar to record but only supports index access for elements (`d["key"]`). Selector notation
reserved for member functions. The Map-shaped family's one real member — rich query, filtering, and key-removal
operations.

### [range](types/range.md)

Lazy sequences of integers. Efficiently represents large sequences without memory allocation until materialized.
Shares a handful of Sequence-shaped operations (`contains`, `find`, `for_each`, `is_empty`, `len`, `join`) but not
the full set — see the family table below.

## Specialized Types

### [time](types/time.md)

Date and time values representing instants in time. Parse from ISO 8601 strings or Unix timestamps. Includes extensive
date/time field access and timezone handling.

### [error](types/error.md)

First-class error values carrying payloads. Errors don't interrupt execution unless explicitly `raise()`d; use
conditional checks with `is_error()`, or `raise()`/`recover()` for exception-style handling — see
[Errors and recovery](language.md#errors-and-recovery).

### [container semantics](types/container-semantics.md)

The detailed contract behind the family capabilities summarized below: reference behavior, immutability wrappers,
slice/chunk/append aliasing, and the `_in_place`/`_view`/`_shallow` twin conventions. Read this before relying on
any sharing/mutation behavior beyond what's spelled out per-method on a type's own page.

## Type-Shape Families

Kavun's member-function surface isn't ad hoc per type — it's organized into a small number of **shape families**.
Knowing a type's family tells you most of what it can do without reading its full page; the full page then tells
you the type-specific rest. This grouping is sourced directly from
[`types/function-matrix.md`](types/function-matrix.md), which audits it against the actual implementation
(`core/*.go`) rather than aspiration — treat that file as the ground truth if this summary and the implementation
ever disagree.

### Core (near-universal)

Every type gets these except `record` (which has no member functions at all — see above; reachable via the
matching free builtin instead):

| Function | What it does |
| --- | --- |
| `copy()` | Independent, deep copy. Detaches fully; the default, safe choice. |
| `copy_shallow()` | Independent top-level copy; nested values still shared with the source. A no-op synonym of `copy()` on types with no nested values (every scalar). |
| `freeze()` | `copy()` plus recursive deep-immutability. Always detaches first, so no live alias is ever affected. |
| `freeze_shallow()` | Marks *this* header immutable, no detach. Genuinely pure (like `copy_shallow()`) — always needs `x = x.freeze_shallow()` to stick, and never protects a pre-existing sibling binding into the same body. |
| `format([spec])` | Renders via the [Format Mini-Language](format-mini-language.md). |
| `repeat(n)` | Concatenates itself `n` times. Missing on `dict`/`range`/`error` — not implemented, no stated reason. |

### Sequence-shaped (`array`, `bytes`, `runes` — `string` is a thinner member)

The "ordered collection of elements" family. `array`/`bytes`/`runes` get the complete set below; `string` gets
only `all`/`any`/`contains`/`count`/`filter`/`find`/`for_each`/`is_empty`/`join`/`len`/`slice`/`split`/
`split_lines`/`partition`/`trim`/`lower`/`upper`/`reverse` — no mutating twins at all (strings never have a
mutating operation, full stop), and no `append`/`splice`/`map`/`reduce`/`sort`/`dedup`/`unique`/`chunk`/`min`/
`max`/`sum`/`avg`/`first`/`last`/`flatten` (`+`/`.join()` already cover concatenation for text). `range` shares
only `contains`/`find`/`for_each`/`is_empty`/`len`/`join`.

| Function | Shape |
| --- | --- |
| `append(...)` / `append_in_place(...)` | Add items. Pure copy vs. mutate-in-place twins. |
| `splice(start[, n[, ...items]])` / `splice_in_place(...)` | Remove a range and/or insert items. Pure returns the result; the `_in_place` twin returns what was removed. |
| `slice(i, j)` / `slice_view(i, j)` | Sub-range. Copy vs. shares-storage twins — same rule as `a[i:j]`/its `_view` twin. |
| `chunk(size)` / `chunk_view(size)` | Split into fixed-size pieces. Copy vs. shares-storage twins. |
| `is_view()` | Predicate: does this value share storage with something else (only ever `true` for a `_view` result)? |
| `sort()` / `sort_in_place()`, `reverse()` / `reverse_in_place()` | Reorder. Copy vs. mutate-in-place twins. |
| `dedup()`, `unique()` | Remove duplicates (adjacent-only vs. all, respectively). |
| `map(fn)`, `filter(fn)`, `reduce(init, fn)`, `for_each(fn)` | Transform/iterate. |
| `all(fn)`, `any(fn)`, `find(fn)`, `count(fn)`, `contains(x)` | Predicate/search. |
| `min()`, `max()`, `sum()`, `avg()` | Aggregation over comparable/numeric elements. |
| `first()`, `last()`, `len()`, `is_empty()`, `join(sep)` | Basic accessors. |
| `flatten([depth])` | `array`-only: unwrap nested arrays. |
| `split(sep)`, `split_lines()`, `partition(sep)`, `lower()`, `upper()`, `trim(cutset)` | `bytes`/`runes`/`string`-only: text-shaped operations. |

See [container semantics](types/container-semantics.md) for the full copy-vs-share contract every `_in_place`/
`_view`/`_shallow` twin follows, including the aliasing hazards of the sharing forms.

### Map-shaped (`dict` — `record` is the outlier)

| Function | What it does |
| --- | --- |
| `keys()`, `values()` | Enumerate. |
| `delete(key)` / `delete_in_place(key)` | Remove a key. Pure copy vs. mutate-in-place twins. |
| `record()` / `record_view()` | Convert to `record`. Independent shallow copy vs. shares storage (both top-level keys *and* nested values) with the source. |
| `all(fn)`, `any(fn)`, `filter(fn)`, `find(fn)`, `for_each(fn)`, `count(fn)` | Predicate/iterate — 1-arg callbacks receive the **key**, not the value (see [Lambda Callbacks](#lambda-callbacks) below). |
| `contains(x)`, `is_empty()`, `len()` | Basic accessors. |

`record` has none of the above as member calls — use the free `copy`/`copy_shallow`/`delete`/`delete_in_place`/
`freeze`/`freeze_shallow`/`dict`/`dict_view` builtins instead, e.g. `delete(some_record, "key")` rather than
`some_record.delete("key")`.

### Numeric-specific (classification, rounding, scale)

Only `decimal` gets the full precision-sensitive toolkit; this is deliberate (decimal is the exact-arithmetic
type), not a gap:

| Function | Where |
| --- | --- |
| `abs()` | `int`, `decimal` (not `float` — a known, tracked gap, see `TODO.md`) |
| `sign()` | `int`, `float`, `decimal` |
| `sqrt()`, `is_zero()`, `is_negative()`, `is_positive()`, `is_nan()`, `canonical()`, `error_details()`, `negate()`, `next_up()`/`next_down()`, `rescale(scale)`, `round_*(scale)`, `scale()`, `trunc(scale)` | `decimal` only |

### Type-specific singletons

`time` (`day`, `hour`, `minute`, `month`, `year`, `unix`, `zone_offset`, `format_date`, ...) and `error`
(`value`, `kind`, `is_runtime`, `is_fatal`) each have a family of one — their member functions aren't shared with
any other type. See their own pages for the full list.

## Type Overview Quick Reference

| Type    | Shape family     | Indexed By | Primary Use           |
| ------- | ---------------- | ---------- | ---------------------- |
| int     | Core, Numeric    | N/A        | Whole numbers          |
| byte    | Core             | N/A        | Binary data            |
| rune    | Core             | N/A        | Unicode code points    |
| float   | Core, Numeric    | N/A        | Approximate decimals   |
| decimal | Core, Numeric    | N/A        | Exact decimals         |
| string  | Core, Sequence (thin) | Bytes | Text, UTF-8 encoded    |
| runes   | Core, Sequence   | Runes      | Text, rune indexed     |
| bytes   | Core, Sequence   | Bytes      | Binary data            |
| array   | Core, Sequence   | Integers   | Ordered collections    |
| record  | (none — outlier) | Strings    | Object representation  |
| dict    | Core, Map        | Strings    | Dictionary operations  |
| range   | Sequence (thin)  | Integers   | Integer sequences      |
| time    | Core             | N/A        | Date/time values       |
| error   | Core             | N/A        | Error handling         |

## Conversion Functions

Most types support a same-named top-level constructor/conversion function: `int()`, `float()`, `string()`, `array()`,
`bool()`, `byte()`, `rune()`, `decimal()`, `time()`, `runes()`, `bytes()`, `dict()`, etc. — plus a same-named member
function for converting *to* that type from another value (e.g. `[1,2,3].string()`). Each type's own documentation
details what converts into it via its member functions; see
[Built-in functions](language.md#built-in-functions) in the Language Reference for the complete, corner-case-by-
corner-case reference of the top-level constructors themselves (zero-value/passthrough/fallback rules,
`array(n)`/`bytes(n)`/`runes(n)` preallocation, and the `decimal`/`dict`/`error`/`range` outliers). See
[`types/function-matrix.md`](types/function-matrix.md#table-4--member-call-type-conversion-matrix) for the full
source-type × target-type reachability matrix.

## Lambda Callbacks

Most collection operations (`map`, `filter`, `find`, `count`, `all`, `any`, `for_each`, …) accept callbacks that take
either one or two arguments. The binding rule is consistent across all such operations:

- **Single-argument callbacks receive the type's primary item:**
  - the **value** for `array`, `bytes`, `runes`, `string`, and `range`
  - the **key** for `dict`
- **Two-argument callbacks always receive `(locator, value)`**, where the locator is the index for sequences and the
  key for dicts. The value is always the last argument.

```go
[1, 2, 3, 4].filter(x => x % 2 == 0)              // [2, 4]              -- 1-arg: value
[1, 2, 3].map((i, v) => i * v)                    // [0, 2, 6]           -- 2-arg: (index, value)
[1, 2, 3].reduce(0, (acc, v) => acc + v)          // 6

dict({a: 1, b: 2, c: 3}).filter(k => k != "b")    // 1-arg on dict: key
dict({a: 1, b: 2, c: 3}).filter((k, v) => v > 1)  // 2-arg on dict: (key, value)
```

This asymmetry is intentional: for sequences the value is the data being processed (the index is positional metadata),
while for dicts the key is the identity of an entry (and the value can always be looked up via the key).

## The Value Model: Sharing, Mutation, and Immutability

This page deliberately doesn't restate the sharing/mutation model — it's a single unified rule with one
corollary, not a per-type or per-family concern, and duplicating it here risks the two descriptions drifting
apart. Read it once in [Builtin types overview](language.md#builtin-types-overview): every value is shared, not
copied, on assignment/argument-passing; scalars simply have no mutating operation, so the sharing is never
observable for them. The developer-facing implementation detail (which types are `Ptr`-backed vs. `Data`-inline,
and why that's a *different* axis from "scalar vs. container") is in
[Conventions: Value model](conventions.md#value-model-scalars-vs-containers).

What *is* worth stating here: which specific operations are the mutating ones. Every mutating operation in the
language is named with an explicit twin suffix — `_in_place` (mutates the shared body, e.g. `append_in_place`),
or is a plain indexed assignment (`a[i] = x`, `d["k"] = x`). Nothing else mutates. `_shallow`/`_view` twins
(`copy_shallow`, `slice_view`) are about *depth*/*ownership*, not mutation — see
[container semantics](types/container-semantics.md) for how the three axes (`_in_place`, `_view`, `_shallow`)
combine.

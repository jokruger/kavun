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

## Operators across types

Arithmetic (`+ - * / %`), bitwise (`& | ^ &^ << >>`), ordering (`< > <= >=`), and unary (`- ! ^`)
operators resolve per pair of operand types, not just per single type — this section is the map of
which pairs are defined and what they produce; each type's own page has the full worked examples.
`==`/`!=` dispatch through a separate mechanism from the other five operator groups (see "Equality
across types" below), but their cross-type *outcomes* are deliberately coordinated with ordering's,
not independent of it.

- **Numeric family (`int`, `float`, `decimal`), arithmetic:** same-type arithmetic works as expected.
  Across types, only lossless widening is defined: `int` widens into `float` or `decimal` on either
  side (`1 + 2.5` → `float`, `1 + decimal(2)` → `decimal`) — but `float` and `decimal` **do not** mix
  with each other for arithmetic (`0.1 + 2.5d` is an error, not a silently-computed answer) since
  neither representation is a clear winner over the other. `bool` deliberately has **no** arithmetic
  at all, even with itself (`true + 1`, `true + true` are both errors) — deferred scope, not designed
  yet.
- **Numeric family, ordering — a deliberately wider allowlist than arithmetic's:** `bool`, `byte`,
  `rune`, `int`, `decimal`, and `float` all order against each other now, including `float`/`decimal`
  (which still can't be added or multiplied together — only ordering crosses that boundary).
  `bool`/`byte`/`rune` widen to `int`/`decimal`/`float` via the same 0/1 or code-point value used
  elsewhere; against `float` specifically they're always exact (their entire ranges sit inside
  `float64`'s exact-integer mantissa). `int`/`decimal` vs `float` is the one pairing that needs real
  care: ordering compares the two operands' *exact* mathematical values via `math/big.Rat` — never
  the lossy `float64(x)` conversion arithmetic uses when the result type is `float` (that's fine for
  arithmetic, where the answer is a `float` anyway; it's not fine for a yes/no ordering question).
  `decimal`'s coefficient/scale and any Go integer are already exact; `float64` converts to its exact
  rational value with no rounding at all (`(*big.Rat).SetFloat64`). This is what keeps
  `9007199254740993 <= float(9007199254740992)` correctly `false` instead of silently collapsing two
  adjacent large integers onto the same rounded `float64`. `NaN`/`±Inf`: any real number is less than
  `+Inf` and greater than `-Inf`; nothing is ordered against `NaN` from the exact side (an `int` or
  `decimal` compared to a `NaN` `float` is `false` for every one of `< > <= >=`) — except `decimal`'s
  own `NaN` state, which carries the same total-order placement as `float`'s (see next bullet): the
  unique minimum, sorting below even `-Inf`, equal only to another `NaN`.
- **`float`'s own `NaN` is a total order now, not IEEE-754 unordered:** `NaN == NaN` is `true`, and
  `NaN` sorts as the unique minimum — below even `-Inf` — for `< > <= >=` too, matching `decimal`'s
  pre-existing `NaN` convention exactly (`decimal`'s `NaN` was always a total order; only `float`
  changed here). This is a deliberate departure from `float64`'s native comparison operators, made
  because `array.sort()` depends on `<`/`==` forming a valid order to behave deterministically —
  IEEE-754 unordered semantics made sorting a `float` array containing `NaN` silently
  non-deterministic, since `NaN` compared false in both directions at once. `±Inf` needed no change:
  it was already a well-ordered, definite value under IEEE-754 (`Inf == Inf`, `5 < Inf`, etc. already
  worked correctly) — only `NaN`'s reflexivity/ordering changed.
- **`byte` and `rune` are not numeric types**, despite looking like small integers — see
  [byte](types/byte.md)/[rune](types/rune.md) for the reasoning. `byte` is a full mod-256 ring: every
  same-type or `int`-mixed `+`/`-` wraps (`byte(255) + 1 → byte(0)`, symmetric either operand order),
  and unary `-byte` is the ring's additive inverse. `rune` is a position/symbol type, not a ring:
  `rune ± int → rune` (offset), but `rune - rune → int` (a genuine distance between two code points,
  not same-type subtraction) — and `rune + rune`, unary `-rune`, and all `rune` bitwise are errors
  (none have a meaningful reading). `byte` and `rune` combine directly with each other too (a `byte`
  safely widens to its equivalent Latin-1 code point), producing whichever of the two behaviors
  above `rune op rune` would give.
- **Bitwise (`& | ^ &^`) is same-type only**, for `int` and `byte` only — no cross-width mixing.
  Shift operators (`<< >>`) are the one exception: the right-hand count may always be plain `int`
  regardless of the shifted value's own type (`byte(1) << 4` is valid), matching the universal
  shift-count convention in mainstream languages.
- **Sequence/text (`string`, `bytes`, `runes`) have a fixed rank**, `bytes > runes > string` — when
  two different sequence types combine with `+` or ordering, the result is always the
  higher-ranked type, regardless of which side it's written on (`"a" + bytes("b")` and
  `bytes("b") + "a"` both produce `bytes`). A `byte`/`rune` scalar joining any of the three always
  produces that sequence type, never the scalar (`b'A' + bytes("bc") → bytes`) — see
  [container semantics](types/container-semantics.md) and each type's own page for exactly which
  scalar pairs with which sequence. `-` means "remove all occurrences from the lhs" and has no
  either-order form — see each type's own page for its removal table. There is **no** implicit
  stringification of unrelated types: `"a" + 5`, `"a" + true`, `"a" + array(...)` are all errors,
  not silent string concatenation — use `.string()`, `f"..."`, or `print()` to format explicitly.
- **Collections (`array`, `dict`, `record`) get exactly the pairings listed, nothing implicit:**
  `array + array` concatenates (`array`'s only operator — no scalar append/prepend, no `-` at all,
  since an array element can be any type including another array, making "append one element" vs.
  "concatenate" genuinely ambiguous by operand type alone). `dict + dict`, `record + record`,
  `record + dict`, `dict + record` all merge (rhs wins key collisions); the result is `dict` the
  moment either side is `dict`, `record` only when both sides are. `dict - "key"` removes that key
  (non-mutating). `int_range` has no operators yet (deferred, tracked in `TODO.md`).
- **`time`:** `time + int`/`time - int` adds/subtracts nanoseconds, `time - time` gives the
  nanosecond duration between them (`int`), and same-type ordering works — no arithmetic with any
  other type. In operator position an `int` is always a **duration in nanoseconds**; the *timestamp*
  reading of an `int` belongs to conversion position only (`time(n)`, `n.time()`, `t.unix()`, …).
  There is deliberately no `time` vs `int` ordering or equality, since it would have to pick one of
  the two roles — see [time](types/time.md#what-an-int-means-next-to-a-time) for the rule and the
  explicit conversions to use instead.
- **`undefined` propagates through everything:** any arithmetic/bitwise/ordering operator touching
  `undefined`, on either side, produces `undefined` — "unknown contaminates everything it touches."
  `undefined == undefined → true`; `undefined` compared against anything else is always `false`
  (`!=` always `true`), regardless of operand order.
- **`error` rejects everything except `undefined`:** any arithmetic/bitwise/ordering operator on an
  `error`, on either side, is an error — except `error op undefined`, which yields `undefined`
  (unknown-ness always wins over a prior failure). `error == error` compares payloads; `error`
  compared against any other concrete type is always `false` (`!=` always `true`).
- **Unary `!`** converts any value to `bool` via truthiness and negates — `!undefined → true`
  (`undefined` is falsy), `!error → false` (every `error` is truthy, supporting the common
  "`undefined` on success, `error` on failure" `if x`/`if !x` idiom). Unary `-`/`^` are only defined
  for the specific types listed above; every other type errors on unary `-`/`^`, `undefined`
  propagates, and `error` errors (matching its binary behavior).

## Equality across types

`==`/`!=` dispatch through a separate mechanism from `BinaryOp` (see
[Extending types: operators](extending-types.md) for the implementor-facing contract), but their
cross-type outcomes are deliberately built to agree with ordering's wherever both are defined, not
independent of it. Two properties hold everywhere, no exceptions: **`==`/`!=` never error** — any
two values of any types can be compared, unrelated types simply compare `false` (`array() == 5` is
`false`, not an error) — and **equality is always commutative**, `a == b` and `b == a` never
disagree, for every pairing below.

- **The exact chain — `bool` < `byte` < `rune` < `int` < `decimal`:** each level losslessly embeds
  every value of the level below it (`bool`→0/1, `byte`→`rune` via the Latin-1 bijection, `rune`→`int`
  via code-point value, `int`→`decimal` via widening), and every type recognizes every type below it
  directly — flattened, not chained (`int == true` doesn't hop through `byte`/`rune` one step at a
  time). `true == 1`, `byte(1) == rune(1)`, `rune(65) == 65`, `5 == decimal("5")` are all `true`.
- **`float` is a conditionally-exact relationship, not a fixed rung** — `bool`/`byte`/`rune` are
  *always* exact against `float` (their entire ranges sit inside `float64`'s exact-integer mantissa),
  but `int`/`decimal` are only exact for the *specific value*: `9007199254740993 ==
  float(9007199254740993)` is `false` (the float rounds to `9007199254740992.0`), while
  `9007199254740992 == float(9007199254740992)` is `true`. Likewise `decimal("0.1") == 0.1` is
  `false` — the literal float `0.1` is actually `0.1000000000000000055511151231257827021181583404541015625`,
  not exactly a tenth — while `decimal("0.5") == 0.5` is `true` (0.5 has an exact binary form). This
  is computed by comparing the two operands' true exact mathematical values (via `math/big.Rat`),
  never by rounding one side into the other's representation first — rounding either direction would
  produce exactly these kinds of false positives.
- **`NaN`/`Inf`:** `decimal`'s `NaN` state and a `NaN` `float` are the same "unique minimum" concept
  from both directions — `decimal("NaN")` compared against a `NaN` `float` is `true`, and `NaN`
  compared to any ordinary number is `false`. `float`'s own same-type `NaN == NaN` is `true` too now
  (see "Operators across types" above) — reflexive, unlike raw IEEE-754.
- **The text tier — `string`, `runes`, `bytes`:** every member of the exact chain plus `float`
  converts to its own canonical text form (digit text, `"true"`/`"false"`, or `float`'s
  shortest-round-trip string) and compares as text against all three — `5 == "5"`, `true ==
  "true"`, `decimal("2.5") == "2.5"` are all `true`. Among the three text types themselves: `string`
  vs `runes` compares by Unicode code point (`runes` outranks `string` in the existing sequence-family
  rank); anything involving `bytes` compares as raw bytes instead (`bytes` outranks both) — encoding
  the other side down via its own already-exact conversion, never decoding arbitrary `bytes` up into
  text, since `bytes` may not hold valid UTF-8 at all. For well-formed UTF-8 these two domains agree
  exactly (a deliberate property of UTF-8's design: byte-lexicographic order equals code-point order),
  so `bytes` joining the tier doesn't introduce any inconsistency — it's a strict generalization.
- **Numeric-vs-text ordering stays undefined** even though numeric-vs-text *equality* is deliberate:
  `byte(1) < "1"` is still meaningless and errors — lexicographic and numeric order are different
  things, and this document doesn't invent a resolution just because equality has one.
- **Untouched by any of this:** `time`, `array`, `dict`, `record`, `error`, `undefined`, iterators,
  compiled functions, and `format_spec` all stay same-type-only for `==`/`!=`, exactly as described
  under "Operators across types" above for the operators they do support (`dict`/`record`'s existing
  mutual equality is a separate, already-decided relationship, not part of this cross-type model).

For the exact reasoning behind why any specific pairing is or isn't defined — including several
that look plausible but were deliberately rejected (`array + scalar`, `float`/`decimal` mixing,
implicit stringification) — see the relevant type's own page. Implementors adding a new builtin or
embedder type should read [Extending types: operators](extending-types.md) instead — this section
is the result, that document is the mechanism.

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

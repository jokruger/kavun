# range

Lazy integer sequence defined by `start`, `stop`, and `step`.

## Overview

A `range` is an arithmetic progression of `int` values that is never stored: every element is computed from
three components — `start` (inclusive), `stop` (exclusive), and `step` (always positive). Direction comes from
the relation of `start` to `stop`, not from the sign of `step`:

```go
range(2, 8)          // 2, 3, 4, 5, 6, 7        (ascending, step 1)
range(1, 10, 2)      // 1, 3, 5, 7, 9           (ascending, step 2)
range(10, 0, 3)      // 10, 7, 4, 1             (descending — start > stop)
range(3, 3)          // empty
```

`range` is always immutable, its elements are always `int`, and `len()`, membership, aggregation, and every
transform are O(1) closed forms computed on the three components — nothing is ever materialized unless you ask
for it with `.array()` (or another conversion).

That laziness fixes the member roster: **no member of `range` answers a new materialized sequence of its own
elements.** Every transform that exists (`slice`, `reverse`, `sort`, `dedup`, `unique`) is a closed form on
`start`/`stop`/`step` and answers another `range`; `chunk(n)` answers an array *of* ranges. Anything that could
produce arbitrary elements (`map`, `keep`, `append`, …) is deliberately absent — spell it `.array().map(...)`.
See [Excluded members](#excluded-members).

## Construction

### Constructor forms

```go
range()              // the empty range — range(0, 0)
range(2, 8)          // start, stop; step defaults to 1
range(1, 10, 2)      // start, stop, step
```

- There is **no one-argument form**: `range(5)` is a runtime error (`expected 0, 2 or 3 argument(s)`).
  Write `range(0, 5)`.
- `step` must be greater than zero; `range(1, 10, 0)` and `range(1, 10, -2)` raise `invalid_value`
  (`range step must be greater than 0`). Descending sequences come from `start > stop`, never from a
  negative step.

### The `..` literal

`a..b` is the literal form of `range(a, b)` (step 1, either direction):

```go
1..4                 // range(1, 4) — 1, 2, 3
5..1                 // range(5, 1) — 5, 4, 3, 2
(1..4).array()       // [1, 2, 3] — parenthesize before calling a member
```

There is no literal spelling for a step other than 1; use the constructor.

### From a components record

`range(rec)` rebuilds a range from a record with keys `start`, `stop`, and optionally `step`; an unknown key
raises. `components()` is the reverse direction. A `dict` with the same keys is accepted too.

```go
range({start: 1, stop: 10, step: 2})        // range(1, 10, 2)
range({start: 1, stop: 5})                  // range(1, 5) — step defaults to 1
range({start: 1, stop: 5, bogus: 1})        // runtime error: unknown component "bogus"

r := range(1, 10, 2)
format(r.components())                      // "{\"start\": 1, \"step\": 2, \"stop\": 10}"
range(r.components()) == r                  // true — the round trip
```

`components()` answers a `record`, which has no member surface — render it with the free `format(x)` and read
fields with `rec.start` etc.

## Iteration and indexing

`for x in r` yields the elements lazily, in the range's direction:

```go
acc := []
for x in range(10, 0, 3) {
    acc = acc + x
}
// acc == [10, 7, 4, 1]
```

Single elements are O(1)-indexable, with negative indices counting from the end:

```go
r := range(1, 10)
r[2]                 // 3
r[-1]                // 9
r[99]                // runtime error (index_out_of_bounds)
```

Slice *syntax* is not supported — `r[1:3]` raises `not_sliceable`. The member `slice(i, j)` is the spelling,
and unlike the syntax form it clamps instead of raising.

## Operators

### Equality

Ranges compare by their **definition** — the `start`/`stop`/`step` components — not by the element sequence
they generate:

```go
range(1, 4) == range(1, 4)          // true
range(1, 4) == range(1, 4, 1)       // true  — step 1 is the default, same definition
range(1, 4) != range(1, 5)          // true
range(1, 11, 3) == range(1, 12, 3)  // false — both generate 1, 4, 7, 10, but the definitions differ
range(1, 4) == [1, 2, 3]            // false — never equal to another type
```

Ordering (`<`, `<=`, …) is not defined between ranges and raises.

### The operators a range does not have

`+`, `-` and `*` are **not defined** on a range and raise (`range + int`, `range - int`, `range * int`) — a
range is a read-only formula, so there is nothing to append to, remove from, or repeat. Nor does a range
spread into another sequence's operator: it is **one element**, like any other non-`array` value
(`[9] + (1..4)` → `[9, range(1, 4)]`). Materialize it first when you want the elements — `[9] + (1..4).array()`
→ `[9, 1, 2, 3]`. The members mirror this exactly: a range has `contains`, and no `append`/`remove`/`repeat`.

### Membership: `x in r`

An `int` operand is a closed-form membership test — arithmetic on the components, no materialization:

```go
3 in range(1, 10, 2)     // true
4 in range(1, 10, 2)     // false
```

A sequence operand (`array` or `range`) asks for the *run* reading, which is deferred on `range` and raises
saying so (`the run reading on a range is deferred until the vectorised integer sequence type exists; write
.array() explicitly`). A callable operand raises too — an operator operand is always a value; the predicate
reading is spelled `r.contains(f)` / `r.any(f)`:

```go
[2, 3] in range(1, 10)                    // runtime error — deferred (not_implemented)
(2..4) in range(1, 10)                    // runtime error — deferred
func(x) { return x > 2 } in range(1, 10)  // runtime error — write r.any(fn)
```

### A range as the other side's operand

On `array`'s side a `range` is **one element**, like every value that is not itself an `array` — a range is
never spread implicitly. Materializing is spelled at the call site, and then it is an ordinary array:

```go
[9] + range(1, 4)                  // [9, range(1, 4)]  — one element
[9] + range(1, 4).array()          // [9, 1, 2, 3]      — spelled
[1, 2, 3] - range(2, 4)            // [1, 2, 3]         — no element equals the range
[1, 2, 3] - range(2, 4).array()    // [1]               — removes the run 2, 3
```

The reason is forward-looking: a range that quietly became an `array` would answer an `array` today and an
`ints` tomorrow — silently, in a script that named neither. It is the same rule that keeps `map`/`keep` off
this type.

`range` itself has no add (or remove) operator — a range cannot be extended and stay a closed form:

```go
range(1, 4) + 5             // runtime error: range + int
range(1, 4) + range(4, 7)   // runtime error: range + range
```

`range + array` is the one exception, and it is the array's doing, not the range's: the range has no reading
for an array, so it hands the operation over and the array prepends it as one element.

```go
range(1, 4) + [1]           // [range(1, 4), 1]
```

On the text types (`string`/`runes`/`bytes`), a `range` argument or operand raises — they accept text content
only. Convert first: `bytes("ab") + range(1, 4).bytes()`.

## Member functions

Argument-taking search members (`contains`, `count`, `any`, `all`, `index`, `index_last`) share one reading
menu: an **`int` element**, a **variadic set** of ints (match = "∈ the set"), a **predicate function**, or
**no argument** — the blank-set reading, which for `range` is the set `{0}` ("significant" = non-zero). All
readings are closed forms. A sequence argument (`array` or `range`) would be the run reading, which is
**deferred** on `range` and raises telling you to write `.array()` explicitly — except on `any`/`all`, where a
run argument is refused **permanently** (there is no universal reading of "any equals this run"). Mixing a
function with other arguments in one variadic call raises.

### Universal

#### `format([spec])`

Renders the range using the [Format Mini-Language](../format-mini-language.md).

```go
range(1, 10, 2).format("v")   // "range(1, 10, 2)"
```

#### `copy()` / `freeze()`

Identity no-ops: a range is always immutable, so both answer the receiver itself.

```go
r := range(1, 4)
r.copy() == r        // true
r.freeze() == r      // true
is_immutable(r)      // true
```

#### `is_true()`

Truthiness: `true` iff the range is non-empty.

```go
range(1, 4).is_true()   // true
range().is_true()       // false
```

### Size

#### `len()` / `is_empty()`

Element count / emptiness, both O(1) closed forms.

```go
range(2, 8).len()       // 6
range(10, 0, 3).len()   // 4
range().is_empty()      // true
```

### Search and quantifiers

#### `contains(x)` / `contains(...set)` / `contains(fn)` / `contains()`

```go
range(1, 10, 2).contains(5)      // true
range(1, 10, 2).contains(4)      // false — not on the progression
range(1, 10, 2).contains(4, 5)   // true  — set reading: any of them
range(1, 10, 2).contains(func(x) { return x % 2 == 0 })   // false — no even element
range(0, 3).contains()           // true  — blank reading: some element is non-zero
range(1, 10).contains([2, 3])    // runtime error — run reading deferred; write .array()
```

`contains(fn)` ≡ `any(fn)` and `contains()` ≡ `any()`.

#### `count(x)` / `count(...set)` / `count(fn)` / `count()`

How many elements match. Same readings as `contains`.

```go
range(1, 10).count(3)                            // 1
range(1, 10).count(1, 3)                         // 2 — the set reading
range(1, 10).count(func(x) { return x > 6 })     // 3
range(0, 5).count()                              // 4 — non-zero elements (1, 2, 3, 4)
```

#### `any(x)` / `all(x)`

Value, set, predicate, or blank readings; a sequence argument raises **permanently** (not the deferred wording —
"any equals this run" has no universal meaning).

```go
range(1, 10).any(func(x) { return x > 8 })   // true
range(1, 10).all(func(x) { return x > 0 })   // true
range(1, 4).all()                            // true  — every element non-zero
range(0, 3).all()                            // false — contains 0
range(1, 10).all(2..4)                       // runtime error — no run reading on all/any
```

#### `index(x[, default])` / `index_last(x[, default])`

Position of the first/last match; the only locators. A miss answers `undefined`, or the trailing `default`
when given — never `-1`. Readings: element, variadic set, predicate, blank (first/last non-zero element).
A sequence argument raises as deferred.

```go
range(1, 4).index(2)                             // 1
range(1, 4).index(9)                             // undefined
range(1, 4).index(9, -1)                         // -1 — only because you asked for it
range(1, 4).index(func(x) { return x > 1 })      // 1
range(1, 4).index_last(func(x) { return x > 1 }) // 2
range(0, 3).index()                              // 1 — first non-zero element
```

### Element answers and aggregation

All six carry the uniform optional trailing `[default]`: an empty range has no answer to give, so without a
default they raise `invalid_value`.

#### `first([default])` / `last([default])` / `min([default])` / `max([default])`

```go
range(2, 8).first()      // 2
range(2, 8).last()       // 7 — stop is exclusive
range(10, 0, 3).min()    // 1
range(10, 0, 3).max()    // 10
range().first()          // Error: invalid_value: (first) empty sequence
range().first(99)        // 99
```

#### `sum([default])` / `avg([default])`

Closed-form aggregation; elements are `int`, so both answer `int`, and `avg` performs the same integer
division `array`'s does (quotient truncated toward zero).

```go
range(1, 4).sum()    // 6
range(1, 4).avg()    // 2
range(1, 3).avg()    // 1 — (1 + 2) / 2, integer division, same as [1, 2].avg()
range().sum()        // undefined
range().sum(0)       // 0
```

### Closed-form transforms

Each answers a **`range`** — computed on the components, nothing materialized. The result's printed
components may differ from what you would write by hand while generating exactly the intended elements.

#### `slice(i, j)`

Elements at positions `[i, j)`, keeping the receiver's direction. Positions clamp to the valid interval
(reading past the end is harmless); negative positions count from the end.

```go
range(1, 10).slice(2, 5)              // range(3, 6)  — elements 3, 4, 5
range(1, 10).slice(2, 100)            // range(3, 10) — clamps
range(10, 0, 3).slice(1, 3).array()   // [7, 4] — direction kept
```

#### `reverse()`

The same elements in the opposite direction.

```go
range(1, 4).reverse()           // range(3, 0) — 3, 2, 1
range(10, 0, 3).reverse()       // range(1, 11, 3) — 1, 4, 7, 10
```

#### `sort()`

Ascending order: the identity on an ascending range, `reverse()` on a descending one.

```go
range(1, 4).sort()               // range(1, 4)
range(10, 0, 3).sort().array()   // [1, 4, 7, 10]
```

#### `dedup()` / `unique()`

Both are the identity — an arithmetic progression never repeats an element; they exist so generic sequence
code need not special-case `range`.

```go
range(1, 4).dedup()    // range(1, 4)
range(1, 4).unique()   // range(1, 4)
```

### Partitioning

#### `chunk(n)`

Consecutive pieces of `n` elements (the last may be shorter), answering an **array of ranges** — the outer
array holds no `int` element, so nothing is materialized beyond the piece definitions. `n` must be positive.

```go
range(1, 10).chunk(4)            // [range(1, 5), range(5, 9), range(9, 10)]
range(1, 10).chunk(4)[0].array() // [1, 2, 3, 4]
range(1, 10).chunk(0)            // runtime error: chunk size must be positive
```

### Iteration members

#### `for_each(fn)`

Visits every element; the callback's return value is ignored (early exit is `break`'s job, inside a `for`
statement). Answers the **receiver**, so it chains — `r.for_each(fn) == r`. Callbacks bind as everywhere:
1-arg gets the element, 2-arg gets `(position, element)`.

```go
acc := []
range(5, 8).for_each(func(i, x) { acc = acc + [[i, x]] })
// acc == [[0, 5], [1, 6], [2, 7]]
```

#### `reduce(init, fn)`

Left fold over the elements; the callback gets `(accumulator, element)`.

```go
range(1, 5).reduce(0, func(acc, x) { return acc + x })   // 10
```

### Render

#### `join([sep])`

Renders every element and joins with the separator. The result type follows the separator's type; with no
separator the result is a `string`.

```go
range(1, 4).join(",")    // "1,2,3"
range(1, 4).join()       // "123"
range(1, 4).join('-')    // u"1-2-3" — a rune separator answers runes
```

### Conversions

#### `array()`

Materializes the elements, in order — the escape hatch to the whole `array` surface.

```go
range(1, 5).array()      // [1, 2, 3, 4]
range(10, 0, 3).array()  // [10, 7, 4, 1]
range(1, 4).array().map(func(x) { return x * x })   // [1, 4, 9]
```

#### `string()` / `runes()` / `bytes()`

Read the elements as text content: `string()`/`runes()` take each element as a Unicode code point,
`bytes()` takes each element as an octet (0–255). Keep the elements inside the target's domain.

```go
range(65, 68).string()   // "ABC"
range(65, 68).runes()    // u"ABC"
range(65, 68).bytes()    // bytes([65, 66, 67])
```

#### `components()`

The record `{start, stop, step}` — the exact definition, ready to feed back to `range(rec)`.

```go
format(range(1, 10, 2).components())   // "{\"start\": 1, \"step\": 2, \"stop\": 10}" — a map renders key-sorted
```

#### `range()`

The identity self-conversion — answers the same value.

```go
r := range(1, 4)
r.range() == r    // true
```

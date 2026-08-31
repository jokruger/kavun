# undefined

The absence of a value.

## Overview

`undefined` is what the language answers where there is nothing to answer: a missing dict key, a locator or
aggregation miss, a function that returns nothing. It is not an error and not a zero — it is *absence*, and it
has the thinnest member surface in the language: **`format()`, `is_true()`, and the conversion members with a
mandatory default** (`undefined.int(0)` → `0` — see [Absence and defaults](#absence-and-defaults)). Absence can
be rendered, tested, and materialized with an explicit fallback, but there is no value there to operate on —
which is why `copy` and `freeze` do not exist on it, as members or as free calls.

```go
undefined.format()        // "undefined"
undefined.is_true()       // false
!!undefined               // false
type_name(undefined)      // "undefined"
is_undefined(undefined)   // true
```

## Where it appears

```go
d := dict({a: 1})
d["missing"]         // undefined — a missed key
[1, 2].index(9)      // undefined — a locator miss (never -1)
[].first()           // undefined — aggregation on the empty sequence
range().sum()        // undefined

f := func() { }      // no return value
f()                  // undefined

g := func() out { }  // a named result starts as undefined
g()                  // undefined — until the body (or a deferred handler) assigns it
```

Everywhere absence can occur, the member also takes an optional trailing default that replaces the
`undefined` answer: `[1, 2].index(9, -1)` → `-1`, `[].first(0)` → `0`.

## Operators

One rule governs everything here: `undefined` **propagates on the data plane and raises on the action plane**.
Chaining-shaped *reads* — selectors, indexing, slicing, arithmetic, concatenation, ordering — flow through and
answer `undefined`, so a chain like `a.b.c` can miss at any level and still answer one `undefined` at the end,
with no null-check flood. *Actions* — calling, membership, iteration, member calls — raise: absence performs
nothing.

On the data plane, arithmetic, concatenation, and ordering answer `undefined` rather than raising — with
equality as the one exception that answers a real `bool`:

```go
undefined + 1              // undefined
undefined + "a"            // undefined
undefined - 1              // undefined
undefined < 1              // undefined — ordering has no answer against absence
undefined == undefined     // true  — the exception: equality is answerable
undefined != undefined     // false
undefined == 0             // false — absence equals nothing but itself
undefined && true          // undefined
undefined || 5             // 5 — the usual "fallback" spelling
```

The action plane refuses instead of flowing:

```go
1 in undefined          // runtime error (invalid_binary_operator) — membership in an
                        // absent container is a program error, not an absent answer
u := undefined
u()                     // runtime error (not_callable)
for x in undefined { }  // runtime error (not_iterable) — absence has no elements,
                        // not zero of them; is_iterable(undefined) is false
```

Indexing and slicing both propagate (`undefined[0]` → `undefined`, `undefined[0:1]` → `undefined`) — they are
chaining-shaped reads, so a chain of lookups can miss at any level and still answer one `undefined` at the
end. And since nothing is there to mutate, `is_immutable(undefined)` answers `true`.

## Absence and defaults

`undefined` converts to nothing on its own: `int(undefined)` raises (`cannot convert undefined to int: value
is missing`). But `undefined` carries **every conversion member** — `bool`, `byte`, `rune`, `int`, `float`,
`decimal`, `time`, `string`, `runes`, `bytes`, `array`, `dict`, `record` — with a **mandatory default**, and
the member answers the default. This is the terminal step of a propagated chain — materialize the miss with a
fallback — and it replaces the test-for-undefined dance:

```go
d := dict({a: 1})
d["missing"].int(0)         // 0    — instead of: v := d["missing"]; if v == undefined { ... }
d["missing"].string("-")    // "-"
d["a"].int(0)               // 1 — on a present value the member is the ordinary conversion
```

The default is answered **as-is** — an explicit opt-out, not a type-checked replacement value
(`undefined.int("n/a")` answers `"n/a"`). And the default is not optional here: with nothing to convert and
no fallback there is nothing to answer, so the no-default form raises:

```go
undefined.int()             // runtime error: cannot convert undefined to int: value is missing
```

There is no free spelling of the rescue — constructors take one value, so `int(d["missing"], 0)` raises
`wrong_num_arguments`; the default belongs to the conversion *member*.

## Member functions

#### `format([spec])`

The universal render.

```go
undefined.format()      // "undefined"
undefined.format("v")   // "undefined"
```

#### `is_true()`

Always `false` — absence is never truthy.

```go
undefined.is_true()   // false
```

#### `int(default)`, `string(default)`, … — the conversions

Every conversion member exists on `undefined`, each demanding a default — the full set is `bool` `byte`
`rune` `int` `float` `decimal` `time` `string` `runes` `bytes` `array` `dict` `record`. See
[Absence and defaults](#absence-and-defaults) above.

```go
undefined.int(0)        // 0
undefined.string("-")   // "-"
undefined.array([])     // []
undefined.int()         // runtime error: cannot convert undefined to int: value is missing
```

## Excluded members

| absent member(s) | why |
| --- | --- |
| `copy()`, `freeze()` | absence has no identity to copy or freeze — a member *about* the absence (`format`, `is_true`) or one that *replaces* it (the defaulted conversions) is admissible; one that operates *on* it is a category error. The free `copy(undefined)` and `freeze(undefined)` raise too |
| `len()`, `contains()`, the whole sequence surface | nothing is there to measure or search |

Any of these raises `invalid_method` (`type undefined has no method copy`).

## Migration notes

- **Slicing now propagates** — `undefined[0:1]` answers `undefined` (it used to raise); slicing is a
  chaining-shaped read, exactly like indexing.
- **`for x in undefined` now raises `not_iterable`** — it used to iterate zero times, silently absorbing a
  missing collection; `is_iterable(undefined)` is now `false`.
- **The free maybe-missing form `T(x, default)` is gone** — `int(d["missing"], 0)` raises
  `wrong_num_arguments`; the rescue is `undefined`'s own conversion member, `d["missing"].int(0)`.
- **`copy(undefined)` and `freeze(undefined)` now raise** (member and free form both) — they used to be
  accepted; absence takes no operation on it. Test with `x == undefined` or `is_undefined(x)` first.
- **Misses answer `undefined`, never an in-band sentinel** — locators no longer answer `-1`; use the trailing
  default (`xs.index(v, -1)`) where a sentinel is genuinely wanted.

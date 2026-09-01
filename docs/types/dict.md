# dict

Mutable string-keyed maps.

## Overview

**A dict is a set of keys, each with an attached value: the key is the element, the value is the attachment.**
That one sentence generates the whole surface. Search members (`contains`, `count`, `index`, `any`, `all`) match
keys; `keys()` and `values()` name the two axes; `filter`/`remove` keep or drop whole entries; `map` transforms
the attachment with the keys fixed. Any operation that would have to choose between the key axis and the value
axis is deliberately absent — instead you name the collection you mean: `d.values().contains(x)`,
`d.values().sum()`, `d.keys().min()`.

Dicts are reference-typed: `b = a` shares the underlying map, `copy()` makes an independent one — see
[container semantics](container-semantics.md). Keys are always strings. Values can be anything.

`dict` and [`record`](record.md) are the same map shape with two access grammars: on a `dict`, `d[k]` is element
access and `d.name(...)` is a member-function call; on a `record`, `r.k` is field access and there are no member
functions at all. Conversions between them are one call in either direction (see
[Conversions](#conversions-array-dict-record-record_view-time-range)).

## Literals and construction

There is no dict literal — `{a: 1}` is a [record](record.md) literal. A dict is built by the `dict()`
constructor:

```go
d = dict({a: 1, b: 2})       // from a record literal: dict({"a": 1, "b": 2})
q = dict({"x y": 1})         // string key form for keys that aren't identifiers
e = dict()                   // empty dict — a real, writable map:
e["x"] = 1                   // dict({"x": 1})

// a dict argument is copied, never aliased — a constructor constructs
c = dict(d)                  // a new, independent dict (shallow: values are shared)
c["a"] = 9                   // d["a"] is still 1
dict(freeze(d))              // mutable again
```

### The entries form

`dict([[k, v], ...])` constructs from a list of entries, where each element must be exactly a 2-element array —
anything else raises (`conversion`). Keys go through their own string conversion; a duplicate key is
last-one-wins. `arr.dict()` is the same reading as a member on `array`:

```go
dict([["a", 1], ["b", 2]])       // dict({"a": 1, "b": 2})
[["a", 1], ["b", 2]].dict()      // same
dict([[1, "x"], [2.5, "y"]])     // dict({"1": "x", "2.5": "y"}) — keys stringified
dict([[1, 2, 3]])                // raises: conversion — an entry is exactly [key, value]
```

The reverse edge is [`d.array()`](#conversions-array-dict-record-record_view-time-range), which answers the
entries key-sorted — the two round-trip up to ordering.

### From a record

```go
dict({a: 1})          // independent dict with the record's entries
dict_view({a: 1})     // a dict sharing the record's map — see record: views
```

The `_view` pair is the one place the constructor rule does **not** apply: `dict()` / `record()` always build
a new value, and `dict_view()` / `record_view()` always share storage — that is their entire job. On an
argument of the target type a view is already a view of itself, so `dict_view(d)` answers `d`.

## Keys and the index operator

Keys are strings. A non-string key stores under (and looks up by) its **canonical text** — the same reading
`.string()` answers: `d[65]` keys `"65"`, `d[b'A']` keys `"A"`, `d[true]` keys `"true"`. A value with no
canonical text — a `dict`, a `record`, a callable, `undefined`, a high octet like `b'\xFF'` — raises
`invalid_index_type`, never silently keys by its rendering. A sequence-typed key (`array`, `range`) raises
too: a transcode is not a key.

```go
d = dict({a: 1})
d[1] = "x"            // stored under "1"
d[1.5] = "y"          // stored under "1.5"
d[true] = "z"         // stored under "true"
d[b'A'] = "w"         // stored under "A" — a byte's canonical text is its ASCII symbol
d["1"]                // "x" — the same entry d[1] wrote

d[{k: 1}]             // raises: invalid_index_type — record has no string conversion
d[dict()]             // raises: invalid_index_type
d[func() { return 1 }]  // raises: invalid_index_type
d[undefined]          // raises: invalid_index_type
d[b'\xFF']            // raises: invalid_index_type — a high octet has no symbol
d[[1, 2]]             // raises: invalid_index_type — a sequence is not a key
d[range(1, 3)]        // raises: invalid_index_type
```

Reading a missing key answers `undefined` — presence and "attached `undefined`" are told apart with
`contains`:

```go
d = dict({a: 1})
d["nope"]             // undefined
d.contains("nope")    // false
```

`d[k] = v` is a statement, not an expression. On a frozen dict it raises `not_assignable`:

```go
f = dict({a: 1}).freeze()
f["b"] = 2            // raises: not_assignable
```

Dot access on a dict is **not** element access — that namespace belongs to member functions, and it is refused
in **both directions**: `d.a` (read) and `d.a = 5` (write) each raise `invalid_selector`
(`type dict has no property a`). In a chain the refusal hits at the first dot — `d.x.y = 1` raises at `.x`.
Element access is always `d["a"]` / `d["a"] = 5`; the selector grammar is [`record`](record.md)'s feature.

## Operators

| operator | meaning | result |
| --- | --- | --- |
| `d1 + d2` | merge, last (right) wins on key collision | `dict` |
| `dict + record`, `record + dict` | merge — either operand being a `dict` makes the result a `dict` | `dict` |
| `d - "key"` | new dict without that entry; missing key is a no-op; non-mutating | `dict` |
| `k in d` | key membership, with `contains`' rules (below) | `bool` |
| `==` / `!=` | deep structural equality; crosses `dict`/`record` freely | `bool` |
| `< > <= >=` | **not defined** — a map has no total order | raises |

```go
dict({a: 1, b: 2}) + dict({b: 9, c: 3})   // dict({"a": 1, "b": 9, "c": 3})
dict({a: 1}) + {b: 2}                     // dict({"a": 1, "b": 2})
{a: 1} + dict({b: 2})                     // dict({"a": 1, "b": 2})

dict({a: 1, b: 2}) - "a"                  // dict({"b": 2}) — the receiver is untouched
dict({a: 1}) - "zz"                       // dict({"a": 1}) — missing key, no-op
```

`-` takes a `string` key only; there is no reflected form (`"key" - d` is not defined).

`in` asks key membership and follows `contains`' acceptance — it converts what a key probe converts and raises
on everything else, never answering a silent `false` for an operand it cannot read:

```go
"a" in dict({a: 1})               // true
"z" in dict({a: 1})               // false
1 in dict([["1", 5]])             // true — the int converts to the key "1"

dict({a: 1}) in dict({a: 1, b: 2})   // raises: not_implemented — the submap reading is reserved
(func() { return 1 }) in dict({a: 1})  // raises: invalid_value — an operator operand is always a value;
                                       // the predicate reading is contains(f)/any(f)
undefined in dict({a: 1})            // raises: invalid_argument_type — not convertible to a key
```

Equality is deep — element values compare by their own equality, key order never matters, and `dict` and
`record` compare across each other:

```go
dict({a: [1, 2]}) == dict({a: [1, 2]})   // true
dict({a: [1, 2]}) == dict({a: [1, 3]})   // false
dict({a: 1}) == {a: 1}                   // true — dict/record cross freely
```

## Iteration

A map's element is its key, so the single-variable form yields **keys**; the two-variable form yields
`(key, value)`; values alone are spelled with the blank:

```go
d = dict({b: 2, a: 1})
for k in d { ... }         // k: "a", "b" — the keys
for k, v in d { ... }      // ("a", 1), ("b", 2)
for _, v in d { ... }      // 1, 2 — the values spelling
```

Iteration order is **not defined**. The members whose answer depends on visit order — `reduce`, `index` — visit
keys in sorted order instead, so their results are deterministic, and so do rendering and encoding: a display,
a `json.encode` payload, and a binary blob are key-sorted and identical on every run.

## Member functions

The two failure modes shared by the surface:

- Every `_in_place` twin raises `not_mutable` on a frozen receiver; on success it mutates the shared map and
  returns the receiver, so mutators chain.
- Every member that takes an argument has **no blank (no-argument) reading**: a map has two axes, so a bare
  `d.filter()` / `d.contains()` / `d.any()` has no single meaning and raises `wrong_num_arguments` rather than
  guessing one.

### Size and the two axes: `len()`, `is_empty()`, `keys()`, `values()`

```go
d = dict({b: 2, a: 1, c: 3})
d.len()          // 3
d.is_empty()     // false
d.keys()         // ["a", "b", "c"] — a fresh array, sorted
d.values()       // [1, 2, 3] — a fresh array, in keys() order
```

`keys()`/`values()` answer new arrays; mutating them does not touch the dict. They are also the spelling for
every value-axis or whole-collection question the dict deliberately does not answer itself:

```go
dict({a: 1, b: 2}).values().sum()        // 3
dict({z: 1, a: 2}).keys().min()          // "a"
dict({a: 1}).values().contains(1)        // true — "does any entry hold this value?"
```

### Add: `merge(...maps)` / `merge_in_place(...maps)`

`merge` is the whole add side. It is variadic over `dict` **and** `record` arguments; entries are applied in
argument order, last one wins; no arguments is a no-op; any non-map argument raises `invalid_argument_type`:

```go
dict({a: 1}).merge(dict({b: 2}), {a: 9})       // dict({"a": 9, "b": 2})
dict({a: 1}).merge(dict({a: 2}), dict({a: 3})) // dict({"a": 3}) — argument order, last wins
dict({a: 1}).merge()                           // dict({"a": 1})
dict({a: 1}).merge([1, 2])                     // raises: invalid_argument_type — dict or record only

m = dict({a: 1})
m.merge_in_place(dict({b: 2}))     // returns m itself; m is now dict({"a": 1, "b": 2})
dict({a: 1}).freeze().merge_in_place(dict({b: 2}))   // raises: not_mutable
```

There is deliberately **no single-entry add member** (`set`, `put`, …). The two spellings are:

```go
d[k] = v                     // the mutating statement
d.merge(dict([[k, v]]))      // non-mutating, via the entries constructor
```

### Remove and keep: `remove(...)` / `remove_in_place(...)`, `filter(...)` / `filter_in_place(...)`

Both act on whole entries and take either a **key set** (variadic strings — or values convertible to keys) or a
**predicate**. `remove` drops the matching entries, `filter` keeps exactly them. A missing key is a silent
no-op. A 1-parameter predicate gets the key; a 2-parameter one gets `(key, value)`:

```go
dict({a: 1, b: 2}).remove("a")                   // dict({"b": 2})
dict({a: 1, b: 2, c: 3}).remove("a", "c")        // dict({"b": 2})
dict({a: 1}).remove("zz")                        // dict({"a": 1}) — absent, no-op
dict({aa: 1, b: 2}).remove(func(k) { return k.len() > 1 })     // dict({"b": 2})
dict({a: 1, b: 2}).remove(func(k, v) { return v > 1 })         // dict({"a": 1})

dict({a: 1, b: 2, c: 3}).filter("a", "c")        // dict({"a": 1, "c": 3})
dict({a: 1, b: 2}).filter(func(k, v) { return v > 1 })         // dict({"b": 2})

dict({a: 1}).filter()      // raises: wrong_num_arguments — a map has no blank reading
dict({a: 1}).remove()      // raises: wrong_num_arguments
```

A match set is homogeneous: mixing a function with keys in one call raises. The `_in_place` twins mutate the
receiver and return it; on a frozen dict they raise `not_mutable`.

### Search, on the key axis: `contains`, `count`, `any`, `all`, `index`

All of these match **keys** — the value axis is `d.values()`'s job. They share three readings: a single value
(converted to a key the way the index operator converts), a variadic set of them ("∈ the set"), or a predicate
(1-arg gets the key, 2-arg gets `(key, value)`). No argument raises; mixing a function into a set raises:

```go
d = dict({a: 1, b: 2})
d.contains("a")                              // true
d.contains(1)                                // false — the key "1" is absent (never matches values)
d.contains("x", "a")                         // true — any of the set
d.contains(func(k) { return k.len() > 1 })   // false

d.count("a", "b", "z")                       // 2
d.count(func(k, v) { return v > 1 })         // 1

d.any("a")                                   // true
d.all(func(k, v) { return v > 0 })           // true

d.contains()                                 // raises: wrong_num_arguments
d.contains(dict({a: 1}))                     // raises: not_implemented — the submap reading is reserved
```

`index(x[, default])` answers a **key** — the map's locator is its element, not a position. It visits keys in
sorted order, so a predicate's answer is deterministic; a miss answers `undefined`, or the trailing default:

```go
dict({apple: 1, pear: 2}).index(func(k, v) { return v == 2 })   // "pear"
dict({z: 1, b: 1, a: 1}).index(func(k, v) { return v == 1 })    // "a" — sorted visit
dict({a: 1}).index("z")                                         // undefined
dict({a: 1}).index("z", "fallback")                             // "fallback"
```

There is no `index_last` — with no defined order, "last" does not exist on a dict.

### Transform: `map(fn)`, `reduce(init, fn)`, `for_each(fn)`

`map` transforms the **attachment** with the keys fixed: the callback's result becomes the new value for that
key, and the answer is a new `dict` of the same keys. A 1-parameter callback gets the key, a 2-parameter one
gets `(key, value)`; anything else raises. Re-keying is a different operation — and one that can collide — so
`map` never touches the key axis (build a new dict via `reduce` or a loop if you need it):

```go
dict({a: 1, b: 2}).map(func(k, v) { return v * 10 })   // dict({"a": 10, "b": 20})
dict({a: 1, b: 2}).map(func(k) { return k + "!" })     // dict({"a": "a!", "b": "b!"})
```

There is no `map_in_place` — `map`'s result is a new dict by contract.

`reduce(init, fn)` folds over the entries in **sorted key order** (deterministic by construction). A
2-parameter callback gets `(acc, key)`, a 3-parameter one gets `(acc, key, value)`:

```go
dict({b: 1, a: 1}).reduce("", func(acc, k) { return acc + k })              // "ab"
dict({b: 2, a: 1, c: 3}).reduce(0, func(acc, k, v) { return acc*10 + v })   // 123
```

`for_each(fn)` makes one full pass — the callback's return value is ignored (early exit is `break`'s job, in a
`for` loop) — and returns the receiver:

```go
d = dict({a: 1})
d.for_each(func(k, v) { ... })     // visits every entry; answers d itself
```

### Conversions: `array()`, `dict()`, `record()`, `record_view()`, `time()`, `range()`

`d.array()` materialises the entries, key-sorted — each entry exactly a 2-element `[key, value]` array. It is
the inverse of the entries constructor, so the two round-trip up to ordering:

```go
dict({b: 2, a: 1}).array()             // [["a", 1], ["b", 2]]
d = dict({b: 2, a: 1})
d.array().dict() == d                  // true
```

`d.dict()` is the identity — the same value, not a copy (every conversion target converts from itself;
independent copies are `copy()`'s job):

```go
d = dict({a: 1})
d.dict() == d          // true — same map
```

`d.record()` answers an independent record with the same entries; `d.record_view()` answers a record **sharing
the dict's map** — the performance opt-in when you want record-style field access over live dict data:

```go
d = dict({a: 1})
r = d.record_view()
d["b"] = 2
r.b                    // 2 — the view sees the write
```

`d.time()` and `d.range()` are the component-form constructors: the dict supplies named components, and an
unknown key raises (`invalid_value`) rather than being ignored:

```go
dict({year: 2026, month: 8, day: 29}).time()   // time("2026-08-29T00:00:00Z")
dict({start: 1, stop: 5}).range()              // range(1, 5)
dict({start: 1, xx: 5}).range()                // raises: invalid_value — unknown component "xx"
```

There is **no `.string()`** on dict, and the free `string(d)` raises too — a map's content has no text form.
The rendering is `format()` / f-strings.

### Copies, freezing, render, truthiness

```go
d = dict({a: [1, 2]})
c = d.copy()                   // deep: c["a"] is an independent array
s = d.copy_shallow()           // one level: s is a new map, s["a"] is d's array

f = dict({a: [1]}).freeze()    // deep: the dict and its elements are immutable
f["a"].push_in_place(2)        // raises: not_mutable — the inner array froze too

g = dict({a: [1]}).freeze_shallow()
g["b"] = 2                     // raises: not_assignable — the map itself is frozen
g["a"].push_in_place(2)        // fine — one level only, the inner array stayed mutable

dict({a: 1}).format()          // the render: dict({"a": 1})
dict({a: 1}).is_true()         // true — a dict is truthy iff non-empty
dict().is_true()               // false
!!dict()                       // false — same test
```

## Exclusions — what a dict deliberately does not have

Every absence below is a decision, not a gap. The reasons come straight from the model: a dict is unordered,
its keys cannot repeat, and it has two axes.

| absent | reason |
| --- | --- |
| `first` `last` `slice` `chunk` `splice` `reverse` `sort` `index_last` | position and order do not exist on an unordered container |
| `has_prefix`/`has_suffix`, `remove_prefix`/`remove_suffix`, `pad_*`, `trim*`, `split*`, `repeat`, `join` | anchored/edge/run operations — same reason: no order, no edges |
| `dedup` / `unique` | not applicable — keys cannot repeat |
| `min` `max` `sum` `avg`, no-arg `any()`/`all()`, the blank set | two axes — name the collection: `d.keys().min()`, `d.values().sum()` |
| `flat_map` | a map has no run to concatenate a mapped result into |
| `contains_value` / `remove_value` | would hard-wire one axis into the name — name the collection instead: `d.values().contains(x)`, `d.filter(func(k, v) { return v != x })` |
| `append` `prepend` `push` `push_first` `insert` | the sequence add verbs; a map's add side is `merge` and `d[k] = v` |
| `.string()` | a map's content has no text form; render with `format()` |
| submap readings of `contains`/`count`/`remove`, and `in` | reserved — they raise `not_implemented` today rather than shipping a guessed meaning |

## Migration notes

Breaking changes from the pre-redesign surface, loud or silent as noted:

- **`for k in d` now yields keys** (a map's element is its key); it used to yield values. This flip is
  *silent*: a single-variable loop body written against values keeps compiling and now sees keys. The values
  spelling is `for _, v in d`.
- **`delete`/`delete_in_place` are renamed `remove`/`remove_in_place`** — one removal verb across the language.
  The old names raise `invalid_method`.
- **`d.a = 5` now raises `invalid_selector`** — dot-assign used to silently write the key `"a"`. The dot is
  refused in both directions; `d["a"] = 5` is the data spelling.
- **`d[container]` raises `invalid_index_type`** where it used to silently key by the value's rendering: a
  `dict`, `record`, `array`, `range`, or callable key is now a runtime error. (Scalars with a canonical text —
  ints, floats, bools — still convert: `d[1]` keys by `"1"`.)
- **A `byte` key stores under its symbol**: `d[b'A']` now keys `"A"` (it used to key `"65"`); a high octet
  (`b'\x80'`–`b'\xFF'`) has no symbol and raises.
- **`filter()` and every other no-argument form raises `wrong_num_arguments`** — a map has no blank reading;
  the old no-arg `filter()` behaviour is gone.
- **`find(fn)` left the language** — the readings live on `contains`/`index`/`count`: `d.index(fn)` answers the
  first matching key (sorted), `d.contains(fn)` asks existence.

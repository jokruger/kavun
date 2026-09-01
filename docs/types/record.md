# record

The field-access shape.

## Overview

A record is the same string-keyed map as a [`dict`](dict.md), carried in the **field-access shape**: `{a: 1}`
is a record literal and `r.a` reads a field. That grammar is the whole design — on a record, `r.anything` is
field access, so **member functions can never exist on it**: `r.len` would be the field `"len"`, not an
operation. A record therefore has **no member surface at all**, and every operation that must reach one has a
free-builtin spelling instead (`len(r)`, `copy(r)`, `remove(r, k)`, …) — which is exactly why those free
builtins exist for every type.

Records are reference-typed: `b = a` shares the map, `copy()` makes an independent one — see
[container semantics](container-semantics.md). Use a record where the data is object-shaped and read by name;
use a `dict` where you need the map operations (`merge`, `keep`, `keys`, …) — the two convert into each other
with one call, and compare equal across the boundary.

## Literals and construction

```go
r = {name: "Alice", age: 30}     // record literal
q = {"x y": 1}                   // string key form for keys that aren't identifiers
n = {a: {b: 1}}                  // nests freely; n.a.b is 1
e = record()                     // the empty record: {}
```

`record([[k, v], ...])` constructs from a list of entries — each element exactly a 2-element `[key, value]`
array, anything else raises (`conversion`); keys go through their own string conversion, last one wins:

```go
record([["a", 1], ["b", 2]])     // {"a": 1, "b": 2}
record(["bad"])                  // raises: conversion — an entry is exactly [key, value]
```

`record(r)` on a record is **not** a pass-through: a constructor constructs, so it answers a new,
independent, mutable record — a shallow copy (`record(freeze(r))` is writable again; the field *values* keep
their own state). A record literal is likewise built at run time, which is why it is mutable while a text
literal is a shared constant — see [Constant literals and constructed
literals](../language.md#constant-literals-and-constructed-literals).

```go
r = {a: 1}
c = record(r)                    // a new record
c.a = 9                          // r.a is still 1
```

`record(m)` / `record_view(m)` convert from a dict — see [Conversions](#conversions-to-and-from-dict).

## Field access

`r.k` and `r["k"]` are the same access. A missing field answers `undefined`. Assignment adds or overwrites a
field; on a frozen record it raises `not_assignable`:

```go
r = {a: 1}
r.a               // 1
r["a"]            // 1
r.missing         // undefined

r.c = 3           // adds a field: {"a": 1, "c": 3}
r["d"] = 4        // same operation, index spelling

f = freeze({a: 1})
f.a = 2           // raises: not_assignable
```

A field holding a function is called through the same access — `r.f()` is "read field `f`, call it", which is
precisely the grammar member functions would collide with:

```go
r = {f: func() { return 7 }}
r.f()             // 7
```

## The free-builtin surface

Everything a member would do elsewhere is a free builtin here. Free and member forms share one domain — each of
these also works on every other type that has the corresponding member:

| free builtin | answer |
| --- | --- |
| `len(r)` | number of fields |
| `copy(r)` / `copy_shallow(r)` | independent deep copy / new map sharing the field values |
| `freeze(r)` / `freeze_shallow(r)` | deep-frozen / map-only-frozen record |
| `format(r[, spec])` | the render (records have no `.string()` — `string(r)` raises) |
| `is_true(r)` | truthiness: non-empty |
| `remove(r, k)` / `remove_in_place(r, k)` | field removal — non-mutating / mutating |
| `is_view(r)` | whether `r` is borrowing another value's map (a `_view` result) |
| `array(r)` | the entries, key-sorted: `[[k, v], ...]` |
| `is_record(r)` / `is_immutable(r)` | type / storage predicates |

```go
r = {name: "Alice", age: 30}
len(r)                       // 2
format(r)                    // {"age": 30, "name": "Alice"} — a map renders key-sorted
is_true(record())            // false — empty
is_true(r)                   // true

remove(r, "age")             // {"name": "Alice"} — r untouched
remove_in_place(r, "age")    // mutates r; raises not_mutable on a frozen record

array({b: 2, a: 1})          // [["a", 1], ["b", 2]] — entries, key-sorted
```

Rendering, encoding **and iteration** are all key-sorted, so a display, a `json.encode` payload, a binary blob
and a `for k in r` pass are the same on every run (see [Iteration](#iteration)).

`copy`/`copy_shallow` and `freeze`/`freeze_shallow` behave exactly as on [`dict`](dict.md#copies-freezing-render-truthiness):
the unsuffixed forms are deep, the `_shallow` twins stop one level down.

## Operators

| operator | meaning | result |
| --- | --- | --- |
| `r1 + r2` | merge, last (right) wins | `record` |
| `record + dict`, `dict + record` | merge — either operand being a `dict` makes the result a `dict` | `dict` |
| `k in r` | key membership, same rules as on `dict` | `bool` |
| `==` / `!=` | deep structural equality; crosses `record`/`dict` freely | `bool` |
| `r - "key"` | **not defined** — raises; field removal is the free `remove(r, k)` | raises |
| `< > <= >=` | **not defined** — a map has no total order | raises |

```go
{a: 1} + {b: 9, c: 3}        // {"a": 1, "b": 9, "c": 3}
{a: 1} + dict({b: 2})        // dict({"a": 1, "b": 2})
"a" in {a: 1}                // true
{a: [1]} == {a: [1]}         // true — deep
dict({a: 1}) == {a: 1}       // true — dict/record cross freely
{a: 1, b: 2} - "a"           // raises: invalid_binary_operator — use remove({a: 1, b: 2}, "a")
```

## Iteration

Same as `dict`: the single-variable form yields **keys**, the two-variable form `(key, value)`, and **key
order is lexical** — a `record` and a `dict` share one iterator, so they agree exactly:

```go
for k in {a: 1, b: 2} { ... }       // "a", "b" — the keys
for k, v in {a: 1, b: 2} { ... }    // ("a", 1), ("b", 2)
for _, v in {a: 1, b: 2} { ... }    // 1, 2 — the values spelling
```

For anything beyond a loop — `keep`, `map`, `merge` with several sources, the key/value arrays — convert to a
`dict` and use its members.

## Conversions, to and from dict

Each direction has an independent-copy form and a `_view` form; the view shares the same underlying map — the
performance opt-in when the copy would be pure overhead:

```go
r = {a: 1}
d = dict(r)          // independent dict — writes to d never reach r
w = dict_view(r)     // shares r's map:
w["b"] = 2
r.b                  // 2 — the record sees the write
is_view(w)           // true

d2 = dict({a: 1})
r2 = d2.record()         // independent record
v2 = d2.record_view()    // shares d2's map
d2["c"] = 9
v2.c                     // 9 — live
```

The entries boundary works in both directions and round-trips up to ordering: `array(r)` answers the key-sorted
`[[k, v], ...]` entries, `record([[k, v], ...])` builds from them.

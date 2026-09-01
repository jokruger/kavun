# Container semantics

Cross-container rules shared by `array`, `bytes`, `runes`, `dict`, and `record`: which assignments share a
body and which copy, how `_in_place` mutation interacts with aliases, deep and shallow `copy`/`freeze`, what a
frozen container still allows, storage-sharing views, nesting, the entries boundary between sequences and
maps, and structural equality. The per-type pages carry each type's member roster; [types.md](../types.md)
carries the family model.

## Value vs reference

Five types have a heap **body**: `array`, `bytes`, `runes`, `dict`, `record`. Assignment binds a second name
to the *same* body — it never copies:

```go
a := [1, 2, 3]
b := a
b[0] = 99
a                        // [99, 2, 3] — one body, two names
```

The same holds for the other body types:

```go
d1 := dict({x: 1})
d2 := d1
d2.merge_in_place(dict({y: 2}))
d1                       // dict({"x": 1, "y": 2})

bs := bytes("abc")
bs2 := bs
bs2[0] = byte(90)
bs                       // bytes([90, 98, 99])

r1 := {a: 1}             // record
r2 := r1
r2.a = 9
r1.a                     // 9
```

Every other type — the scalars, `string`, `range` — is an immutable **value**. There is nothing to share:
no operation can change one, so rebinding is the only way a variable's content moves on, and other names are
never affected:

```go
s1 := "abc"
s2 := s1
s2 = s2 + "d"            // rebinds s2 to a new string
s1                       // "abc"
```

## Pure members and `_in_place` twins

Every not suffixed member is non-mutating: it answers a new value and never touches the receiver. The
`_in_place` twin performs the same operation on the receiver's own body and returns the receiver (so mutators
chain). `y = x.m(...)` and `x.m_in_place(...)` leave the same content in `x`'s role — the difference is only
*where* it lands:

```go
a := [3, 1, 2]
b := a                   // alias
s := a.sort()            // pure: new array
a                        // [3, 1, 2] — untouched
s                        // [1, 2, 3]

a.sort_in_place()
a                        // [1, 2, 3]
b                        // [1, 2, 3] — visible through every alias
a.sort_in_place() == a   // true — the twin returns the receiver
```

`string` and `range` have no `_in_place` members at all — no mutable body.

## `copy()` — deep; `copy_shallow()` — top level only

`copy()` is a **deep** copy: the result shares nothing with the source, at any depth. `copy_shallow()` copies
only the top-level body — the elements themselves are shared:

```go
nested := [[1], [2]]

dc := nested.copy()
dc[0][0] = 99
nested                   // [[1], [2]] — unaffected at every depth

sc := nested.copy_shallow()
sc[0][0] = 77
nested                   // [[77], [2]] — inner bodies shared
sc.push_in_place([3])
nested.len()             // 2 — the top level itself is independent
```

`copy_shallow` (like `freeze_shallow`) exists only on `array` and `dict` — plus the free forms for `record` —
the only types whose elements can themselves be containers. On `bytes`/`runes` the elements are scalars, so
the distinction has nothing to observe and only `copy()` exists.

```go
d := dict({a: [1]})
c := d.copy();          c["a"][0] = 9;  d["a"][0]   // 1 — deep
c2 := d.copy_shallow(); c2["a"][0] = 9; d["a"][0]   // 9 — attachments shared
```

## `freeze()` — deep; `freeze_shallow()` — header only

`freeze()` answers an **independent frozen value**; the receiver itself stays mutable and the two do not
share:

```go
n := [1, 2]
f := n.freeze()
n[0] = 9
n                        // [9, 2] — still mutable
f                        // [1, 2] — frozen snapshot, unaffected
is_immutable(n)          // false
is_immutable(f)          // true
type_name(f)             // "immutable-array"
```

`freeze()` is deep — every nested container is frozen too. `freeze_shallow()` freezes only the header: the
container refuses mutation, but mutable elements inside stay mutable:

```go
f1 := [[1], [2]].freeze()
is_immutable(f1[0])      // true — deep

f2 := [[1], [2]].freeze_shallow()
is_immutable(f2)         // true
is_immutable(f2[0])      // false
f2[0].push_in_place(9)
f2[0]                    // [1, 9] — the header is frozen, the elements are not
```

On a frozen container **reads work unchanged** — indexing, `len()`, iteration, and every pure member (whose
result is an ordinary mutable value again). The two write paths raise, each with its own error kind:

| operation | raises | message shape |
| --- | --- | --- |
| element assignment `f[0] = x`, `f.k = x` | kind `not_assignable` | `type immutable-array does not support assignment via indexing or field access` |
| any `_in_place` member | kind `not_mutable` | `(sort_in_place) type immutable-array is immutable` |

```go
fz := [3, 1].freeze()
fz[0]                    // 3
fz.sort()                // [1, 3] — pure members work
is_immutable(fz.sort())  // false — results are mutable again
fz.sort_in_place()       // raises, kind "not_mutable"
```

Thawing is spelled `copy()` — the deep copy of a frozen container is mutable:

```go
is_immutable([1].freeze().copy())   // false
```

`is_immutable(x)` is a free predicate with universal domain; it reads the header, so it answers `true` for
every immutable value (`is_immutable("abc")`, `is_immutable(5)`, `is_immutable(1..3)` are all `true`).

## Views

`slice_view(i, j)` and `chunk_view(n)` are the storage-sharing twins of `slice`/`chunk`: the result is a
window onto the receiver's own body. Mutations travel in **both** directions:

```go
src := [1, 2, 3, 4]
v := src.slice_view(1, 3)
v                        // [2, 3]
v[0] = 99
src                      // [1, 99, 3, 4] — through the view into the source
src[2] = 55
v                        // [99, 55] — source changes visible in the view

cv := src.chunk_view(2)  // [[1, 99], [55, 4]] — each chunk shares storage
cv[0][0] = 11
src                      // [11, 99, 55, 4]
```

Plain `slice`/`chunk` copy — mutating their result never touches the source. A view keeps its receiver's type
(`type_name(v)` is `"array"`); the free predicate `is_view(x)` tells them apart, and answers `false` on
anything that is not a view:

```go
is_view(v)               // true
is_view(src)             // false
is_view(5)               // false
is_view(copy(v))         // false — copy materializes
```

The `_view` members exist only where sharing is observable — on the mutable sequence bodies `array`, `bytes`,
`runes`. `string` and `range` have none: on an immutable value sharing cannot be observed, so `slice` already
*is* the zero-copy form (`(1..9).slice(1, 3)` answers a `range` computed from the bounds, nothing
materialized).

## Nesting, rows, and the spread trap

Arrays nest freely — `[[1, 2], [3, 4]]` is an array of arrays, and that shape is the **entries idiom**: a
`[[key, value], ...]` array crosses the boundary to maps and back.

```go
dict([["a", 1], ["b", 2]])         // dict({"a": 1, "b": 2})
dict([["a", 1], ["a", 2]])         // dict({"a": 2}) — last wins
[["a", 1]].record()                // a record
dict({b: 2, a: 1}).array()         // [["a", 1], ["b", 2]] — entries, key-sorted
```

**The trap:** on the add side, `append`, `prepend`, `splice`, and `+` read an `array` operand as a *run* and
spread its elements. Building a list of rows with `append(row)` silently flattens:

```go
// WRONG — append spreads an array operand
rows := []
rows = rows.append([10, 20])
rows = rows.append([30, 40])
rows                     // [10, 20, 30, 40] — four numbers, no rows

// RIGHT — push adds each argument as one element, whatever its type
rows2 := []
rows2.push_in_place([10, 20])
rows2.push_in_place([30, 40])
rows2                    // [[10, 20], [30, 40]]
```

`insert` is the element-inserting sibling — it never spreads — and wrapping is the general escape hatch:

```go
[9].insert(1, [1, 2])        // [9, [1, 2]]
[9, 8].splice(1, 0, [1, 2])  // [9, 1, 2, 8] — splice's inserts spread, like append
[] + [[1, 2]]                // [[1, 2]] — wrap one level to add an array as an element
```

## Equality

`==` is **deep and structural** across containers — element by element, entry by entry, at every depth:

```go
[1, [2, dict({a: 3})]] == [1, [2, dict({a: 3})]]   // true
[1, 2] != [1, 3]                                   // true
```

A `dict` and a `record` with the same entries are equal — equality compares content, not container kind — and
frozen-ness does not participate either:

```go
dict({a: 1, b: [2]}) == {a: 1, b: [2]}   // true
[1, 2].freeze() == [1, 2]                // true
```

`string` and `runes` holding the same text are equal (`"abc" == u"abc"`), and a view equals any container with
the same elements — `is_view`/`is_immutable`/`type_name` are the header questions, `==` never asks them.

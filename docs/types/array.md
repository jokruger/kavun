# array

Mutable, heterogeneous sequences of values.

## Overview

Arrays are ordered, mutable sequences whose elements can be values of any type, including other arrays.

Arrays are reference-typed: `b = a` makes both names point at the same body; use `copy()` for an independent
value. Every member without an `_in_place` suffix is non-mutating and returns a new array; the `_in_place` twins
mutate the shared body (see [In-place twins](#in-place-twins)).

## Literals and construction

```go
uniform = [1, 2, 3]
mixed = [1, "two", 3.0, true] // elements are any type
```

The free `array(...)` constructor has two forms with two different jobs:
- **Arity 1**: it decomposes a convertible sequence into elements and wraps any other value as one element.
- **Arity 2**: `array(x, n)` is `n` copies of `x` as one element each and never spreads, whatever `x` is.

```go
array()                  // []
array(range(1, 4))       // [1, 2, 3]
array("abc")             // ['a', 'b', 'c'] — a string materializes as its runes
array(5)                 // [5] — a non-sequence value is one element
array(-1)                // [-1]
array(undefined)         // Error: cannot convert undefined to array: value is missing

// the count form: n copies of x, x stays whole
array("ab", 2)           // ["ab", "ab"]
array([1, 2], 2)         // [[1, 2], [1, 2]] — never spreads
array(undefined, 3)      // [undefined, undefined, undefined] — the preallocation spelling
```

### Indexing and slicing

```go
a = [10, 20, 30, 40, 50]
a[0]             // 10
a[-1]            // 50 — negative indices count from the end
a[1:3]           // [20, 30]
a[::-1]          // [50, 40, 30, 20, 10] — three-part slice with negative step
a[7]             // Error: (index access) 7 out of range [0, 5]
```

Slice bounds clamp; single-element access raises out of range. Index assignment (`a[0] = 99`) mutates in place
and raises on an immutable array.

## Operators

| operator | reading |
| --- | --- |
| `a + x` | exactly `append(x)`: an `array` operand spreads its elements in as a run; **every other value** is appended as one element; `undefined` propagates (`a + undefined` → `undefined`); an `error` operand raises |
| `x + a` | exactly `prepend(x)` — the mirror, for a left operand that has no reading of its own for an array |
| `a * n` | exactly `repeat(n)`: the right operand is a **count**, not an element. No reflected direction — `n * a` raises |
| `a - x` | exactly `remove(x)`: an `array` operand removes every occurrence of that contiguous run — never set difference; every other value removes every equal element; `undefined` propagates |
| `x in a` | the value readings of `contains`: element or run; a callable operand raises — an operator operand is always a value |
| `==` / `!=` | deep equality, element by element, through any nesting |

```go
[1, 2] + [3, 4]          // [1, 2, 3, 4]
[1, 2] + 3               // [1, 2, 3] — one element
[1, 2] + "ab"            // [1, 2, "ab"] — not an array: one element
[9] + (1..4)             // [9, range(1, 4)] — a range is one element too; see below
[1, 2] + undefined       // undefined
[1, 2] + error("x")      // Error: invalid_binary_operator: array + error
```

An array on the **right** works too, and means `prepend`. A left operand that has no reading of its own for an
array hands the operation over, and the array adds it at the front:

```go
3 + [1, 2]               // [3, 1, 2]   — the mirror of [1, 2] + 3
"ab" + [1, 2]            // ["ab", 1, 2]
(1..3) + [1]             // [range(1, 3), 1]
undefined + [1]          // undefined   — the universal contracts still come first
error("x") + [1]         // Error: invalid_binary_operator: error + array
```

Only `+` has this reflected form, because only the add side has a front member. Removal has no `prepend`
counterpart, so every other operator raises rather than inventing one:

```go
3 - [1, 2]               // Error: invalid_binary_operator: int - array
3 * [1, 2]               // Error: invalid_binary_operator: int * array
```

The operator adds **one** element at the front, and `+` is left-associative, so it does not chain the way
`prepend` does — `1 + 2 + [3]` is `(1 + 2) + [3]` = `[3, 3]`. Use `prepend` for more than one:

```go
[1].prepend(2, 3)        // [2, 3, 1]
2 + 3 + [1]              // [5, 1]  — the ints add first
```

`*` is `repeat`'s operator form — the right operand is a **count**, not an element:

```go
[1, 2] * 3               // [1, 2, 1, 2, 1, 2]
[1, 2] * 0               // []
[1, 2] * 2.0             // [1, 2, 1, 2] — a lossless numeric is a count
[1, 2] * 2.5             // Error: (*) argument right operand must be a whole number
[1, 2] * -1              // Error: (*) repeat count must be non-negative
[1, 2] * [3]             // Error: invalid_binary_operator: array * array — only a number is a count
3 * [1, 2]               // Error: invalid_binary_operator: int * array
```

There is **no** reflected direction here, unlike `+`: `a * n` reads as "apply `n` to the sequence", while
`n * a` would read as "apply the sequence to `n`", which has no meaning. Element-wise arithmetic — multiplying
each element by `n` — is a separate, future operator family (`.+`, `.-`, `.*`, `./`), never plain `*`.

`-` removes by value, every occurrence:

```go
[1, 2, 1, 2, 3] - 1      // [2, 2, 3] — every equal element
[1, 2, 3, 1, 2] - [1, 2] // [3] — every occurrence of the contiguous run
[1, 2, 3] - (1..3)       // [1, 2, 3] — a range is one element, and no element equals it
[1, [2], 1] - [2]        // [1, [2], 1] — [2] means the element 2, which does not occur
```

`in` accepts values only:

```go
2 in [1, 2, 3]                     // true
[2, 3] in [1, 2, 3]                // true — a run
undefined in [undefined, 1]        // true
[{a: [1]}] == [{a: [1]}]           // true — deep equality, through any nesting
f = func(x) { return true }
f in [1, 2]    // Error: (in) an operator operand is always a value — the predicate reading is contains(f)/any(f)
```

`+=`, `-=` and `*=` are the usual sugar: `a += 3` appends one element, `a -= 1` removes every `1`, `a *= 3`
repeats.

## Argument dispatch

Every member that searches, matches, or adds selects its *reading* from the argument's type — one menu, shared
by members and operators alike:

| argument | reading |
| --- | --- |
| absent | the [blank set](#the-blank-set) |
| a function | a predicate — `f/1` receives the element, `f/2` receives `(index, element)` |
| an `array` — the receiver's own kind, and nothing else | a contiguous **run** of elements |
| anything else, `range` and the text types included | **one element** |
| a one-element array `[x]` | the element `x` — so wrapping is the uniform way to spell "this array, as one element": `[[2, 3]]` means the single element `[2, 3]` |

**Only an `array` spreads — nothing else, however sequence-like.** A `range`, a `string`, a `runes`, a `bytes`
are each one element. Spreading is never inferred from a type; it is spelled at the call site:

```go
[9] + (1..4)                  // [9, range(1, 4)]
[9] + (1..4).array()          // [9, 1, 2, 3]
[1] + "ab"                    // [1, "ab"]
[1] + "ab".array()            // [1, 'a', 'b']
```

Two reasons, and they will still hold when the language grows typed vectors (`ints`, `floats`, …):

- `string` and `runes` hold the *same* text and are interchangeable everywhere (see [runes](runes.md)), so
  neither can spread while the other stays an element — and `arr + "text"` must keep meaning *append the text*.
- a `range` that quietly became an `array` would answer an `array` today and an `ints` tomorrow, silently, in a
  script that named neither. `.array()` puts the target where a reader — and a migration — can find it.

```go
[1, 2, 3].contains(2)              // true — element
[1, 2, 3].contains([2, 3])         // true — run
[1, [2, 3], 4].contains([[2, 3]])  // true — the wrap: the element [2, 3]
[1, 2, 3].contains(func(x) { return x > 2 })   // true — predicate
```

A variadic call is a homogeneous **set** — "equals the argument" becomes "is in the set". Every argument in one
call must have the same reading; mixing elements with runs raises, and a function among several arguments always
raises:

```go
[1, 2, 3].contains(4, 2)           // true — any of the set
[1, 2, 1, 3].count(1, 3)           // 3
[1, 2, 3].contains(9, [2, 3])      // Error: a HOMOGENEOUS set — one reading per call
```

Each member declares which readings it takes; a reading the member does not declare raises an error naming the
ones it has (see `any`/`all` and `trim` below for members that refuse runs).

Run matching is **leftmost-longest and non-overlapping**:

```go
[1, 1, 1, 1].count([1, 1])                       // 2 — non-overlapping
["a", "b", "c", "d"].remove(["a", "b"], ["a", "b", "c"])  // ["d"] — longest wins at a tie
```

The empty run matches nothing for counting and keeping verbs, but is "contained" everywhere:

```go
[1, 2].contains([])      // true
[1, 2].count([])         // 0
[1, 2].remove([])        // [1, 2]
```

## Adding

**`append(...xs)`** / **`prepend(...xs)`** — whole-operand add at the back/front; `x.append(a, b)` ≡
`x + a + b`. An `array` argument spreads as a run; every other value is one element. Arguments land in
order — `x.prepend(a, b)` puts `a, b` in that order at the front. No arguments is a no-op.

```go
[1].append(2, 3)             // [1, 2, 3]
[1].append([2, 3])           // [1, 2, 3] — spreads
[3].prepend(1, 2)            // [1, 2, 3] — argument order preserved at the front
[0].append(1, [2, 3], 4)     // [0, 1, 2, 3, 4]
```

`prepend` is O(n) — it rebuilds the array. When building front-first in a loop, accumulate with
`append`/`push` and `reverse()` once at the end.

**`push(...items)`** / **`push_first(...items)`** — each argument is exactly **one element**, whatever its type;
an array never spreads. Postcondition: `a.push(x).last() == x`.

```go
[1].push([2, 3])             // [1, [2, 3]] — the pair stays a pair
[2].push_first(0, 1)         // [0, 1, 2]
```

**`insert(i, ...items)`** — positional edit: inserts each argument as one element (never spreads) at position
`i`. `i` outside `[0, len]` raises; a negative `i` counts from the end.

```go
[1, 4].insert(1, 2, 3)       // [1, 2, 3, 4]
[1, 4].insert(1, [2, 3])     // [1, [2, 3], 4] — one element
[1, 2].insert(-1, 9)         // [1, 9, 2]
[1, 2].insert(3, 9)          // Error: (insert) 3 out of range [0, 2]
```

**`splice(start[, delete_count[, ...items]])`** — removes `delete_count` elements at `start`, then inserts the
items there. The inserts take the add-side reading: an `array` argument **spreads** — the wrap (or `insert`)
spells the element. `start` outside `[0, len]` raises: reading past the end is harmless (`slice` clamps), editing
past it is not.

```go
[1, 2, 3, 4].splice(1, 2)          // [1, 4]
[1, 5].splice(1, 0, [2, 3], 4)     // [1, 2, 3, 4, 5] — inserts spread
[1, 5].splice(1, 0, [[2, 3]])      // [1, [2, 3], 5] — the wrap spells the element
[1, 2].splice(5)                   // Error: (splice, start index) 5 out of range [0, 2]
```

## Removing and searching

**`remove(x)`** / **`filter(x)`** — synonym-duals over the same readings: `remove` drops what matches, `filter`
keeps it. Element and run readings remove/keep **every occurrence**; absence of a match is a silent no-op. The
no-argument forms are synonyms of each other: both drop the [blank set](#the-blank-set).

```go
[1, 2, 1, 3].remove(1)                     // [2, 3]
[1, 2, 3, 1, 2].remove([1, 2])             // [3] — every occurrence of the run
[1, 2, 3, 4].remove(func(x) { return x % 2 == 0 })   // [1, 3]
[1, 2, 3, 4].filter(func(x) { return x % 2 == 0 })   // [2, 4]
[1, 2, 3, 4].remove(1, 3)                  // [2, 4] — a set
[1, 0, undefined, "", 0.0, [], 2].remove() // [1, 2] — drops the blank set
```

**`contains(x)`** — all five readings: element, run, variadic set, predicate, and no-argument (any element
outside the blank set). `contains(fn)` ≡ `any(fn)` and `contains()` ≡ `any()`.

```go
[1, 2, 3].contains([2, 3])         // true — a run
[1, 2, 3].contains(range(2, 4))    // false — a range is one element; say .array() for the run
[0, undefined].contains()          // false — only blanks
[0, undefined, 1].contains()       // true
```

**`count(x)`** — same five readings, answering how many (non-overlapping occurrences, for a run).

```go
[1, 2, 1, 1].count(1)              // 3
[1, 2, 1, 2, 2].count([1, 2])      // 2
[1, 0, undefined, "", 2].count()   // 2 — significant elements
```

**`any(x)`** / **`all(x)`** — value, predicate, set, or no-argument (truthiness against the blank set). A run
argument raises **permanently** — "all of a run" has no universal reading:

```go
[2, 4].all(func(x) { return x % 2 == 0 })  // true
[2, 2].all(2)            // true
[1, 2].any(5, 2)         // true — a set
["", 0, "x"].any()       // true — some element is significant
[1, 2, 3].any([2, 3])    // Error: (any) ... an element or a predicate (no run reading)
```

**`index(x[, d])`** / **`index_last(x[, d])`** — the only locators: first/last position of an element, run, or
predicate match. Never variadic — the optional trailing argument is always the miss default. A miss answers
`undefined` (never `-1`). The no-argument form finds the first/last element outside the blank set.

```go
[1, 2, 3, 2].index(2)          // 1
[1, 2, 3, 2].index_last(2)     // 3
[1, 2].index(9)                // undefined — a miss, never -1
[1, 2].index(9, -1)            // -1 — the explicit default
[0, undefined, "", 7, 0].index()   // 3 — first significant element
[1, 2, 3].index(func(x) { return x > 1 })   // 1
```

## Structural edits

These are sequence verbs, not text verbs: a fill *element*, a set of *elements*, a run-or-element substitution.

**`trim(...set)`** / **`trim_start(...set)`** / **`trim_end(...set)`** — repeatedly drop edge elements while
they belong to the set. Elements only — a run argument raises (the anchored run form is
`remove_prefix`/`remove_suffix`), and there is no predicate reading. No arguments trims the
[blank set](#the-blank-set).

```go
[0, 1, 2, 0, 0].trim(0)            // [1, 2]
[0, 9, 1, 9, 0].trim(0, 9)         // [1]
[undefined, 0, 1, 0, undefined].trim()   // [1]
[1, 2, 3].trim([1, 2])   // Error: (trim) ... a set of elements (the anchored run form is remove_prefix/remove_suffix)
```

**`remove_prefix(x)`** / **`remove_suffix(x)`** — remove one exact element-or-run, **once**, anchored at the
edge; absent → unchanged.

```go
[1, 2, 1, 2, 3].remove_prefix([1, 2])   // [1, 2, 3] — once, not repeatedly
[1, 2, 3].remove_suffix([2, 3])         // [1]
[1, 2].remove_prefix([9])               // [1, 2] — no match, unchanged
```

**`has_prefix(x)`** / **`has_suffix(x)`** — anchored tests: element, run, or a variadic any-of set. No predicate
("the first element satisfies `f`" is `index(f) == 0`) and no no-argument form.

```go
[1, 2, 3].has_prefix(1)            // true
[1, 2, 3].has_prefix([1, 2])       // true
[1, 2, 3].has_prefix(9, 1)         // true — any of the set
```

**`replace(old, new)`** — substitutes every occurrence. Both positions take an element or a run; never variadic
(the second position is the replacement), never a predicate.

```go
[1, 2, 1].replace(1, 9)            // [9, 2, 9]
[1, 2, 3, 1, 2].replace([1, 2], 0) // [0, 3, 0] — run to element
[1, 2].replace(1, [7, 8])          // [7, 8, 2] — element to run
```

**`pad_start(n[, fill])`** / **`pad_end(n[, fill])`** — pad with the fill element until length `n`. The fill is
always **one element**, whatever its type — an array fill is inserted whole per slot, never spread — and
defaults to `undefined`. A width at or below the current length is a no-op.

```go
[1, 2].pad_start(5, 0)     // [0, 0, 0, 1, 2]
[1, 2].pad_end(4)          // [1, 2, undefined, undefined]
[1].pad_end(3, [0, 0])     // [1, [0, 0], [0, 0]] — the fill is one element
[1, 2].pad_start(1, 0)     // [1, 2] — width below length: no-op
```

## Transforms

**`map(fn)`** — strictly 1:1. A nested result nests; `undefined` stays:

```go
[1, 2, 3].map(func(x) { return x * 2 })        // [2, 4, 6]
[10, 20].map(func(i, v) { return i + v })      // [10, 21]
[1, 2].map(func(x) { return [x] })             // [[1], [2]] — nesting preserved
[1, undefined, 2].map(func(x) { return x })    // [1, undefined, 2]
```

**`flat_map(fn)`** — map, then concatenate: an `array` callback result spreads as a run, `undefined`
contributes nothing, anything else is one element.

```go
[1, 2].flat_map(func(x) { return [x, x * 10] })   // [1, 10, 2, 20]
[2, 3].flat_map(func(x) { return range(0, x).array() })   // [0, 1, 0, 1, 2]
[1, 2, 3].flat_map(func(x) { if x == 2 { return undefined }; return x })   // [1, 3]
```

**`reduce(init, fn)`** — folds left. The callback takes `(acc, elem)` or `(acc, index, elem)`:

```go
[1, 2, 3].reduce(0, func(acc, v) { return acc + v })          // 6
[1, 2, 3].reduce(0, func(acc, i, v) { return acc + i * v })   // 8
```

**`for_each(fn)`** — always a **full pass**; the callback's return value is ignored, and the member returns the
receiver. Early exit belongs to `for`/`break` or a search member.

```go
sum = 0
r = [1, 2, 3].for_each(func(v) { sum += v; return false })
sum    // 6 — the false did not stop it
r      // [1, 2, 3] — the receiver
```

**`sort()`** / **`reverse()`** / **`dedup()`** / **`unique()`** — ordering transforms; elements must be mutually
comparable for `sort`.

```go
[3, 1, 2].sort()             // [1, 2, 3]
[1, 2, 3].reverse()          // [3, 2, 1]
[1, 1, 2, 2, 1].dedup()      // [1, 2, 1] — collapses consecutive runs only
[3, 1, 3, 2, 1].unique()     // [3, 1, 2] — first occurrence wins
```

**`flatten([depth])`** — unwraps nested `array` elements, one level by default. Only array elements are
unwrapped; every other element type stays intact. `flat_map(f)` ≠ `map(f).flatten()` — `flatten` unwraps any
nested array, at any position.

```go
[1, [2, [3]]].flatten()          // [1, 2, [3]] — one level
[1, [2, [3]]].flatten(2)         // [1, 2, 3]
[range(0, 2), [1]].flatten()     // [range(0, 2), 1] — only arrays unwrap
```

## Aggregation

`first`, `last`, `min`, `max`, `sum`, `avg` all take the uniform optional trailing default, answered when the
receiver is empty (or, for `sum`/`avg`, when there is nothing to add). Without it, absence answers `undefined` —
never an in-band sentinel. `min`/`max` need mutually comparable elements; `sum`/`avg` need numeric ones. With
all-`int` input `avg` is an `int` (integer division) — include a float for a fractional mean.

```go
[1, 2, 3].first()        // 1
[].first()               // undefined
[].first(0)              // 0
[1, 2, 3].sum()          // 6
[1, 2].avg()             // 1 — all-int mean is an int
[1, 2.0].avg()           // 1.5
[undefined, 7].first()   // undefined — first answers the plain first element
```

## Slices and chunks

**`slice([start[, end]])`** — a copying sub-range; bounds **clamp** (contrast `splice`, which raises), negative
indices count from the end. Equivalent to `a[start:end]` with a name usable in a chain.

```go
a = [1, 2, 3]
a.slice(1, 99)     // [2, 3] — clamps
a.slice(-2)        // [2, 3]
a.slice()          // [1, 2, 3] — full copy
```

**`slice_view(start, end)`** — the sharing twin: the result aliases the source's storage, so mutation flows both
ways. The free `is_view(x)` reports it; see [container semantics](container-semantics.md) before using views.

```go
a = [1, 2, 3, 4]
b = a.slice_view(1, 3)
b[0] = 99
a              // [1, 99, 3, 4]
is_view(b)     // true
```

**`chunk(n)`** / **`chunk_view(n)`** — split into arrays of up to `n` elements; `chunk` copies, `chunk_view`
shares:

```go
[1, 2, 3, 4, 5].chunk(2)       // [[1, 2], [3, 4], [5]]
a = [1, 2, 3, 4]
c = a.chunk_view(2)
c[0][0] = 9
a                              // [9, 2, 3, 4]
```

## join and repeat

**`join([sep])`** — the collection is the receiver, the separator the optional argument. Elements are rendered;
the result type follows the separator: `string` with no argument or a `string` separator, `bytes` with a `bytes`
separator, `runes` with a `runes` or `rune` separator. A container element raises — a nested collection is not
renderable — and so do `undefined` and callables.

```go
[1, 2, 3].join(", ")       // "1, 2, 3"
[1, 2, 3].join()           // "123"
[1, 2].join(bytes("-"))    // bytes([49, 45, 50]) — "1-2" as bytes
[1, 2].join(u"-")          // u"1-2"
[1, [2, 3]].join(",")      // Error: ... a nested collection is not renderable in join
```

**`repeat(n)`** — sequence self-concatenation, `n` times; `a * n` is the operator form. The count must be a
whole non-negative number: `2.0` converts, `1.5` raises — never silent truncation. A result past
`4294967296` elements raises rather than exhausting the host.

```go
[1, 2].repeat(3)       // [1, 2, 1, 2, 1, 2]
[1, 2] * 3             // [1, 2, 1, 2, 1, 2] — the operator form
[1, 2].repeat(2.0)     // [1, 2, 1, 2]
[1, 2].repeat(1.5)     // Error: (repeat) argument first must be a whole number, got 1.5
```

## Conversions

Container conversions are the element conversion applied element-wise, **all-or-nothing** — one inconvertible
element fails the whole call.

**`string()`** / **`runes()`** / **`bytes()`** — each element becomes one symbol/octet: ints are read as code
points (for `bytes`, octet values 0–255), runes and bytes as themselves.

```go
[72, 105].string()       // "Hi"
[72, 105].runes()        // u"Hi"
[72, 105].bytes()        // bytes([72, 105])
[1, "a"].string()        // Error: cannot convert array to string
```

**`dict()`** / **`record()`** — the **entries** reading: every element must be exactly a 2-element array
`[key, value]`. Keys go through their own string conversion; a repeated key — last wins.

```go
[["usd", 100], ["eur", 90]].dict()   // dict({"eur": 90, "usd": 100})
[[1, "x"], [true, "y"]].dict()       // dict({"1": "x", "true": "y"})
[["a", 1], ["a", 2]].dict()          // dict({"a": 2}) — last wins
[["a", 1]].record()                  // {"a": 1}
```

**`array()`** — the identity: the same value, not a copy (`a.array() == a` is `true`).

## Copies, freezing, rendering, truthiness

- **`copy()`** — independent deep copy. **`copy_shallow()`** — fresh top level, nested values still shared.
- **`freeze()`** — deep-immutable *copy*; the source and its aliases are untouched. **`freeze_shallow()`** —
  marks the header you hold immutable without detaching; reassign to see it (`a = a.freeze_shallow()`).
- **`format([spec])`** — the render, per the [format mini-language](../format-mini-language.md).
- **`is_true()`** — truthiness is non-emptiness: `!!x` ⟺ `x != array()`.

```go
a = [[1]]
b = a.copy()
b[0][0] = 9            // a is still [[1]]
c = a.copy_shallow()
c[0][0] = 7            // a is now [[7]] — nested values shared

d = [1]
f = d.freeze()
d[0] = 9               // d is [9]; f stays [1]
is_immutable(f)        // true

[0].is_true()          // true — non-empty, even if every element is blank
```

The free forms `len(x)`, `copy(x)`, `freeze(x)`, `format(x)`, `is_true(x)` are the same operations by another
spelling.

## The blank set

The no-argument form of `remove`/`filter`, `trim`/`trim_start`/`trim_end`, `contains`/`any`/`all`, `count`, and
`index`/`index_last` matches the array's **blank set**: `undefined` together with each element type's own zero
value — `0`, `0.0`, `""`, empty containers, and so on. Blankness is about separators and filler, not arithmetic:
a script in which zeros are data names its own set instead.

```go
[1, 0, undefined, "", 0.0, [], 2].remove()   // [1, 2]
[undefined, 0, 5, 0, undefined].trim(undefined)   // [0, 5, 0] — zeros are data here
```

The default pad fill is `undefined` — the blank set's canonical member.

## In-place twins

Every mutating member is the `_in_place` twin of a non-mutating one with the same arguments and readings:

`append_in_place`, `prepend_in_place`, `push_in_place`, `push_first_in_place`, `insert_in_place`,
`splice_in_place`, `remove_in_place`, `filter_in_place`, `sort_in_place`, `reverse_in_place`,
`dedup_in_place`, `unique_in_place`, `trim_in_place`, `trim_start_in_place`, `trim_end_in_place`,
`remove_prefix_in_place`, `remove_suffix_in_place`, `replace_in_place`, `pad_start_in_place`,
`pad_end_in_place`.

A twin mutates the shared body — the change is visible through every alias, no reassignment needed — and
returns the receiver, so twins chain and `y = x.m(args)` leaves the same content in `x` as `x.m_in_place(args)`.
On a frozen receiver every twin raises kind `not_mutable`:

```go
a = [1]
b = a                        // b shares a's body
a.append_in_place(2, 3)
b                            // [1, 2, 3] — no reassignment needed

[1].freeze().append_in_place(2)   // Error: (append_in_place) type immutable-array is immutable
```

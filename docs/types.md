# Type Reference

Kavun's types form a small number of **shape families** that share one member surface, rather than a flat list
of types each with its own API. This page is the map: what types exist, which family each belongs to, and the
handful of language-wide rules — argument dispatch, the blank set, the universal members, conversions,
member/operator correspondence, callbacks, mutability, truthiness — that every family obeys. The per-type pages
carry the full contracts; [types/function-matrix.md](types/function-matrix.md) is the existence table, and
[types/container-semantics.md](types/container-semantics.md) covers copying, freezing, aliasing, and views.

## The roster

| type | identity |
| --- | --- |
| [undefined](types/undefined.md) | the absence of a value; propagates through operators, raises at the point of use |
| [bool](types/bool.md) | `true` / `false` |
| [int](types/int.md) | signed 64-bit integer; overflow raises |
| [float](types/float.md) | IEEE 754 double; NaN/±Inf are storable error states, not domain values |
| [decimal](types/decimal.md) | exact decimal arithmetic for money and precision-critical work |
| [byte](types/byte.md) | one octet, 0–255; ordinal, not numeric — arithmetic wraps mod 256 |
| [rune](types/rune.md) | one Unicode code point; ordinal, not numeric |
| [time](types/time.md) | one instant, nanosecond precision, zone-aware |
| [error](types/error.md) | a wrapped payload plus classification (`kind`, `is_fatal`, `is_runtime`); always truthy |
| [string](types/string.md) | immutable Unicode text in compact UTF-8; element = the symbol (rune) |
| [runes](types/runes.md) | mutable Unicode text as a materialized symbol array; element = the rune |
| [bytes](types/bytes.md) | mutable raw octet sequence; element = the byte |
| [array](types/array.md) | mutable heterogeneous sequence; the general-purpose container |
| [range](types/range.md) | lazy integer sequence `start..stop` (stop excluded); immutable, computed, never materialized |
| [dict](types/dict.md) | mutable set of keys with attached values; the key is the element |
| [record](types/record.md) | field bag with `r.field` access; **no member surface by design** — free builtins only |
| function / closure / builtin | callables; `type_name()` answers `"function"` for all three |

## The type-shape families

Every type belongs to exactly one family; the family decides which member blocks it carries.

### Scalars

`bool`, `int`, `float`, `decimal`, `byte`, `rune`, `time`, `error`, `undefined` — values with no elements: no
`len()`, no iteration, no indexing (`undefined` is the one exception in form only — indexing and slicing it
propagate `undefined` rather than raising, so a lookup chain can miss at any level; see its page). Their surface is conversions plus domain members (numeric predicates and
`abs`/`sign` on `int`/`float`/`decimal`; calendar accessors on `time`; `kind`/`value`/`is_fatal`/`is_runtime`
on `error`). `byte` and `rune` are *ordinal* — comparable and orderable but not arithmetic — and they are the
two scalars with text content: they promote into `bytes`/`runes` and carry `.string()`/`.runes()` as content
conversions (`byte(65).string()` is `"A"`, not `"65"`; the render `byte(65).format()` is `"65"`).

### The sequence family

`array`, `string`, `runes`, `bytes`, `range` — one shared surface:

- size and edges: `len` `is_empty` `first` `last`
- the match members: `contains` `count` `index` `index_last` `any` `all`
- add / search / remove: `append` `prepend` `push` `push_first` `insert` `remove` `filter` `splice`,
  the trim/pad/replace/prefix/suffix verbs
- slicing and order: `slice` `chunk` `reverse` `sort` `dedup` `unique`
- iteration / aggregation: `for_each` `reduce` `map` `flat_map` `min` `max` (`sum`/`avg` where the elements
  are numeric)

Each type takes the subset its traits allow — `string` drops the mutating twins (it has none), `range` drops
every member that would answer a new materialized sequence (`(1..3).map(f)` raises; write
`(1..3).array().map(f)`), the text triple owns `split`/`partition`/`split_lines`, `array` and `range` own
`join`.

Within the family, two **sub-families** decide what a run argument is (see the dispatch section below):

- **The text triple** — `string`, `runes`, `bytes`. Closed: every accepted argument is *text content*, encoded
  into the receiver's representation. An `array` or `range` argument raises.
- **The general family** — `array` and `range` (future homogeneous vector types — `ints`, `decimals` — will
  join it). To an `array`, another `array` or a `range` is a run; anything else, the triple included, is one
  element.

### The map family

`dict` — a set of **keys**, each with an attached value; the key is the element. It shares the element-based
sequence members (`contains`, `count`, `index`, `any`, `all`, `remove`, `filter`, `map`, `reduce`, `for_each`)
where they read on keys, plus `keys`, `values`, and `merge`. Position/order members (`first`, `sort`,
`index_last`, `slice`, …) do not exist on it, and members that would have to choose between the key axis and
the value axis are spelled through the axis explicitly: `d.keys().min()`, `d.values().sum()`.

`record` belongs here semantically but has **no member surface at all** — that is deliberate. Everything a
record needs is a free builtin with universal domain: `len(r)`, `copy(r)`/`freeze(r)` (and the `_shallow`
pair), `format(r)`, `is_true(r)`, `remove(r, key)`. Field access and assignment (`r.name`, `r.name = v`) are
the operator surface.

### Callables

Functions, closures, and builtins carry exactly the universal set: `copy`, `freeze`, `format`, `is_true` —
nothing else. `copy`/`freeze` are provable no-ops kept so generic code never type-errors; `format()` renders
the detail form:

```go
format(func(a, b) { return a })    // "<compiled-function/2>"
type_name(func() { return 1 })     // "function" — uniformly for all three kinds
```

Two predicates, two questions: `is_function(x)` is the **type** predicate (`"function"` is a type name, and
every type name has its `is_T`); `is_callable(x)` is the **capability** predicate — the unifier that also
answers `true` for host-defined callable types that keep their own type name.

## string, runes, bytes — the text role model

| type | role | mutability | indexing |
| --- | --- | --- | --- |
| `string` | Unicode text in compact UTF-8, cached symbol count | immutable | O(n) worst case |
| `runes` | Unicode text as materialized symbols | mutable | O(1) |
| `bytes` | raw octets: binary data, ASCII-range text | mutable | O(1) |

`string` and `runes` are **semantically interchangeable** — the same text, two storage strategies — and they
compare equal:

```go
"abc" == u"abc"       // true
```

The element of all text is the **symbol** (a `rune`): `len()` counts symbols, `s[i]` yields a `rune`, `s[i:j]`
slices at symbol boundaries, every member offset is symbol-based. Only `bytes` differs — its element is the
octet. Octet access on a string is `.bytes()`:

```go
s := "héllo"
s.len()              // 5 — symbols, not octets
s[1]                 // 'é'
s[1:3]               // "él"
s.bytes().len()      // 6 — 'é' is two octets in UTF-8
```

Symbols are code points, **not** grapheme clusters — a decomposed character is several symbols:

```go
"é".len()      // 2 — e + a combining accent renders as é but is two symbols
"é".len()            // 1 — the precomposed code point U+00E9
```

**Text that is not valid UTF-8 is data too**, so every conversion among `byte`/`bytes`/`string`/`runes` is
total and lossless: an octet no decoder can read as a symbol becomes its own reserved rune (`U+DC80`–`U+DCFF`)
and encodes straight back to that octet. Nothing raises, nothing becomes `U+FFFD`, and `is_valid()` is how a
script asks. See [string: Undecodable octets](types/string.md#undecodable-octets).

```go
b = bytes([0x61, 0xFF, 0x62])
b.string().bytes() == b     // true — read it, hold it as text, write it back unchanged
b.string().is_valid()       // false
b.string()[1].byte()        // byte(255) — and now it can be repaired
```

Choose `string` by default: it is correct-by-default (symbol-based everywhere, cached count) at the price of
O(n) worst-case positional work. Choose `runes` for index-heavy or mutating text work. Choose `bytes` for
binary data; it may operate on literal values, subsequences, and caller-supplied element sets, never on symbol
classes or case mappings — so `split`/`trim`/`replace`/`index` exist on `bytes`, while `upper`/`lower`/the
casing family require decoding first (`b.string().upper()`).

## One name, one operation — the argument's type selects the reading

Sequence members never encode the argument's role in the name (there is no `remove_slice`, no `index_any`).
One name per operation; the **argument's type** picks the reading:

| argument | reading |
| --- | --- |
| absent | the type's **blank set** (below) |
| a function | **predicate** — `f/1` gets the element, `f/2` gets `(locator, element)` |
| the receiver's own kind (own sub-family) | a contiguous **run** |
| anything else | one **element** |
| variadic | a homogeneous **set** on the match side; operands in order on the add side |

```go
"banana".remove('a')          // "bnn"      — element
"banana".remove("an")         // "ba"       — run
[1,2,3,2,3].remove([2,3])     // [1]        — run (own kind)
[1,2,3].contains(func(x) { return x > 2 })   // true — predicate
"a,b;c".split(',', ';')       // ["a", "b", "c"] — variadic set
"abc".contains('a', 'z')      // true — set: "∈ the set" replaces "== the argument"
```

A one-element sequence is the uniform "as one element" hatch: `[1,2].remove([2])` removes the run `[2]`, which
is the element `2`. Runs match **leftmost-longest, non-overlapping** — in a variadic run set where one run
prefixes another, the longest wins, so the set is order-independent:

```go
"abcd".remove("ab", "abc")    // "d"
```

A match set is homogeneous — all elements or all runs; mixing raises and names the mixture, and a function
among several arguments always raises. On the **add** side variadic means operands in order — mixing is the
point: `x.append("ab", 'c')` ≡ `x + "ab" + 'c'`.

Each member declares which readings it takes; an absent reading raises and names the ones it has. Notable
declarations: `push` is the element-only add (a sequence argument raises — that refusal is its purpose);
`any`/`all` refuse the run reading permanently; `trim` takes an element set, never a run (the run forms are
`remove_prefix`/`remove_suffix`); `replace(old, new)` takes element or run in both positions, never a set or a
predicate.

Argument **acceptance** is by safe, unambiguous conversion into the reading's target, and it can be
value-dependent — then the error names the range, not the type:

```go
bytes("ab").push(98)     // bytes([97, 98, 98]) — int reads as the element type
"abc" + b'a'             // "abca" — a byte is a symbol in ASCII range
"abc" + b'\xff'          // raises: an octet reads as one symbol only in [0x00, 0x7F]
"ab".repeat(2.0)         // "abab" — lossless numeric accepted
"ab".repeat(1.5)         // raises: must be a whole number — never silent truncation
```

On `range`, the run reading is **deferred, not approximated**: `(0..10).contains(2..4)` raises saying the
vectorised integer sequence type does not exist yet — it never silently materializes an `array`.

## The blank set

The no-argument form of the match members (`trim()`, `split()`, `count()`, `index()`, …) matches the type's
**blank set** — its notion of insignificant content, one notion projected into each element domain:

| receiver | blank set | canonical member (default pad fill) |
| --- | --- | --- |
| `string` / `runes` | NUL ∪ the Unicode White_Space class | the space `' '` |
| `bytes` | NUL ∪ ASCII whitespace (all the whitespace an octet can express) | the space octet |
| `array` | `undefined` ∪ each element's own zero (`0` **is** blank) | `undefined` |
| `range` | `{0}` | — |
| `dict` | excluded — keys are identities, never filler | — |

```go
"  a  b ".split()                        // ["a", "b"]
[undefined, 0, 1, 2, 0, undefined].trim()  // [1, 2]
[0, "", 5].index()                       // 2 — first significant element
"7".pad_start(3, '0')                    // "007"
"7".pad_end(3)                           // "7  " — default fill is the canonical member
[1].pad_end(3)                           // [1, undefined, undefined]
```

A script that means amounts, where `0` is significant, passes its own set: `amounts.trim(undefined)`.

## The universal surface

Four members exist on (nearly) everything; `record` reaches them through the free forms.

| member | domain | contract |
| --- | --- | --- |
| `is_true()` | every type | truthiness (see the table below); raises on an error state (NaN) |
| `copy()` / `freeze()` | every type except `undefined` | never type-errors in generic code; no-ops on scalars and callables, real deep operations on containers and `error` |
| `format([spec])` | every type | the render — total, callables included; f-strings use the same path |
| `copy_shallow()` / `freeze_shallow()` | `array` and `dict` only (+ `record` free forms) | one level deep — the only types whose elements can be containers |

`undefined` takes no `copy` or `freeze` — `copy(undefined)` raises — but predicates about it answer honestly
(`is_view(undefined)` is `false`, `is_immutable(undefined)` is `true`).

**Conversions** are `x.T([default])`; the free spelling `T(x)` is the same conversion — `"123".int()` ≡
`int("123")`, on `x`'s own type as much as any other — but the default slot is the member's alone
(`int("12x", -1)` raises `wrong_num_arguments`).
One failure mode everywhere — a valid `T` or a catchable raise, never a silent zero, `undefined`, or `false`;
the explicit default converts the miss into a value:

```go
"123".int()          // 123
int("12x")           // raises: cannot convert string to int
"12x".int(-1)        // -1
undefined.int(7)     // 7 — the maybe-missing form rescues absent data...
undefined.array(def)  // answers def ITSELF — a default is answered as-is, never copied
error("boom").int(0)  // raises — ...but never a program error, default or not
(1.9).int()          // 1 — in-range resolution loss is silent, toward zero
(256).byte()         // raises — range violations always raise
[1, 2, "x3"].string()  // raises — container conversions are element-wise, all-or-nothing
[72, 105].runes().string()  // "Hi" — decoding is conversion composition
```

`T()` with no argument is the zero value: `int()` is `0`, `range()` the empty range, `time()` the zero
instant. **A conversion to your own type still constructs**, in either spelling: `array`, `dict`, `record`,
`bytes` and `runes` answer a new, independent, **mutable** value — a shallow copy, exactly `x.copy_shallow()`
— so neither `array(a)` nor `a.array()` ever writes through to `a`, and `bytes(b"ab")` turns a constant
literal into a writable buffer. Elements are the values handed in (a frozen element stays frozen); `x.copy()`
is the deep spelling. A conversion **never** hands back shared storage: that is the [`_view`
constructors'](types/container-semantics.md#views) job and only theirs, and `is_view()` marks their results.
`string` and the scalars are immutable and have no identity, so the same question cannot be asked of them —
`"ab".string()` is the receiver.

A same-type conversion cannot fail, so its trailing default is unreachable; it is still **accepted** on every
type, so generic `x.T(fallback)` code does not break on the one receiver that already has type `T`:

```go
a = [1, 2]
b = a.array()        // a NEW array — a.copy_shallow(), not a
b.append_in_place(9)
a                    // [1, 2] — untouched
(5).int(0)           // 5 — the default is accepted and ignored, on every type alike
``` `.string()` converts *content* (`byte(65).string()` → `"A"`); where the content has no text form —
`dict`, `record`, callables — `.string()` is absent and the free `string(x)` raises too: `format()` is the
answer there. On `undefined` the conversion members exist but demand a default (`undefined.string("-")` →
`"-"`; without one it raises) — see [types/undefined.md](types/undefined.md).

## Members and operators correspond

Members and operators accept and read alike — same readings, same acceptance:

| member | operator |
| --- | --- |
| `x.append(y)` | `x + y` |
| `x.remove(y)` | `x - y` |
| `x.contains(y)` | `y in x` |
| `x.repeat(n)` | `x * n` — `array`/`string`/`runes`/`bytes` only; the operand is a count, and there is no reflected `n * x` |

Two cells where an operator exists **without** the member, both deliberate and both stated on the type's own
page: `dict + dict` / `record + record` is a **merge** (last one wins), not `append` — neither type has an
`append` member — and `k in r` works on a `record`, which has no `contains` member of its own (it borrows
`dict`'s key rules). Everywhere else the pair either both exist and agree, or both are absent: `range` has
`contains` and `in` and no `+`/`-`/`*`; `record` has neither `-` nor `remove` (field removal is the free
`remove(r, k)`).

```go
[9] + [1, 2, 3]          // [9, 1, 2, 3] — own kind: run
[9].append([1, 2, 3])    // [9, 1, 2, 3]
[1, 2] + "ab"            // [1, 2, "ab"] — not an array: element
[9] + (1..4)             // [9, range(1, 4)] — a range is an element too
[1, 2, 3] - [2, 3]       // [1] — removes every occurrence of the run, not set difference
"ell" in "hello"         // true
"a" in dict({a: 1})      // true — the key is the element
```

Operators take only the **value** readings — the predicate reading belongs to members. `y in x` with a
callable `y` raises outright (`(in) an operator operand is always a value`); `+` and `-` treat a callable as
an ordinary element value (`[1, 2] + func() { return 1 }` appends the function).

The **receiver — the left operand — decides the result type**:

```go
type_name(u"ab" + bytes("cd"))          // "runes"
type_name(u"ab".append(bytes("cd")))    // "runes"
```

A member may be wider than its operator exactly where the operator has no receiver context to resolve an
ambiguity: `bytes("ab").push(98)` works, `bytes("ab") + 98` raises (`65` as text is ambiguous — `"65"` or
`'A'`).

Through every operator except `==`/`!=` (which answer honestly) and `in` (membership is a question that raises on an absent container), `undefined` **propagates** and `error` **raises**:

```go
undefined + 1            // undefined
undefined < 1            // undefined
error("x") + 1           // raises: error + int
```

## Lambda callbacks

Callbacks dispatch on **arity**:

| callback | receives |
| --- | --- |
| `f/1` | the element (`dict`: the key) |
| `f/2` | `(index, element)` (`dict`: `(key, value)`) |
| `reduce` `f/2` | `(acc, element)` (`dict`: `(acc, key)`) |
| `reduce` `f/3` | `(acc, index, element)` (`dict`: `(acc, key, value)`) |

```go
["a", "b"].map(func(i, x) { return string(i) + x })      // ["0a", "1b"]
[1, 2, 3].reduce(0, func(acc, x) { return acc + x })     // 6
dict({a: 1, b: 2}).filter(func(k, v) { return v > 1 })   // dict({"b": 2})
dict({a: 1, b: 2}).map(func(k, v) { return v * 10 })     // dict({"a": 10, "b": 20}) — keys fixed
dict({a: 1, b: 2}).reduce(0, func(acc, k, v) { return acc + v })  // 3
```

`for_each(fn)` makes a **full pass**, ignores the callback's return value, and returns the receiver. The old
return-`false`-to-stop protocol is gone — a `false` from a forgotten `return` cannot be told from a control
signal. Early exit belongs to `for`/`break` or to a search member (`index(fn)`, `any(fn)`, `all(fn)`):

```go
total := [0]
[10, 20].for_each(func(x) { total[0] += x; return false })  // return value ignored
total[0]                                                     // 30 — both elements visited
```

`map` is strictly 1:1 and returns the receiver's type. On the text triple a callback answering a sequence or
`undefined` raises (the message names the alternative); `flat_map` is the map-then-concatenate form — it
splices sequence results and drops `undefined`. On `array`, a nested result simply nests.

## Mutability

Four types have mutable bodies — `array`, `bytes`, `runes`, `dict`; everything else is an immutable value.
Every member is non-mutating by default; where mutation is observable and worthwhile the member has an
`_in_place` twin that leaves the same content in the receiver as `y = x.m(...)` leaves in `y`, mutates the
receiver, and returns it (so mutators chain). On a frozen receiver an `_in_place` member raises an error of
kind `not_mutable`. `string` is the one sequence type with **no mutating member at all** — the single fact
that decides between `string` and `runes`. Freezing is `freeze()` (deep) / `freeze_shallow()` (header-only);
see [types/container-semantics.md](types/container-semantics.md) for copying, freezing, aliasing, and views.

## Truthiness

Truthiness is inequality with the type's own zero value: `x.is_true()` ⟺ `x != T()`. Derived, not enumerated —
spelled `is_true` in member and free form (the free `bool(x)` is the *conversion*, not truthiness).

| value | `is_true()` |
| --- | --- |
| `0`, `0.0`, `decimal(0)`, `byte(0)`, `rune(0)` | `false` |
| `""`, `u""`, empty `bytes`, `[]`, empty `dict`/`record`, `range()` | `false` |
| `time()` (the zero instant) | `false` |
| `false`, `undefined` | `false` |
| every `error` — payload truthy or not | `true` (an error without a payload is not an error) |
| `float("nan")` | **raises** — NaN is neither true nor false in a boolean context |

```go
error("").is_true()      // true — the single stated base case
float("nan").is_true()   // raises
```

NaN and ±Inf on `float` are storable, sortable (total order, NaN first, NaN reflexively equal to itself) and
comparable, but every arithmetic on them raises — so they are interrogable without triggering the raise:
`is_nan()` / `is_inf()` exist on every numeric type and answer constant `false` where the type cannot
represent the sentinel (`(5).is_nan()` is `false`; `decimal` has no infinities).

# bytes

Raw octets — binary data that can also be treated as ASCII-range text.

## Overview

`bytes` is the low-level member of the text family: a mutable sequence of **octets** with no symbol
interpretation. The element is the [`byte`](byte.md).

| type | representation | mutability | indexing |
| --- | --- | --- | --- |
| `bytes` | raw octets, no symbol interpretation | **mutable** | O(1) by octet |
| `string` | Unicode text in compact UTF-8 | immutable | O(n) by symbol — see [string](string.md) |
| `runes` | Unicode text as a symbol array | mutable | O(1) by symbol — see [runes](runes.md) |

The line between `bytes` and the symbol types is **encoding**. Members that need symbol classes — `upper`,
`lower`, `case_fold`, the casing family — do not exist on `bytes`: interpreting an octet as a cased letter
means assuming an encoding, so decode first (`.string()` / `.runes()`). Encoding-free **structural** work —
`split`, `trim`, `replace`, `index`, the pads, the prefixes — is fully available: those are element-set and
subsequence operations, well-defined on binary data.

```go
bytes("héllo").len()                // => 6   (octets — é is two)
bytes("héllo").upper()              // => raises: type bytes has no method upper
bytes("héllo").string().upper()     // => "HÉLLO"   (decode, then the symbol work)
```

Background for arithmetic on the elements: `byte` is the language's only **modular** type — its arithmetic
wraps mod 256 (`b'\xff' + b'\x01'` → `byte(0)`), where every other numeric type raises on overflow. Details
on the [byte](byte.md) page.

## Literals and construction

The literal form is `b"..."`, with the same escapes as `string` literals plus raw octet escapes:

```go
b"ab"               // => bytes([97, 98])
b"\x00\xff"         // => bytes([0, 255])
```

**A literal is an immutable constant.** Its type name is `immutable-bytes`, and writing into it raises;
take a `copy()` or use a constructor to get a mutable body:

```go
type_name(b"ab")            // => "immutable-bytes"
l := b"abc"
l[0] = b'X'                 // => raises: type immutable-bytes does not support assignment ...

m := bytes("abc")           // constructors build mutable values
m[0] = b'X'                 // m is now bytes([88, 98, 99])
```

`bytes(...)` is both the conversion (one argument) and the count constructor (two): `bytes(x, n)` is
`bytes(x)` repeated `n` times, so `bytes(b'\x00', n)` is the preallocation spelling:

```go
bytes("héllo")          // => bytes([104, 195, 169, 108, 108, 111])   (UTF-8 octets of the text)
bytes([104, 105])       // => bytes([104, 105])   (an array of in-range ints or bytes, element-wise)
bytes([b'h', b'i'])     // => bytes([104, 105])
bytes([300])            // => raises              (not an octet)
bytes(b'\x07')          // => bytes([7])          (a byte is one octet — any octet)
bytes('8')              // => bytes([56])         (a rune contributes its UTF-8 octets — not the digit 8)
bytes(5)                // => raises: cannot convert int to bytes: no conversion exists
bytes("ab", 2)          // => bytes([97, 98, 97, 98])   (the count form: bytes(x).repeat(n))
bytes(b'\x00', 3)       // => bytes([0, 0, 0])    (the preallocation: n explicit fill octets)
```

A bare int has no reading at all: `bytes` plays a double role — ASCII-range text and raw memory chunk — so
`bytes(5)` could mean the text `"5"`, the octet `5`, or five zero octets. Spell the one you mean:
`bytes("5")`, `bytes(b'\x05')`, or `bytes(b'\x00', 5)`.

Encoding is total — every text-ish value has UTF-8 octets — so `string`/`runes` → `bytes` never fails.
Non-text scalars have no direct octet reading: `(3).bytes()` does not exist — go through the text,
`(3).string().bytes()`.

## Indexing and slicing

All positions are octet positions. Negative indices count from the end; slices clamp. Slicing can split a
multi-octet symbol — `bytes` does not know or care:

```go
bytes("héllo")[1]           // => byte(195)   (the first octet of é)
bytes("abc")[-1]            // => byte(99)
bytes("héllo").slice(0, 3)  // => bytes([104, 195, 169])
```

Element writes (`m[i] = b'x'`) mutate the shared body and are visible through every alias; on a frozen or
literal value they raise.

## Operators

| operator | meaning |
| --- | --- |
| `+` | concatenation; the **receiver** (left operand) decides the result type |
| `-` | removes **every** occurrence of the right operand (leftmost, non-overlapping); acceptance equals `+`'s |
| `==` `!=` `<` `<=` `>` `>=` | content-based, octet by octet — compares across `string`/`runes`/`bytes` by text content |
| `in` | membership: `x in b` uses `contains`' value readings (element or run); raises on an unacceptable operand — never a silent `false` |

Operand acceptance — every accepted operand is text content, encoded into octets. Encoding is total, so
`bytes` accepts every text-ish operand unconditionally:

| right operand | reading |
| --- | --- |
| `bytes` | its octets, verbatim |
| `string` / `runes` | its UTF-8 octets |
| `rune` | its 1–4 UTF-8 octets |
| `byte` | that octet |
| `int` | **raises** — arithmetic keeps the numeric reading of `+`/`-`; the member forms (`push`, `append`) take an in-range int |
| `bool` / `float` / `decimal` | raises — ambiguous as text |
| `int` (`*` only) | a repeat **count** — `b * n` is exactly `b.repeat(n)`; there is no reflected `n * b` |
| `array` / `range` / `dict` / callables | raises — the text types accept only text content. `bytes("ab") + [1, 2]` → `[bytes([97, 98]), 1, 2]` is not an exception to that: `bytes` declines and the array answers `+` by prepending it (see [array](array.md)) |

```go
bytes("ab") + "cd"          // => bytes([97, 98, 99, 100])   (bytes receiver → bytes)
bytes("ab") + u"cd"         // => bytes([97, 98, 99, 100])
bytes("ab") + 'é'           // => bytes([97, 98, 195, 169])  (a rune contributes its UTF-8 octets)
bytes("ab") + b'\xff'       // => bytes([97, 98, 255])       (any octet — no ASCII limit on this side)
bytes("ab") + 98            // => raises: bytes + int
bytes("ab") + [1]           // => [bytes([97, 98]), 1]   (bytes declines; the array prepends it)
bytes("ab") - [1]           // => raises: bytes - array
"ab" + bytes("cd")          // => "abcd"    (string receiver → decodes the bytes)
"ab" + bytes([255])         // => "ab\xff"  (the octet is carried through as its escape)
b'a' + bytes("bc")          // => bytes([97, 98, 99])   (a scalar on the left takes the sequence's type)

bytes("abcabc") - "bc"      // => bytes([97, 97])       (every occurrence)
bytes("banana") - b'a'      // => bytes([98, 110, 110])
bytes("héllo") - 'é'        // => bytes([104, 108, 108, 111])   (removes the 2-octet run)
bytes("abc") - 98           // => raises: bytes - int

bytes("ab") == "ab"         // => true
bytes("ab") == u"ab"        // => true
bytes("ab") < bytes("b")    // => true

b'a' in bytes("ab")         // => true    (element)
"bc" in bytes("abc")        // => true    (run of octets)
'é' in bytes("héllo")       // => true    (the rune's 2-octet run)
97 in bytes("abc")          // => true    (an in-range int reads as one octet)
300 in bytes("abc")         // => raises  (not an octet — never a silent false)
```

A callable on the right of `in` raises: an operator operand is always a value — the predicate reading is
spelled `contains(f)` / `any(f)`.

## Argument dispatch — the argument's type selects the reading

Every member that matches or adds content reads its arguments from one menu:

| argument | reading |
| --- | --- |
| absent | the **blank set** (below) |
| function | predicate — `f(element)` with one parameter, `f(index, element)` with two |
| `string` / `runes` / `bytes` | a **run** — a contiguous octet subsequence (text contributes its UTF-8 octets) |
| `rune` / `byte` / `int` | one **element** where it fits: a byte always; an int in `0`–`255` (members only — operators refuse int); a rune only if its encoding is one octet (ASCII) |
| several arguments (match side) | a **set** — "∈ the set"; homogeneous: all elements or all runs, mixing raises |
| several arguments (add side: `append`/`prepend`) | operands in order — mixing freely is the point |

A multi-octet rune is still fine wherever a **run** is accepted (`append`, `contains`, `replace`, ...) — it
reads as its 1–4 octets; only the strictly one-element slots (`push`, `insert` items, pad fills, `map`
results) refuse it.

Run matching is leftmost and non-overlapping; in a variadic run set, the longest wins at a tie.

## Decoding is total

`bytes` is the type with nothing to validate — every octet is a valid octet, which is why it carries
`is_ascii()` but no `is_valid()`. Decoding *out* of it never fails either: an octet that is not a symbol
becomes its reserved escape in the text, and converting back returns the same octet.

```go
b = bytes([0x61, 0xFF, 0x62])
b.string().bytes() == b         // true
b.runes().bytes() == b          // true
b.is_ascii()                    // false
b.string().is_valid()           // false — ask the type that can answer it
```

See [string: Undecodable octets](string.md#undecodable-octets) for the model, and note the one boundary that
is **not** total: `json.encode` on text holding escapes raises, while `json.encode(b)` on the bytes
themselves is always fine (base64).

### Text predicates

`is_ascii()` — every octet is below `0x80`. There is deliberately **no `is_valid()`**: every octet is a valid
octet. The decode question is `b.string().is_valid()`, asked of the type that can answer it.

## The blank set

The text types share one notion of insignificant content — **NUL ∪ whitespace** — projected into each
type's element domain. For `bytes` that is NUL plus **ASCII whitespace**: the fixed octet set
`{0x00, ' ', '\t', '\n', '\r', '\v', '\f'}` — all the whitespace a single octet can express. Deciding
anything wider would require decoding, which octets never get: an NBSP's octets (`0xC2 0xA0`) are
**content** here, not blank. The default pad fill is the canonical member of the set, the space octet.

The no-argument form of a member reads through this set:

- `trim()` / `trim_start()` / `trim_end()` strip blank octets; `split()` separates on runs of them.
- The queries — `contains()`, `any()`, `count()`, `index()`, `index_last()`, `all()` — ask about
  **significant** (non-blank) octets.
- `remove()` and `filter()` drop the blank octets, keeping the significant ones.

```go
bytes("  hi \t ").trim()            // => bytes([104, 105])
bytes([0, 104, 0]).trim()           // => bytes([104])          (NUL is blank)
bytes("\u00a0x\u00a0").trim()       // => bytes([194, 160, 120, 194, 160])   (NBSP octets are content)
bytes("  a b \t c ").split()        // => [bytes([97]), bytes([98]), bytes([99])]
bytes("  ").contains()              // => false                 (no significant octet)
bytes(" a b ").count()              // => 2
bytes("  ab").index()               // => 2                     (first significant octet)
bytes(" a b ").remove()             // => bytes([97, 98])
bytes("42").pad_start(5)            // => bytes([32, 32, 32, 52, 50])   (default fill = the space octet)
```

## Members

### Adding: `append`, `prepend`, `push`, `push_first`, `insert`

`append`/`prepend` take whole operands in order — `x.append(a, b)` ≡ `x + a + b` — mixing runs and elements
freely; each text operand contributes its UTF-8 octets. `push`/`push_first` are the validating forms: **one
element per argument** — a sequence argument raises even at length 1, and a multi-octet rune raises too (it
does not fit one element). `insert(i, ...items)` is `push` at a position; as a positional **edit**, `i`
outside `[0, len]` raises (negative `i` counts from the end).

```go
bytes("ab").append("cd", 'é', b'\xff', u"z")  // => bytes([97, 98, 99, 100, 195, 169, 255, 122])
bytes("cd").prepend("ab", b'x')     // => bytes([97, 98, 120, 99, 100])
bytes("ab").push(98)                // => bytes([97, 98, 98])   (an in-range int is one octet)
bytes("ab").push(b'\xff')           // => bytes([97, 98, 255])
bytes("ab").push('c')               // => bytes([97, 98, 99])   (ASCII rune = one octet)
bytes("ab").push(256)               // => raises: (push) an int reads as one octet and must be in [0, 255]
bytes("ab").push('é')               // => raises: the value does not fit a single element (2 octets — use append)
bytes("ab").push("c")               // => raises  (a sequence never reads as one element here)
bytes("ad").insert(1, b'b', b'c')   // => bytes([97, 98, 99, 100])
bytes("ab").insert(3, b'x')         // => raises: (insert) 3 out of range [0, 2]
```

### Matching: `contains`, `any`, `all`, `count`

`contains` takes the full menu: element, run, homogeneous variadic set, predicate, or absent.
`contains(fn)` ≡ `any(fn)` and `contains()` ≡ `any()`. `any`/`all` take a value, a function, or nothing; a
run argument raises.

```go
bytes("héllo").contains("éll")      // => true    (its UTF-8 octet run)
bytes("abc").contains(b'x', b'b')   // => true    (set)
bytes("abc").contains(func(x) { return x > b'b' })   // => true
bytes("abc").any(func(x) { return x == b'c' })       // => true
bytes("abc").all(func(x) { return x >= b'a' })       // => true
bytes("banana").count(b'a')         // => 3
```

### Locating: `index`, `index_last`

The only two locators; answers are **octet positions**, directly usable with `[i]`, `[i:j]`, `slice()`. A
miss answers `undefined` — never `-1` — or the optional trailing default.

```go
bytes("héllo").index(b'l')          // => 3   (h=0, é=1..2, l=3)
bytes("héllo").index("llo")         // => 3
bytes("héllo").index_last(b'l')     // => 4
bytes("abc").index(b'z')            // => undefined
bytes("abc").index(b'z', -1)        // => -1  (only if you ask for it)
```

### Keeping and dropping: `filter`, `remove`

Both take the full menu and act on every occurrence; a miss is a silent no-op.

```go
bytes("banana").filter(b'a')        // => bytes([97, 97, 97])
bytes("banana").remove("an")        // => bytes([98, 97])
```

### Anchored: `has_prefix`, `has_suffix`, `remove_prefix`, `remove_suffix`

Element, run, or variadic run set for the tests; one exact run, removed once, for the removals (a miss
answers the receiver unchanged).

```go
bytes("foobar").has_prefix("foo")           // => true
bytes("foobar").has_prefix(b'f')            // => true
bytes("foobar").has_suffix("foo", "bar")    // => true    (set)
bytes("foobar").remove_prefix("foo")        // => bytes([98, 97, 114])
```

### Trimming: `trim`, `trim_start`, `trim_end`

A **set of elements**, stripped while they repeat at the edge; a run argument raises (the anchored run form
is `remove_prefix`/`remove_suffix`); no arguments = the blank set.

```go
bytes("xxhixx").trim(b'x')          // => bytes([104, 105])
bytes("xyhi").trim("xy")            // => raises: trim takes a set of elements, not a run
```

### Substitution: `replace`

Element or run in both positions; every occurrence, leftmost, non-overlapping.

```go
bytes("a-b-c").replace(b'-', " / ")     // => bytes([97, 32, 47, 32, 98, 32, 47, 32, 99])
```

### Padding: `pad_start`, `pad_end`

Width counts **octets**; the fill is exactly **one literal octet** (a run fill raises; so does a rune whose
encoding is more than one octet). Default fill is the space octet. A width at or below the length is a
no-op; one past `4294967296` octets raises rather than exhausting the host.

```go
bytes("42").pad_start(4, b'0')      // => bytes([48, 48, 52, 50])
bytes("é").pad_end(4, b'.')         // => bytes([195, 169, 46, 46])   (é already counts as 2)
bytes("x").pad_start(3, 'é')        // => raises  (the fill does not fit a single element)
bytes("x").pad_start(3, "ab")       // => raises  (fill is one element)
```

### Splitting: `split`, `partition`, `split_lines`

Separators from the dispatch menu: element, run, homogeneous variadic set, element-level predicate, or
absent (= the blank set). **Explicit separators keep empty pieces** (n hits → n+1 pieces); the no-arg blank
form drops empties; an empty-run separator matches nothing. No limit argument exists. All three answer an
`array` of `bytes`.

```go
bytes("a,,b").split(b',')           // => [bytes([97]), bytes([]), bytes([98])]   (empties kept)
bytes("a::b").split("::")           // => [bytes([97]), bytes([98])]
bytes("a,b;c").split(b',', b';')    // => [bytes([97]), bytes([98]), bytes([99])]
bytes("a,b").split(b',', "::")      // => raises   (mixed element + run set)
bytes("a1b2c").split(func(x) { return x >= b'0' && x <= b'9' })
                                    // => [bytes([97]), bytes([98]), bytes([99])]
bytes("ab").split("")               // => [bytes([97, 98])]   (an empty run matches nothing)
bytes("k=v=w").partition("=")       // => [bytes([107]), bytes([61]), bytes([118, 61, 119])]
bytes("a\nb\r\nc").split_lines()    // => [bytes([97]), bytes([98]), bytes([99])]
```

### Transforming: `map`, `flat_map`, `reduce`, `for_each`

`map` is strictly 1:1 and answers `bytes`: the callback must produce exactly **one element** — a byte, an
ASCII rune, or an int in `0`–`255`. A result outside the octet range raises (no silent wrap), and so does a
sequence or `undefined` result. `flat_map` splices run results and drops `undefined`. `for_each` makes a
full pass, ignores the callback's return value, and returns the receiver.

```go
bytes([1, 2, 3]).map(func(x) { return x.int() * 2 })    // => bytes([2, 4, 6])
bytes([200, 100]).map(func(x) { return x.int() * 2 })   // => raises: an int ... must be in [0, 255], got 400
bytes("ab").map(func(x) { return 'é' })                 // => raises  (2 octets do not fit one element)
bytes("ab").map(func(x) { return "xy" })                // => raises  (map is 1:1; use flat_map)
bytes("ab").flat_map(func(x) { return x == b'a' ? "aa" : undefined })   // => bytes([97, 97])
bytes("abc").reduce(0, func(acc, x) { return acc + x.int() })           // => 294  (the checksum spelling)
```

### Order and slices: `sort`, `reverse`, `dedup`, `unique`, `slice`, `slice_view`, `chunk`, `chunk_view`, `splice`, `repeat`

`slice(i, j)` **clamps**. `splice(start[, count, ...items])` is the positional edit: `start` outside
`[0, len]` raises, a negative `count` raises, the `count` clamps; the inserts take the add-side reading
(runs spread, elements insert). `repeat(n)` takes a whole-number count (`1.5` raises) and has the
operator form `b * n` (no reflected `n * b`). `dedup` collapses
adjacent duplicates; `unique` keeps the first of each. The `_view` forms share storage.

```go
bytes("cba").sort()             // => bytes([97, 98, 99])
bytes("aabbaa").dedup()         // => bytes([97, 98, 97])
bytes("aabbaa").unique()        // => bytes([97, 98])
bytes("abc").slice(1, 99)       // => bytes([98, 99])   (clamps)
bytes("abcd").splice(1, 2, "XY")    // => bytes([97, 88, 89, 100])
bytes("ab").splice(3, 0)        // => raises: (splice, start index) 3 out of range [0, 2]
bytes("abcde").chunk(2)         // => [bytes([97, 98]), bytes([99, 100]), bytes([101])]
bytes("ab").repeat(2)           // => bytes([97, 98, 97, 98])
bytes("ab").repeat(1.5)         // => raises   (never silent truncation)

sv := bytes("abcdef")
v := sv.slice_view(1, 3)        // shares storage
v[0] = b'X'                     // sv is now bytes([97, 88, 99, 100, 101, 102])
```

### Edges and extrema: `first`, `last`, `min`, `max`

All four take the optional trailing default; on an empty receiver they answer `undefined` (or the default).
No `sum`/`avg` — see the exclusions.

```go
bytes("bca").first()        // => byte(98)
bytes("bca").min()          // => byte(97)
bytes("").first(b'?')       // => byte(63)
```

### Conversions

Decoding is the partial direction: `bytes` → `string`/`runes` validates UTF-8 and raises on invalid input,
or answers the optional trailing default. `.array()` materialises the octets as `byte` elements — no
decoding.

```go
bytes("héllo").string()     // => "héllo"
bytes([255]).string()       // => "\xff"   (the octet's escape — never raises, never loses it)
bytes([255]).string("?")    // => "?"
bytes("hé").runes()         // => u"hé"
bytes("hi").array()         // => [byte(104), byte(105)]
bytes("ab").bytes()         // identity — the same value, not a copy
```

There is no `.int()`/`.float()`/`.decimal()`/`.bool()`/`.time()` on `bytes` — octets are not text until
decoded. Parse through the decode: `bytes("42").string().int()` → `42`. There are no `.dict()`/`.record()`
conversions either (octets are never key/value entries), and no `.byte()` (`bytes([65])` is a sequence;
its element is `b[0]`).

### Universal: `len`, `is_empty`, `format`, `copy`, `freeze`, `is_true`

```go
bytes("héllo").len()        // => 6   (octets)
bytes("hi").format("v")     // => "bytes([104, 105])"
```

`copy()` answers an independent mutable value; `freeze()` marks the value immutable (type name
`immutable-bytes`). `is_true()` is truthiness: inequality with the empty `bytes()`.

## In-place twins

`bytes` is a mutable-body type, so every eligible transform has an `_in_place` twin that mutates the shared
body (visible through every alias), **returns the receiver**, and raises kind `not_mutable` on a frozen
receiver:

```text
append_in_place  prepend_in_place  push_in_place  push_first_in_place  insert_in_place
remove_in_place  filter_in_place
trim_in_place  trim_start_in_place  trim_end_in_place
remove_prefix_in_place  remove_suffix_in_place  replace_in_place
pad_start_in_place  pad_end_in_place
sort_in_place  reverse_in_place  dedup_in_place  unique_in_place  splice_in_place
```

```go
t := bytes("  hi  ")
alias := t
t.trim_in_place()               // => bytes([104, 105])  (returns the receiver)
// alias is now bytes([104, 105]) too

fz := bytes("abc").freeze()
fz.push_in_place(b'x')          // => raises: (push_in_place) type immutable-bytes is immutable  (kind "not_mutable")
fz.append(b'd')                 // => bytes([97, 98, 99, 100])   (the copying form works on a frozen receiver)
```

The unsuffixed name is always the safe, copying form. `slice` has no twin (`slice_view` names the saving);
`repeat` has none (its result is n × the receiver); `map`/`split`/`partition`/`chunk` have none.

## Exclusions

- **`upper` / `lower` / `case_fold` / `title_case` and the identifier casings** — symbol classes and case
  mappings assume an encoding, which octets never get; decode first: `.string().upper()`.
- **`sum` / `avg`** — elements are octets, not numbers; the checksum spelling is
  `b.reduce(0, func(acc, x) { return acc + x.int() })`.
- **`join`** — the collection-as-receiver render lives on `array`/`range`; spell it
  `b.array().join(",")` if you want an octet listing.
- **Scalar conversions (`.int()`, `.float()`, `.decimal()`, `.bool()`, `.time()`)** — octets are not text
  until decoded; `string` is the gateway: `.string().int()`.
- **`.dict()` / `.record()`** — octets are never key/value entries.
- **`.byte()`** — wrapping/unwrapping is not conversion; the element access is `b[0]`.
- **`fields`** — no-arg `split()` *is* the whitespace splitter.
- **`copy_shallow` / `freeze_shallow`** — elements are scalars; there is no second level to stop at.

## Migration notes

Breaking changes from the previous surface, before → after:

- **`split` lost its limit argument.** `b.split(",", 2)` now raises (a run separator mixed with an element
  separator); beware `b.split(b',', 2)` — with an element separator it is a *silent* change, splitting on
  the comma **and** on octet 2. Split fully, then slice the pieces.
- **`trim` takes an element set, never a run.** `b.trim("xy")` used to strip a cutset given as text; it now
  raises — spell the set `b.trim(b'x', b'y')`. The anchored run form is `remove_prefix`/`remove_suffix`
  (which replace the old `trim_prefix`/`trim_suffix` names).
- **A locator miss answers `undefined`, never `-1`.** `bytes("abc").index(b'z')` → `undefined`; ask for a
  sentinel explicitly with `index(b'z', -1)`.
- **`map` answers `bytes` and validates.** It returned a silent array of ints before; now the callback must
  produce exactly one octet, and a result outside `0`–`255` — or a sequence, or `undefined` — raises.
- **`sum`/`avg` removed** — they widened octets to `int`, a second arithmetic model; write the `reduce`
  checksum above.
- **`splice_in_place` returns the receiver**, not the deleted run; take `b.slice(i, j)` beforehand if you
  need what was removed.
- **`for_each` makes a full pass** — a `false` return no longer breaks the loop; use `for` + `break` or a
  search member.
- **The sizing constructor is gone.** `bytes(3)` used to build three zero octets; it now raises — an int is
  ambiguous on a type that is both text and memory. The preallocation is `bytes(b'\x00', 3)`, with the fill
  explicit; the count form `bytes(x, n)` repeats any content the same way.
- **Literals are immutable constants.** `b"..."` answers `immutable-bytes`; take `.copy()` before writing
  into it.

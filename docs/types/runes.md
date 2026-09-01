# runes

Unicode text as a materialized symbol array — the mutable, O(1)-indexing sibling of `string`.

## Overview

`runes` and `string` hold the same thing: Unicode text, a sequence of **runes** (code points). They are
semantically interchangeable — the same members, the same operators, the same results. They differ in
representation and mutability:

| type | representation | mutability | indexing |
| --- | --- | --- | --- |
| `string` | compact UTF-8 | immutable | O(n) by symbol |
| `runes` | one machine word per symbol | **mutable** | **O(1)** by symbol |
| `bytes` | raw octets, no symbol interpretation | mutable | O(1) by octet — see [bytes](bytes.md) |

Choose `runes` for heavy positional work (indexing/slicing in loops) or for in-place mutation; choose
`string` everywhere else. The element is the [`rune`](rune.md); every offset a member answers or accepts is
a **symbol position**, never an octet position.

Symbols are **code points, not grapheme clusters**. A combining sequence counts as several symbols — what a
reader sees as one character may occupy several positions:

```go
u"héllo".len()      // => 5
u"e\u0301".len()    // => 2  (e + combining acute: two code points, one on-screen glyph)
```

## Literals and construction

The literal form is `u"..."`, with the same escape sequences as `string` literals (`\n`, `\t`, `\u00a0`,
`\U0001F680`, ...):

```go
u"привіт"           // Cyrillic
u"🚀"               // any code point
```

**`runes` is a mutable sequence — but a literal is an immutable constant.** The *type* is writable; `u"..."`
is a compile-time constant of that type. Its content is fixed in the source, so the compiler stores it once
in the bytecode's static pool and every evaluation loads that one shared value — a value that could be
written into would stop being a constant, since `l[0] = 'X'` would rewrite the program's own literal for
every later evaluation. Hence `immutable-runes`, and hence a raise on assignment. It also fits how the type
is used: `runes` is most often constant text, which is what the short `u"..."` form is for; when you want a
mutable buffer you construct one:

```go
type_name(u"ab")    // => "immutable-runes"
l := u"abc"
l[0] = 'X'          // => raises: type immutable-runes does not support assignment ...

a := runes(u"abc")  // a constructor always builds a new, writable body
a[0] = 'X'          // => u"Xbc"   (the literal is untouched)
a := u"abc".copy()  // the same thing, spelled as a copy
```

`runes(x)` is the conversion constructor — the free spelling of `x.runes()`, one meaning, two forms. It
takes anything with a text reading; a numeric argument converts to its **text**, never a buffer size. The
two-argument `runes(x, n)` is the count form — `runes(x).repeat(n)`:

```go
runes("héllo")                  // => u"héllo"      (mutable)
runes(42)                       // => u"42"         (the number's text — NOT a 42-symbol buffer)
runes("ab", 2)                  // => u"abab"       (the count form: runes(x).repeat(n))
runes('x', 3)                   // => u"xxx"
(42).runes()                    // => u"42"
true.runes()                    // => u"true"
bytes([104, 105]).runes()       // => u"hi"         (UTF-8 decode)
bytes([255]).runes()            // => u"\xff"      (the octet's escape — never raises, never loses it)
bytes([255]).runes(u"?")        // => u"?"          (trailing default rescues the miss)
```

Decoding is the partial direction: `bytes` → `runes` validates UTF-8 and raises on invalid input (or
answers the optional trailing default). Encoding is total: anything text-ish becomes `runes` unconditionally.

## Indexing and slicing

All positions are symbol positions. Negative indices count from the end; slices clamp; three-part slices
take a step:

```go
u"abc"[-1]          // => 'c'
u"abcdef"[1:5:2]    // => u"bd"
u"abc"[::-1]        // => u"cba"
```

Element writes (`s[i] = 'x'`) mutate the shared body and are visible through every alias; on a frozen or
literal value they raise.

## Operators

| operator | meaning |
| --- | --- |
| `+` | concatenation; the **receiver** (left operand) decides the result type |
| `-` | removes **every** occurrence of the right operand (leftmost, non-overlapping); acceptance equals `+`'s |
| `*` | exactly `repeat(n)`: the right operand is a **count**, not content. No reflected direction — `n * u` raises |
| `==` `!=` `<` `<=` `>` `>=` | content-based, symbol by symbol — compares across `string`/`runes`/`bytes` by text content |
| `in` | membership: `x in r` uses `contains`' value readings (element or run); raises on an unacceptable operand — never a silent `false` |

Operand acceptance — every accepted operand is text content, decoded into symbols:

| right operand | reading |
| --- | --- |
| `runes` / `string` | its text, verbatim |
| `bytes` | UTF-8 decode; **total** — an octet that is not a symbol becomes its [escape](string.md#undecodable-octets) |
| `rune` | one symbol |
| `byte` | one symbol **only in ASCII** (`0x00`–`0x7F`); above that raises |
| `int` | **raises** — arithmetic keeps the numeric reading of `+`/`-`; the member forms (`push`, `append`) take an int |
| `bool` / `float` / `decimal` | raises — ambiguous as text |
| `int` (`*` only) | a repeat **count** — `u * n` is exactly `u.repeat(n)`; there is no reflected `n * u` |
| `array` / `range` / `dict` / callables | raises — the text types accept only text content. `u"ab" + [1, 2]` → `[u"ab", 1, 2]` is not an exception to that: `runes` declines and the array answers `+` by prepending it (see [array](array.md)) |

```go
u"ab" + "cd"            // => u"abcd"        (runes receiver → runes)
u"ab" + bytes("cd")     // => u"abcd"        (decodes)
u"ab" + bytes([255])    // => u"ab\xff"     (the octet is carried through as its escape)
"ab" + u"cd"            // => "abcd"         (string receiver → string)
'a' + u"bc"             // => u"abc"         (a scalar on the left takes the sequence's type)
u"ab" + 'é'             // => u"abé"
u"ab" + b'A'            // => u"abA"         (ASCII byte)
u"ab" + b'\xff'         // => raises         (an octet reads as a symbol only in ASCII)
u"ab" + 99              // => raises: immutable-runes + int
u"ab" + [1, 2]          // => [u"ab", 1, 2]  (runes declines; the array prepends it as one element)
u"ab" - [1, 2]          // => raises: immutable-runes - array

u"abcabc" - "bc"        // => u"aa"          (every occurrence)
u"banana" - 'a'         // => u"bnn"
u"aaa" - "aa"           // => u"a"           (leftmost, non-overlapping)
u"abc" - "zz"           // => u"abc"         (a miss is a no-op)
u"abcb" - bytes("b")    // => u"ac"

u"ab" * 3               // => u"ababab"   (the count form of repeat)
3 * u"ab"               // => raises: int * immutable-runes   (no reflected direction)

u"ab" == "ab"           // => true
u"ab" == bytes("ab")    // => true
u"ab" < "b"             // => true

'b' in u"abc"           // => true           (element)
"bc" in u"abc"          // => true           (run)
98 in u"abc"            // => true           (98 is 'b'; a valid code point reads as one symbol)
999999999 in u"abc"     // => raises         (not a code point — never a silent false)
true in u"abc"          // => raises         (not text content)
```

A callable on the right of `in` raises: an operator operand is always a value — the predicate reading is
spelled `contains(f)` / `any(f)`.

## Argument dispatch — the argument's type selects the reading

Every member that matches or adds content reads its arguments from one menu:

| argument | reading |
| --- | --- |
| absent | the **blank set** (below) |
| function | predicate — `f(element)` with one parameter, `f(index, element)` with two |
| `string` / `runes` / `bytes` | a **run** — a contiguous subsequence (bytes decode totally; see [escapes](string.md#undecodable-octets)) |
| `rune` / `byte` / `int` | one **element** (byte in ASCII only; int must be a valid code point — members only, operators refuse int) |
| several arguments (match side) | a **set** — "∈ the set" instead of "== the argument"; the set is homogeneous: all elements or all runs, mixing raises |
| several arguments (add side: `append`/`prepend`) | operands in order — mixing freely is the point |

Run matching is leftmost and non-overlapping; in a variadic run set where one run is a prefix of another,
the longest wins:

```go
u"abcd".remove("ab", "abc")     // => u"d"    (the longer "abc" wins at the tie)
u"abc".contains('x', "bc")      // => raises  (mixed element + run set)
```

## Undecodable octets

`runes` holds the same text `string` does, so it holds undecodable octets the same way: each becomes its
reserved escape rune (U+DC80–U+DCFF), `is_valid()` finds them, and `.bytes()` returns the original octets.
See [string: Undecodable octets](string.md#undecodable-octets) for the full model.

```go
u = bytes([0x61, 0xFF, 0x62]).runes()
u.len()                         // 3
u[1].int()                      // 56575 — the escape for 0xFF
u.is_valid()                    // false
u[1].byte()                     // byte(255)
u.bytes() == bytes([0x61, 0xFF, 0x62])   // true
```

### Text predicates

`is_valid()` — every element is a real symbol, with no [octet escapes](string.md#undecodable-octets).
`is_ascii()` — every element is below `U+0080`.

## The blank set

The text types share one notion of insignificant content — **NUL ∪ whitespace** — projected into each
type's element domain. For `runes` that is NUL plus the full **Unicode White_Space class** (NBSP, ideographic
space, ... included). The default pad fill is the canonical member of the same set: the space `' '`.

The no-argument form of a member reads through this set:

- `trim()` / `trim_start()` / `trim_end()` strip blank elements; `split()` separates on runs of them.
- The queries — `contains()`, `any()`, `count()`, `index()`, `index_last()`, `all()` — ask about
  **significant** (non-blank) elements.
- `keep()` keeps the significant elements; `remove()` removes the blank ones — two actions, same answer.

```go
u"   hi \t ".trim()         // => u"hi"
u"\u00a0x\u00a0".trim()     // => u"x"      (NBSP is Unicode whitespace — blank here, content on bytes)
u"  a b \t c ".split()      // => [u"a", u"b", u"c"]
u"a b".contains()           // => true      (has a significant symbol)
u"  ".contains()            // => false
u" a b ".count()            // => 2         (significant symbols)
u"a b".all()                // => false     (the space is blank)
u"  ab".index()             // => 2         (first significant symbol)
u" a b ".remove()           // => u"ab"
u"42".pad_start(5)          // => u"   42"  (default fill = the space)
```

## Members

### Adding: `append`, `prepend`, `push`, `push_first`, `insert`

`append`/`prepend` take whole operands in order — `x.append(a, b)` ≡ `x + a + b` — mixing runs and elements
freely. `push`/`push_first` are the validating forms: **one element per argument**; a sequence argument
raises even at length 1 — the refusal is the member's purpose (no silent flattening). `insert(i, ...items)`
is `push` at a position; as a positional **edit**, `i` outside `[0, len]` raises (negative `i` counts from
the end).

```go
u"ab".append("cd", 'x', bytes("yz"))    // => u"abcdxyz"
u"cd".prepend("ab", 'x')                // => u"abxcd"   (argument order kept at the front)
u"ab".push('c')                         // => u"abc"
u"ab".push("c")                         // => raises     (a sequence never reads as one element here)
u"ab".push(98)                          // => u"abb"     (an int is a code point on the member side)
u"ab".push(55296)                       // => raises     (a surrogate is not a valid code point)
u"ab".push(b'\xff')                     // => raises     (an octet is a symbol only in ASCII)
u"ab".insert(1, 'x')                    // => u"axb"
u"ab".insert(3, 'x')                    // => raises: (insert) 3 out of range [0, 2]
```

### Matching: `contains`, `any`, `all`, `count`

`contains` takes the full menu: element, run, homogeneous variadic set, predicate, or absent.
`contains(fn)` ≡ `any(fn)` and `contains()` ≡ `any()` — the two named synonyms. `any`/`all` take a value, a
function, or nothing; a run argument raises (there is no universal reading for "all of a run").

```go
u"héllo".contains("éll")                        // => true
u"abc".contains('x', 'b')                       // => true   (set: ∈ {'x','b'})
u"abcd".contains("xy", "cd")                    // => true   (run set)
u"abc".contains(func(r) { return r > 'b' })     // => true
u"abc".any(func(r) { return r == 'c' })         // => true
u"abc".all(func(r) { return r >= 'a' })         // => true
u"abc".any("bc")                                // => raises (run: that query is contains's)
u"banana".count('a')                            // => 3
u"banana".count("an")                           // => 2      (non-overlapping)
u"banana".count('a', 'b')                       // => 4      (set)
```

### Locating: `index`, `index_last`

The only two locators. Element, run, predicate, or absent; a miss answers `undefined` — never `-1` — or the
optional trailing default. Deliberately singular: `index` can miss, so its second argument is the default, and
there is no variadic set form.

```go
u"héllo".index('l')             // => 2          (symbol position)
u"héllo".index("llo")           // => 2
u"abc".index(func(r) { return r > 'a' })    // => 1
u"abc".index('z')               // => undefined
u"abc".index('z', -1)           // => -1         (only if you ask for it)
u"héllo".index_last('l')        // => 3
```

### Keeping and dropping: `keep`, `remove`

Both take the full menu and act on **every** occurrence, in the direction their names give: `keep(x)` answers
what matches, `remove(x)` answers what does not. With no argument each reads against the blank set as its own
verb implies: `keep()` keeps the significant symbols, `remove()` removes the blank ones — the same answer by
two different actions. A miss is a silent no-op.

```go
u"banana".keep('a')                           // => u"aaa"
u"banana".remove("an")                          // => u"ba"
u"banana".remove('a', 'b')                      // => u"nn"
u"abcdef".keep(func(i, r) { return i % 2 == 0 })   // => u"ace"  (two-parameter form: (index, element))
```

### Anchored: `has_prefix`, `has_suffix`, `remove_prefix`, `remove_suffix`

The prefix/suffix tests take an element, a run, or a variadic run set — no predicate, no absent form
("first symbol satisfies f" is `index(f) == 0`). The removals take one exact run, remove it **once**, and
answer the receiver unchanged on a miss.

```go
u"foobar".has_prefix("foo")             // => true
u"foobar".has_prefix('f')               // => true
u"foobar".has_prefix("bar", "foo")      // => true       (set)
u"foobar".remove_prefix("foo")          // => u"bar"
u"foobar".remove_prefix("bar")          // => u"foobar"  (miss → unchanged)
u"foobar".remove_suffix("bar")          // => u"foo"
u"abcd".remove_prefix("ab", "abc")      // => u"d"       (longest run wins the tie)
```

### Trimming: `trim`, `trim_start`, `trim_end`

The trim family takes a **set of elements** and strips while members of the set repeat at the edge. A run
argument raises — the anchored run form is `remove_prefix`/`remove_suffix`. No arguments = the blank set.

```go
u"xxhixx".trim('x')             // => u"hi"
u"xyhixy".trim('x', 'y')        // => u"hi"
u"xyhixy".trim("xy")            // => raises: trim takes a set of elements, not a run
u"  hi  ".trim_start()          // => u"hi  "
u"  hi  ".trim_end()            // => u"  hi"
```

### Substitution: `replace`

Element or run in both positions; every occurrence, leftmost, non-overlapping. Never variadic (the second
position is the replacement), never a predicate.

```go
u"a-b-c".replace("-", " / ")    // => u"a / b / c"
u"a-b-c".replace('-', '+')      // => u"a+b+c"
```

### Padding: `pad_start`, `pad_end`

Width in **symbols**; the fill is exactly **one element** (a run fill would hide a truncation rule — build
the run and append it instead). Default fill is the space. A width at or below the length is a no-op; one
past `4294967296` symbols raises rather than exhausting the host.

```go
u"42".pad_start(5)              // => u"   42"
u"42".pad_start(5, '0')         // => u"00042"
u"ab".pad_end(4, '.')           // => u"ab.."
u"abcdef".pad_start(3)          // => u"abcdef"  (no-op)
u"ab".pad_start(5, "ab")        // => raises     (fill is one element)
```

### Splitting: `split`, `partition`, `split_lines`

Separators come from the dispatch menu: element, run, homogeneous variadic set, element-level predicate, or
absent (= the blank set). **Explicit separators keep empty pieces** — n hits produce n+1 pieces; the no-arg
blank form separates on runs of whitespace and drops empties. An empty-run separator matches nothing. There
is **no limit argument** (see the migration notes). `partition` splits at the **first** hit into
`[before, separator, after]`. All three answer an `array` of `runes`.

```go
u"a,,b".split(',')              // => [u"a", u"", u"b"]   (empties kept)
u"a::b::c".split("::")          // => [u"a", u"b", u"c"]
u"a,b;c".split(',', ';')        // => [u"a", u"b", u"c"]  (element set)
u"a,b;c".split(',', "::")       // => raises              (mixed element + run set)
u"a1b2c".split(func(r) { return r >= '0' && r <= '9' })  // => [u"a", u"b", u"c"]
u"  a b \t c ".split()          // => [u"a", u"b", u"c"]  (blank separators, empties dropped)
u"ab".split("")                 // => [u"ab"]             (an empty run matches nothing)
u"key=value=x".partition("=")   // => [u"key", u"=", u"value=x"]
u"a\nb\r\nc".split_lines()      // => [u"a", u"b", u"c"]
u"a\n\nb".split_lines()         // => [u"a", u"", u"b"]   (interior empty lines kept)
```

### Transforming: `map`, `flat_map`, `reduce`, `for_each`

`map` is strictly 1:1 and answers `runes`: the callback must produce exactly **one element** (a rune, or an
int that is a valid code point); a sequence or `undefined` result raises. `flat_map` is map-then-concatenate:
a run result splices in, `undefined` is dropped. `reduce(init, fn)` folds with the accumulator first.
`for_each` makes a **full pass** — the callback's return value is ignored (early exit is `break`'s job in a
`for` statement) — and returns the receiver, so it chains.

```go
u"abc".map(func(r) { return r + 1 })                     // => u"bcd"
u"abc".map(func(i, r) { return i == 0 ? r : r + 1 })     // => u"acd"
u"abc".map(func(r) { return "xy" })                      // => raises (map is 1:1; use flat_map)
u"abc".map(func(r) { return undefined })                 // => raises (the dropping form is flat_map)
u"ab".flat_map(func(r) { return u"" + r + r })           // => u"aabb"
u"a-b".flat_map(func(r) { return r == '-' ? undefined : r })  // => u"ab"
u"abc".reduce(0, func(acc, r) { return acc + r.int() })  // => 294
u"ab".reduce(u"".copy(), func(acc, r) { return acc.append(r, r) })  // => u"aabb"
```

### Order and slices: `sort`, `reverse`, `dedup`, `unique`, `slice`, `slice_view`, `chunk`, `chunk_view`, `splice`, `repeat`

`slice(i, j)` **clamps** — reading past the end is harmless. `splice(start[, count, ...items])` is the
positional edit: a `start` outside `[0, len]` **raises**, a negative `count` raises, the `count` clamps to
what is available; omitted `count` means "to the end". Splice's inserts take the add-side reading — a
text-run argument spreads, one element stays one element. `repeat(n)` takes a whole-number count
(`2.0` converts, `1.5` raises) and has the operator form `u * n` (no reflected `n * u`). `dedup` collapses adjacent duplicates; `unique` keeps the first of each.
The `_view` forms share storage with the receiver — a write through the view is visible in the original.

```go
u"cba".sort()                   // => u"abc"
u"abc".reverse()                // => u"cba"
u"aabbaa".dedup()               // => u"aba"
u"aabbaa".unique()              // => u"ab"
u"abcdef".slice(1, 3)           // => u"bc"
u"abc".slice(1, 99)             // => u"bc"      (clamps)
u"abcdef".slice(-3, -1)         // => u"de"
u"abcd".splice(1, 2, "XY")      // => u"aXYd"
u"ad".splice(1, 0, "bc", 'x')   // => u"abcxd"   (runs spread, elements insert)
u"ab".splice(3, 0)              // => raises: (splice, start index) 3 out of range [0, 2]
u"abcd".splice(1)               // => u"a"       (no count: delete to the end)
u"abcde".chunk(2)               // => [u"ab", u"cd", u"e"]
u"ab".repeat(2)                 // => u"abab"
u"ab".repeat(1.5)               // => raises     (never silent truncation)

sv := u"abcdef".copy()
v := sv.slice_view(1, 3)        // shares storage
v[0] = 'X'                      // sv is now u"aXcdef"
```

### Edges and extrema: `first`, `last`, `min`, `max`

All four take the optional trailing default; on an empty receiver they answer `undefined` (or the default).
`min`/`max` compare by code point. No `sum`/`avg` — see the exclusions.

```go
u"abc".first()      // => 'a'
u"bca".min()        // => 'a'
u"".first('?')      // => '?'
u"".first()         // => undefined
```

### Casing: `upper`, `lower`, `case_fold`, `title_case`, `snake_case`, `kebab_case`, `camel_case`, `pascal_case`

These need symbol classes and case mappings, so they live on `runes`/`string` only — `bytes` must decode
first.

`case_fold()` is the canonical equality form: each fold orbit maps to its smallest **lowercase** member
(else its minimum). Compare with `a.case_fold() == b.case_fold()`; `.lower()` is not a substitute — they
differ in both directions (`ſ`/`s` vs `İ`/`i`).

```go
u"héllo".upper()                // => u"HÉLLO"
u"HÉLLO".lower()                // => u"héllo"
u"Hello".case_fold()            // => u"hello"
u"ſtraße".case_fold()           // => u"straße"   (long s folds to s)
u"ſtraße".case_fold() == u"Straße".case_fold()    // => true
u"ΣΊΣΥΦΟΣ".case_fold()          // => u"ςίςυφος"  (ς is the orbit's smallest lowercase member)
```

The identifier renderings (`snake_case`, `kebab_case`, `camel_case`, `pascal_case`) segment **fully**: on
written boundaries (runs of whitespace / `_` / `-`, discarded), on lower→upper transitions, and before the
last upper of an upper-run followed by lowers (`parseXMLFile` → `parse|XML|File`). The boundary set is
closed — digits, apostrophes, and periods stay inside their word. Word interiors are normalised to the
rendering's case.

```go
u"parseXMLFile".snake_case()        // => u"parse_xml_file"
u"parseXMLFile".kebab_case()        // => u"parse-xml-file"
u"parse XML file".camel_case()      // => u"parseXmlFile"
u"parse XML file".pascal_case()     // => u"ParseXmlFile"
u"Hello  World-Wide".snake_case()   // => u"hello_world_wide"
u"utf8 codec v2".snake_case()       // => u"utf8_codec_v2"    (digits stay inside)
```

`title_case` is the label rendering: it segments on the **written** boundaries only (each separator run
becomes a single space), uppercases each word's first symbol, and **preserves the interior** — a label keeps
the author's emphasis; case transitions never split a word.

```go
u"ATM fee".title_case()         // => u"ATM Fee"
u"iPhone".title_case()          // => u"IPhone"
u"hELLO world".title_case()     // => u"HELLO World"
u"per_diem rate".title_case()   // => u"Per Diem Rate"
```

### Conversions

Every conversion takes the optional trailing default: `x.T()` raises on failure, `x.T(d)` answers `d`.
The scalar conversions **parse the text**.

```go
u"héllo".string()       // => "héllo"      (re-encode — total)
u"hé".bytes()           // => bytes([104, 195, 169])   (UTF-8 encode — total)
u"ab".array()           // => ['a', 'b']   (an array of runes, not decoded further)
u"42".int()             // => 42
u"4x".int()             // => raises
u"4x".int(0)            // => 0
u"1.5".float()          // => 1.5
u"1.5".decimal()        // => 1.5d
u"yes".bool()           // => true         (accepts true/false, 1/0, t/f, yes/no, case-insensitive)
u"2024-01-02T03:04:05Z".time()   // => time("2024-01-02T03:04:05Z")  (RFC3339)
u"ab".runes()           // a NEW, independent, mutable copy — same as runes(r), not the receiver
```

There is no `.byte()` or `.rune()` on `runes`: text parses into the numeric domain only — write
`u"65".int().byte()`.

### Universal: `len`, `is_empty`, `format`, `copy`, `freeze`, `is_true`

```go
u"abc".len()        // => 3
u"".is_empty()      // => true
u"hi".format("v")   // => "u\"hi\""
```

`copy()` answers an independent mutable value; `freeze()` marks the value immutable (type name
`immutable-runes`). `is_true()` is truthiness: inequality with the empty `runes()`.

## In-place twins

`runes` is a mutable-body type, so every eligible transform has an `_in_place` twin that mutates the shared
body (visible through every alias), **returns the receiver** (so mutators chain, and
`y = x.m(args)` / `x.m_in_place(args)` leave the same content in `x`), and raises kind `not_mutable` on a
frozen receiver:

```text
append_in_place  prepend_in_place  push_in_place  push_first_in_place  insert_in_place
remove_in_place  keep_in_place
trim_in_place  trim_start_in_place  trim_end_in_place
remove_prefix_in_place  remove_suffix_in_place  replace_in_place
pad_start_in_place  pad_end_in_place
sort_in_place  reverse_in_place  dedup_in_place  unique_in_place  splice_in_place
```

```go
t := u"  hi  ".copy()
alias := t
t.trim_in_place()               // => u"hi"  (returns the receiver)
// alias is now u"hi" too

fz := u"abc".freeze()
fz.push_in_place('x')           // => raises: (push_in_place) type immutable-runes is immutable  (kind "not_mutable")
fz.append("d")                  // => u"abcd"  (the copying form works on a frozen receiver)
```

The unsuffixed name is always the safe, copying form. `slice` has no twin (`slice_view` already names the
saving); `repeat` has none (its result is n × the receiver, not the receiver reused); `map`/`split`/
`partition`/`chunk` have none (their result is not the receiver's own single value).

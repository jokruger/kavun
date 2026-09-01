# string

Immutable Unicode text.

## Overview

A `string` is a sequence of **symbols** (Unicode code points). The symbol is the element everywhere on the
type: `len()` counts symbols, `s[i]` yields a `rune`, `s[i:j]` slices at symbol boundaries, and every member
offset — `index()`, `slice()`, `splice()`, `insert()` — is symbol-based.

```go
"héllo".len()        // => 5 (symbols, not bytes)
"héllo"[1]           // => 'é' (a rune)
"héllo"[1:3]         // => "él"
"abc"[-1]            // => 'c' (negative indices count from the end)
"кавун".len()        // => 5
```

Symbols are code points, **not** grapheme clusters. A flag emoji is two regional-indicator symbols; a
combining sequence is a base symbol plus combining marks. What a reader perceives as one character can be
several symbols:

```go
"🇺🇦".len()          // => 2 (two regional indicators)
"é".len()            // => 2 when written as e + U+0301 (combining acute)
```

**Storage and cost.** A `string` is stored as compact UTF-8 with a cached symbol count and an ASCII fast
path: indexing and slicing are O(1) on pure-ASCII strings and O(n) worst case on multibyte content. For heavy
positional work on non-ASCII text, convert to [`runes`](runes.md) first — `runes` is the O(1)-indexing,
materialized form of the same text.

**Immutability.** A `string` is immutable by construction. It is the only sequence type with no mutating
member at all: no `_in_place` twins, and no `slice_view`/`chunk_view` (on an immutable value sharing is
unobservable, so `slice` *is* the view). Element assignment raises:

```go
s := "abc"
s[0] = 'x'           // raises: type string does not support assignment
```

A `string` literal is a compile-time constant as well: its content is fixed in the source, so the compiler
stores it once in the bytecode's static pool and every evaluation loads that one shared value. For `string`
this is invisible — the type has no mutating member either way — but it is the same mechanism that makes
`b"..."` and `u"..."` immutable *constants* of otherwise mutable types, and the reason all three short forms
exist: text written down in a program is usually constant text. See [Constant literals and constructed
literals](../language.md#constant-literals-and-constructed-literals); when you want a mutable buffer of the
same content, construct one — `runes("abc")` or `bytes("abc")`.

**`string` and `runes` are semantically interchangeable** — the same text, the same member surface, the same
operator behavior, and they compare equal on equal content (`"ab" == u"ab"` is `true`). The difference is
storage (UTF-8 vs. a symbol array) and mutability. Choose `string` by default; choose `runes` when you need
in-place mutation or guaranteed O(1) indexing.

## Literals

```go
s := "hello"
esc := "line1\nline2 \"quoted\" \u00a0"   // escape sequences
raw := r"C:\Users\Bob \d+"                 // raw string: backslashes literal
uni := u"кавун"                             // a runes literal — same text, runes type
```

The free `string(x)` constructor is the conversion — the free spelling of `x.string()`. The two-argument
`string(x, n)` is the count form: the conversion repeated `n` times, `string(x).repeat(n)`:

```go
string()             // => ""
string("ab", 2)      // => "abab" (the count form: string(x).repeat(n))
string('x', 3)       // => "xxx"
string(65, 2)        // => "6565" — the conversion first: string(65) is "65"
```

F-strings interpolate expressions and format specs into a `string`: `f"n={n:5d}"`. See
[f-strings](../f-strings.md) and the [format mini-language](../format-mini-language.md).

## Operators

| operator | meaning |
| --- | --- |
| `+` | concatenation of text content |
| `-` | remove every occurrence of the run (leftmost, non-overlapping); acceptance equals `+`'s |
| `*` | exactly `repeat(n)`: the right operand is a **count**, not text. No reflected direction — `n * s` raises |
| `==` `!=` | content equality; `string` and `runes` with the same text are equal |
| `<` `<=` `>` `>=` | lexicographic by symbol |
| `x in s` | membership — runs `contains`' value readings |
| `s[i]` | the symbol at `i`, as a `rune` |
| `s[i:j]` | symbol-boundary slice, a `string` |

Both operands must carry **text content**. The **receiver (left operand) decides the result type**; a scalar
on the left takes the sequence's type:

```go
"ab" + "cd"          // => "abcd"
"ab" + u"cd"         // => "abcd" (string — the left operand decides)
u"ab" + "cd"         // => runes  (runes on the left)
"ab" + bytes("cd")   // => "abcd" (the bytes decode; the decode never fails)
b'a' + "bc"          // => "abc"  (scalar left takes the sequence's type)
'é' + "bc"           // => "ébc"
"abc" + b'\xff'      // raises: a byte is a symbol only in ASCII (0x00–0x7F)
"ab" + 1             // raises: operators never take int — arithmetic keeps its reading
"ab" + 1.5           // raises
"ab" + true          // raises
"ab" + ["cd"]        // => ["ab", "cd"] (string declines; the array prepends it as one element)
"ab" - ["cd"]        // raises: string - array (there is no reflected form for removal)
```

`-` removes runs or single symbols, never set-difference:

```go
"banana" - "na"      // => "ba"   (every occurrence, leftmost non-overlapping)
"banana" - 'a'       // => "bnn"
"banana" - 97        // raises: operators never take int

"-" * 20             // => "--------------------"   (the count form of repeat)
"ab" * 2.0           // => "abab"   (a whole-valued count converts; 1.5 raises)
2 * "ab"             // raises: int * string   (no reflected direction)
```

`in` accepts exactly the values `contains` accepts and raises on anything else — a callable raises, because
an operator operand is always a value, never a predicate:

```go
'a' in "abc"                       // => true
"bc" in "abc"                      // => true
97 in "abc"                        // => true ('a' — members read a valid int as the symbol)
1.5 in "abc"                       // raises
(func(c) { return true }) in "abc" // raises — the predicate spelling is contains(f)
```

Comparisons are lexicographic by symbol, not by length:

```go
"abc" < "abd"        // => true
"b" > "aaaa"         // => true
```

## Argument dispatch

Every member and operator on `string` reads its arguments the same way. An argument is accepted iff it is
**text content encoded into the receiver's representation**:

| argument | reading |
| --- | --- |
| `string` / `runes` | text, verbatim — a contiguous **run** |
| `bytes` | decoded as UTF-8; **total** — an octet that is not a symbol becomes its [escape](#undecodable-octets) |
| `rune` | itself — one symbol |
| `byte` | one symbol iff ASCII (`0x00`–`0x7F`); beyond raises |
| `int` (members only) | one symbol iff a valid code point; **operators never take `int`** |
| `bool` / `float` / `decimal` | raise — a number as text is ambiguous (`65` could be `"65"` or `'A'`) |
| `int` (`*` only) | a repeat **count** — `s * n` is exactly `s.repeat(n)`; there is no reflected `n * s` |
| `array` / `range` / `dict` / callables / `undefined` | raise — the text family is closed. One exception is not `string`'s doing: `"ab" + [1, 2]` → `["ab", 1, 2]`, because a left operand that declines hands `+` to the array, which prepends it as one element (see [array](array.md)) |

The match members (`contains`, `count`, `index`, `index_last`, `filter`, `remove`, `any`, `all`, `split`, …)
share one argument menu:

- **no argument** — the [blank set](#the-blank-set);
- **a function** — a predicate: `f(element)` with one parameter, `f(index, element)` with two;
- **`string`/`runes`/`bytes`** — a contiguous run;
- **`rune`/`byte`/valid `int`** — one element;
- **variadic** — a homogeneous set: all elements or all runs. Mixing the two classes raises, and a function
  among several arguments raises.

```go
"abc".contains('x', 'b')                       // => true (element set)
"abc".count(func(i, c) { return i > 0 })       // => 2 (two-parameter predicate)
"abcd".remove("ab", "abc")                     // => "d" (run set; at a tie the longest run wins)
"banana".remove("na", 'b')                     // raises: mixed element and run classes
"abc".remove("a", func(c) { return true })     // raises: a function among several arguments
```

The one exception is the **add side** (`append`/`prepend`), where operands mix freely in order:
`x.append("ab", 'c')` ≡ `x + "ab" + 'c'`.

Run matching is leftmost and non-overlapping: `"aaa".count("aa")` is `1`.

## Members

Every transforming member returns a new `string`; partitioning members (`split`, `partition`, `split_lines`,
`chunk`) return an `array` of strings.

### Universal

`copy()`, `freeze()`, `format([spec])`, `is_true()`. On an immutable value `copy`/`freeze` are cheap
identity-like operations kept for generic code; `is_true()` is `s != ""`; `format` is the render:

```go
"hi".format("q")     // => "\"hi\"" (quoted render)
is_immutable("abc")  // => true (free predicate — always, for string)
```

### Text predicates

`is_valid()` — every symbol is a real symbol, i.e. the text is well-formed UTF-8 with no
[octet escapes](#undecodable-octets). `is_ascii()` — every octet is below `0x80`, which also means one octet
per symbol.

```go
"abc".is_valid()                      // => true
"abc".is_ascii()                      // => true
"abé".is_ascii()                      // => false
bytes([97, 255]).string().is_valid()  // => false
```

### Size and edges

`len()`, `is_empty()`, `first([d])`, `last([d])`, `min([d])`, `max([d])`. The edge members answer a `rune`;
on an empty string they answer `undefined`, or the optional trailing default:

```go
"abc".first()        // => 'a'
"".first()           // => undefined
"".first('?')        // => '?'
"banana".min()       // => 'a'
"banana".max()       // => 'n'
```

### Adding

- `append(...xs)` / `prepend(...xs)` — whole operands in order, mixing runs and elements freely:
  `x.append("ab", 'c')` ≡ `x + "ab" + 'c'`.
- `push(...items)` / `push_first(...items)` — **validating**: each argument is exactly one element; a
  sequence argument raises even at length 1. The refusal is the member's purpose — use it when appending a
  run would be a bug.
- `insert(i, ...items)` — one element per item at position `i`. An edit position out of `[0, len]` raises.

```go
"x".append("ab", 'c', 98)    // => "xabcb" (98 is 'b')
"ab".append(bytes("cd"))     // => "abcd"
"c".prepend("a", 'b')        // => "abc" (arguments in order, at the front)
"ab".push('c')               // => "abc"
"ab".push("c")               // raises: push takes one element, never a sequence
"ad".insert(1, 'b', 'c')     // => "abcd"
"ab".insert(2, 'c')          // => "abc" (position len is valid)
"ab".insert(3, 'c')          // raises: 3 out of range [0, 2]
```

### Searching and matching

`contains(x)`, `any(x)`, `all(x)`, `count(x)`, `index(x[, d])`, `index_last(x[, d])` — the full dispatch
menu, except `any`/`all` which take a value, a function, or nothing (the contiguous-run query belongs to
`contains`). A locator miss answers `undefined` — never `-1` — or the trailing default:

```go
"banana".contains("nan")     // => true
"banana".count("na")         // => 2
"banana".index("na")         // => 2
"banana".index_last("na")    // => 4
"banana".index("xy")         // => undefined (miss — never -1)
"banana".index("xy", -1)     // => -1 (explicit default)
"héllo".index('l')           // => 2 (symbol offset, usable with s[i])
"abc".index(func(c) { return c == 'b' })  // => 1
"  ab".index()               // => 2 (no argument: first significant symbol)
"abc".all(func(c) { return c >= 'a' })    // => true
"abc".any("ab")              // raises: "all/any of a run" has no reading — use contains
```

No-argument `contains()` ≡ `any()` asks "any significant content?" — `"   ".contains()` is `false`.

### Keeping and removing

`filter(x)` keeps matching symbols, `remove(x)` drops every occurrence; both take the full menu. With no
argument they are the same operation — keep significant content, drop the blank set:

```go
"a1b2".filter(func(c) { return c >= '0' && c <= '9' })   // => "12"
"banana".remove("na")        // => "ba"
" a b ".filter()             // => "ab"
" a b ".remove()             // => "ab" (the documented no-arg synonym of filter())
```

### Anchored and structural

- `has_prefix(x)` / `has_suffix(x)` — element, run, or a variadic run set. No predicate reading ("the first
  symbol satisfies `f`" is `index(f) == 0`).
- `remove_prefix(x)` / `remove_suffix(x)` — one exact run, removed once; a miss answers the receiver
  unchanged.
- `trim(...set)` / `trim_start(...set)` / `trim_end(...set)` — strip **a set of elements**, repeating while
  they match. A run argument raises — the anchored-run form is `remove_prefix`/`remove_suffix`. No argument
  trims the blank set.
- `replace(old, new)` — element or run in both positions; every occurrence; never variadic, never a
  predicate.
- `pad_start(n[, fill])` / `pad_end(n[, fill])` — pad to symbol width `n` with one fill element (default:
  the space). A run fill raises; a width at or below `len()` is a no-op, and one past `4294967296` symbols
  raises rather than exhausting the host.

```go
"foobar".has_prefix('f')            // => true
"foobar".has_prefix("x", "fo")      // => true (run set)
"foobar".remove_prefix("foo")       // => "bar"
"foobar".remove_prefix("bar")       // => "foobar" (miss: unchanged)
"file.txt".remove_suffix(".txt", ".md")  // => "file"
"  hi  ".trim()                     // => "hi"
"xxhixy".trim('x', 'y')             // => "hi"
"abhiab".trim("ab")                 // raises: trim takes elements — spell trim('a', 'b')
"banana".replace("na", "NO")        // => "baNONO"
"banana".replace('a', "oo")         // => "boonoonoo" (element out, run in)
"7".pad_start(3)                    // => "  7"
"7".pad_start(3, '0')               // => "007"
"7".pad_start(3, "ab")              // raises: one fill element, not a run
```

### Splitting

`split(...seps)`, `partition(...seps)`, `split_lines()`. Separators take the menu: element, run, homogeneous
set, element-level predicate, or absent (the blank set). **`split` has no limit argument.**

Two regimes, deliberately different:

- An **explicit separator** keeps empty pieces — n hits make n+1 pieces, so `"".split(",")` is `[""]`.
- The **no-argument form** yields the maximal runs of significant content — empties never appear, and
  splitting pure filler yields `[]`.
- An empty-run separator matches nothing: `"ab".split("")` is `["ab"]`.

```go
"a,b,,c".split(',')          // => ["a", "b", "", "c"] (empties kept)
"a::b::c".split("::")        // => ["a", "b", "c"]
"a,b;c".split(',', ';')      // => ["a", "b", "c"] (separator set)
"a1b2c".split(func(c) { return c >= '0' && c <= '9' })  // => ["a", "b", "c"]
"  a  b\tc ".split()         // => ["a", "b", "c"] (blank set, maximal runs)
"   ".split()                // => []
"".split(",")                // => [""]
"a\nb\r\nc\n".split_lines()  // => ["a", "b", "c"]
```

`partition` answers exactly three pieces, `[before, sep, after]`, splitting at the first hit; a miss answers
`[s, "", ""]`. With no argument the separator is the whole first maximal run of blank filler:

```go
"key=value=x".partition('=')     // => ["key", "=", "value=x"]
"abc".partition('=')             // => ["abc", "", ""]
"key  value rest".partition()    // => ["key", "  ", "value rest"]
```

### Order, slices, repetition

`sort()`, `reverse()`, `dedup()`, `unique()`, `slice(i, j)`, `chunk(n)`, `splice(start, delete_count,
...items)`, `repeat(n)`.

- `sort()` orders symbols by code point and takes no comparator.
- `dedup()` collapses adjacent duplicates; `unique()` keeps the first occurrence of each symbol.
- `slice` **clamps** out-of-range bounds (reading past the end is harmless); negative indices count from the
  end.
- `splice` **raises** out of range (editing past the end is not harmless). Its inserts take the add-side
  reading: runs spread, scalars are one element, anything else raises.
- `repeat(n)` self-concatenates; `n` must be a non-negative whole number (`2.0` converts, `1.5` raises).
  `s * n` is the operator form — same count slot, same errors. There is no reflected `n * s`, and a result
  past `4294967296` symbols raises rather than exhausting the host.

```go
"banana".sort()              // => "aaabnn"
"abc".reverse()              // => "cba"
"aabbaa".dedup()             // => "aba"
"aabbaa".unique()            // => "ab"
"hello".slice(1, 3)          // => "el"
"hello".slice(3, 99)         // => "lo" (clamps)
"hello".slice(-3, -1)        // => "ll"
"abcde".chunk(2)             // => ["ab", "cd", "e"]
"abcd".splice(1, 2, "XY", 'z')  // => "aXYzd"
"ad".splice(1, 0, 'b')       // => "abd"
"ab".splice(5, 1)            // raises: 5 out of range [0, 2]
"ab".repeat(3)               // => "ababab"
"ab" * 3                     // => "ababab"   (the operator form)
"-" * 40                     // => "----------------------------------------"
3 * "ab"                     // raises: int * string — no reflected direction
"ab".repeat(1.5)             // raises: whole number required — never silent truncation
```

### Transforming and iterating

- `map(fn)` is **strictly 1:1 and answers a `string`** — the callback must return exactly one symbol per
  element; a sequence or `undefined` result raises.
- `flat_map(fn)` is map-then-concatenate: a run result splices in, `undefined` is dropped.
- `reduce(init, fn)` folds; the callback is `f(acc, element)`.
- `for_each(fn)` makes a full pass — the callback's return value is ignored, early exit is `break`'s job in a
  `for` loop — and returns the receiver, so it chains.

```go
"abc".map(func(c) { return (c.int() + 1).rune() })       // => "bcd"
"abc".map(func(c) { return "xx" })   // raises: map is 1:1 — the concatenating form is flat_map
"a-b".flat_map(func(c) { return c == '-' ? " / " : c })  // => "a / b"
"a-b".flat_map(func(c) { return c == '-' ? undefined : c })  // => "ab" (undefined dropped)
"abc".reduce(0, func(acc, c) { return acc + c.int() })   // => 294
"abc".for_each(func(c) { fmt.println(c) })               // prints each symbol, returns "abc"
```

### Case

`lower()`, `upper()`, `case_fold()`, and the casing family `title_case()`, `snake_case()`, `kebab_case()`,
`camel_case()`, `pascal_case()`.

```go
"héllo".upper()      // => "HÉLLO"
"HÉLLO".lower()      // => "héllo"
```

`case_fold()` is the canonical fold: each fold orbit maps to its smallest lowercase member (else the smallest
member), so `a.case_fold() == b.case_fold()` is exact case-insensitive equality. Because it is a transform —
not a comparison predicate — the result composes: it works as a dict key, a sort basis, a dedup basis.
`.lower()` comparison is not a substitute; they differ in both directions:

```go
"Hello".case_fold()                                   // => "hello"
"ſtraße".case_fold() == "Straße".case_fold()          // => true  (long s ſ and S both fold to s)
"İstanbul".case_fold() == "istanbul".case_fold()      // => false (İ keeps its own identity, distinct from i)
```

The four **identifier renderings** segment the text fully and normalise each word's interior. Word boundaries
are: runs of whitespace/`_`/`-` (discarded); a lower→upper transition; and in an upper run followed by a
lower, the last upper starts the new word (`parseXMLFile` → `parse` | `XML` | `File`). The boundary set is
closed — digits, apostrophes, and periods stay inside words:

```go
"parseXMLFile".snake_case()      // => "parse_xml_file"
"Parse XML file".kebab_case()    // => "parse-xml-file"
"parse_xml_file".camel_case()    // => "parseXmlFile" (interior normalised)
"parse xml file".pascal_case()   // => "ParseXmlFile"
"don't stop".snake_case()        // => "don't_stop" (the apostrophe is not a boundary)
```

`title_case()` is the **label rendering**: it segments on written boundaries only (whitespace/`_`/`-`),
uppercases each word's first symbol, and **preserves** the interior — a label keeps the author's emphasis:

```go
"ATM fee".title_case()           // => "ATM Fee"
"iPhone".title_case()            // => "IPhone"
"hELLO world".title_case()       // => "HELLO World"
"atm_fee-total".title_case()     // => "Atm Fee Total"
```

### Conversions

`int([d])`, `float([d])`, `decimal([d])`, `bool([d])`, `time([d])` parse the text; `runes()`, `bytes()`,
`array()` re-represent it; `string()` answers the receiver — a string is immutable and identity-less, so
there is nothing to construct (the mutable-bodied types copy instead: see
[`array()`](array.md#conversions)). Every conversion follows the uniform failure policy:
a valid result or a catchable raise, and with the optional trailing default, the default on any error:

```go
"123".int()          // => 123
"12a".int()          // raises: cannot convert string to int
"12a".int(0)         // => 0
"1.5".float()        // => 1.5
"Yes".bool()         // => true (true/false, 1/0, t/f, yes/no — case-insensitive)
"maybe".bool()       // raises
"2026-08-29T00:00:00Z".time().year()   // => 2026 (RFC 3339)
"nope".time(time())  // => 0001-01-01T00:00:00Z (the default)
"abc".runes()        // => u"abc" (the O(1)-indexing form)
"héllo".bytes().len()  // => 6 (octets — the UTF-8 encoding)
"abé".array()        // => ['a', 'b', 'é'] (an array of runes)
```

## Undecodable octets

Text that is not valid UTF-8 is data too — a file off disk, a legacy export, a truncated frame. So **every
conversion among `byte`, `bytes`, `string` and `runes` is total and lossless**: nothing raises, nothing is
substituted, and the octets you started with are the octets you get back.

An octet no decoder can read as a symbol becomes its own reserved rune — **U+DC80–U+DCFF, one per octet
0x80–0xFF** — and encoding turns that rune straight back into the octet it came from.

```go
b = bytes([0x61, 0xFF, 0x62])   // 'a', a stray octet, 'b'
s = b.string()                  // no raise
s.len()                         // 3 — the octet is one symbol
s[1].int()                      // 56575  (U+DCFF, the escape for 0xFF)
s.bytes() == b                  // true
s.runes().bytes() == b          // true — through runes and back, unchanged
```

Asking, and repairing:

```go
s.is_valid()                    // false — some symbol is an escape
"abc".is_valid()                // true
s[1].is_valid()                 // false — this one
s[1].byte()                     // byte(255) — the escape converts back to its octet
s.replace(s[1], '?')            // "a?b"
s.filter(c => c.is_valid())     // "ab"
```

**Why this range, and why it cannot go wrong.** Only an octet ≥ 0x80 can ever be undecodable — an ASCII octet
is always a symbol — so 128 values suffice. Every undecodable octet decodes with width 1, so *n* bad octets
always become *n* escapes. And U+DC80–U+DCFF are low surrogates: never Unicode scalar values, so well-formed
text can never produce one and an escape can never be mistaken for content. (The scheme is Python's
`surrogateescape`, PEP 383; Rust's WTF-8 answers the same problem the same way.)

**The two places this is visible elsewhere.** A [`rune`](rune.md)'s domain is exactly what can be encoded —
scalar values plus the 128 escapes — so `rune(0xDCFF)` exists while `rune(0xD800)` raises where it is written,
and no conversion *out* of `rune`/`runes` can fail. And the boundary **out** of the language is not total:
`json.encode` raises on text holding escapes, because JSON text is UTF-8 by definition. Encode the `bytes`
instead (they go as base64), or repair the text first.

## The blank set

The no-argument form of the match members acts on the type's **blank set** — its notion of insignificant
content. For `string` that is **NUL ∪ Unicode whitespace** (the Unicode `White_Space` class): this is what
no-arg `trim()`, `split()`, `partition()`, `index()`, `count()`, `filter()`, and `remove()` act on, and the
default `pad_start`/`pad_end` fill is the set's canonical member, the space.

Being the Unicode class, it includes the non-breaking space and every other Unicode space — not just ASCII
whitespace:

```go
"\u00a0x".trim()     // => "x" (the NBSP is blank)
"\x00x\x00".trim()   // => "x" (NUL is blank)
```

`bytes` projects the same notion into its element domain (the ASCII subset — all the whitespace an octet can
express); the sets agree wherever the domains overlap.

## What string deliberately does not have

- **No mutating member.** `string` is immutable by construction — no `_in_place` twin exists for any verb.
  Mutating workflows use `runes` (same surface plus the twins) and convert back.
- **No `slice_view` / `chunk_view`.** On an immutable value sharing is unobservable, so `slice` *is* the
  view; a `_view` twin would name a distinction that cannot be seen.
- **No `sum()` / `avg()`.** Symbols are not numbers; summing them would smuggle in a second arithmetic model.
  Spell the intent: `"abc".reduce(0, func(acc, c) { return acc + c.int() })`.
- **No `byte()` / `rune()` conversions.** Text parses into the numeric domain only: `"65".int().byte()`,
  never `"65".byte()` — a direct edge would conflate parsing with encoding.
- **No `join()`.** A separator is not a collection; the receiver is the subject. Spell it
  `["a", "b"].join(",")`.
- **No `fields()`.** No-argument `split()` *is* the Unicode-whitespace splitter; a second name for it would
  be a near-duplicate.

## Migration notes

The redesign takes clean breaks — old spellings raise loudly rather than shifting meaning silently, with one
exception noted in (a).

**(a) `len()`, indexing, and slicing are symbol-based (were byte-based).** This is the one *silent* change on
this page — positions on multibyte text mean something different now:

```go
// before: "héllo".len() == 6; "héllo"[1] was a byte of the é encoding
"héllo".len()        // => 5 now
"héllo"[1]           // => 'é' now (a rune)
"héllo"[1:3]         // => "él" now
```

Octet access moved wholesale to `bytes`: `"héllo".bytes().len()` → `6`.

**(b) `split` lost its limit argument.**

```go
// before: "a,b,c".split(",", 2) => ["a", "b,c"]
"a,b,c".split(",", 2)    // raises now (2 reads as another separator — a homogeneous-set violation)
"a,b,c".split(",")       // => ["a", "b", "c"]
```

**(c) `trim` takes an element set, never a run.**

```go
// before: "abhiab".trim("ab") stripped the characters a and b
"abhiab".trim("ab")      // raises now
"abhiab".trim('a', 'b')  // => "hi" — the element-set spelling
"foobar".remove_prefix("foo")  // => "bar" — the anchored run form
```

**(d) `index` misses answer `undefined`, never `-1`.** A `-1` would silently read the tail through negative
indexing; `undefined` raises at the point of use.

```go
// before: "banana".index("xy") => -1
"banana".index("xy")     // => undefined now
"banana".index("xy", -1) // => -1 — opt back in explicitly
```

**(e) `map` answers a `string` and is strictly 1:1** (it answered an `array` and accepted anything):

```go
"abc".map(func(c) { return (c.int() + 1).rune() })  // => "bcd" (a string)
"abc".map(func(c) { return "xx" })                  // raises — the splicing form is flat_map
```

**(f) The no-argument forms act on NUL ∪ Unicode whitespace**, not ASCII space only: `"\u00a0x".trim()` now
trims the NBSP, and `split()` with no argument splits on every Unicode space.

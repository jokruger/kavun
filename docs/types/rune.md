# rune

One Unicode code point.

## Overview

`rune` is an ordinal scalar with text content: the element type of [`string`](string.md) and
[`runes`](runes.md). It holds a single code point (`U+0000`–`U+10FFFF`, excluding the surrogate range),
whatever its UTF-8 length — `'A'` and `'є'` are both one rune.

Its **domain** is exactly what can be encoded back to octets: a Unicode scalar value, **or** one of the 128
[octet escapes](string.md#undecodable-octets) (`U+DC80`–`U+DCFF`) standing for an octet that is not a symbol.
Everything else — a high surrogate, a negative, anything past `U+10FFFF` — raises where it is written, so no
conversion *out* of a `rune` can fail. `is_valid()` distinguishes the two: a real symbol, or an escape.

`rune` is ordinal, not numeric: it is comparable and supports offset arithmetic, but has no `*`/`/`, no
`sum`/`avg` participation, and **its overflow policy is raise** — an arithmetic result that leaves the
code-point space is an error, never a silent `U+FFFD`.

`rune` values are immutable.

## Literals and Construction

```go
a := 'A'           // U+0041
ye := 'є'          // U+0454 — any code point, directly
nl := '\n'         // escapes accepted
x := '\x41'        // 'A'
u := '\u0454'    // 'є'
top := '\U0010FFFF'
```

```go
rune()             // '\x00' — the zero value (NUL)
rune(65)           // 'A' — conversion from int
rune(1114112)      // raises — above U+10FFFF
rune(55296)        // raises — the surrogate range U+D800–U+DFFF is not a code point
(1114112).rune('?')  // '?' — the member's optional default rescues the failure
```

Promotion into a sequence: there is no `repeat()` on scalars — go through the one-symbol string:

```go
'a'.string().repeat(3)   // "aaa"
u"".pad_end(3, 'a')      // u"aaa"
```

## Operators

### Arithmetic — offsets raise on overflow

`rune + int` and `rune - int` move along the code-point axis and return a `rune`; a result outside
`U+0000`–`U+10FFFF`, or inside the surrogate range, **raises** — never a replacement character:

```go
'a' + 1              // 'b'
'a' - 1              // '`'
1 + 'a'              // 'b'
'\U0010FFFF' + 1     // raises: rune overflow: 1114112 is not a valid code point
'\uD7FF' + 1         // raises: rune overflow: 55296 is not a valid code point (surrogate)
'a' - 98             // raises: rune overflow: -1 is not a valid code point
```

`rune - rune` is the distance between two code points — a plain `int`, not a rune (`rune - byte` reads the
same way):

```go
'd' - 'a'            // 3
'a' - b'a'           // 0
```

| operator | with `int` | with `rune` | with `byte` | with `float` / `decimal` |
| --- | --- | --- | --- | --- |
| `+` | `rune` (guarded) | raises | raises | raises |
| `-` | `rune` (guarded) | `int` — the distance | `int` — the distance | raises |
| `*` `/` `%` | raises | raises | raises | raises |

There is no `rune + rune`: adding two code points has no meaning.

### Comparison

Comparisons order by code point and accept the numeric-valued scalars — `rune`, `byte`, `int`, `bool`
(as 0/1), `float`:

```go
'a' < 'b'            // true
'a' == 97            // true
97 == 'a'            // true
'a' == b'a'          // true
'a' < 97.5           // true
```

### As text content

Next to `string`/`runes` a rune is itself — one symbol; into `bytes` it is its UTF-8 octets (1–4 of them).
A scalar on the left takes the sequence's type:

```go
"ab" + 'c'             // "abc"
'a' + "bc"             // "abc" — scalar on the left, result is still a string
u"ab" + 'є'            // u"abє"
bytes("ab") + 'є'      // bytes([97, 98, 209, 148]) — the 2-octet UTF-8 encoding
"абв".contains('б')    // true — same acceptance as the operators
```

## Members

The full roster: `byte` `int` `rune` `string` `runes` `is_valid` `is_ascii` `copy` `freeze` `format`
`is_true`. No sequence members — one code point has no elements.

`is_valid()` reports whether this is a real symbol rather than an octet escape; `is_ascii()` whether it is
below `U+0080`.

### Conversions

Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given.

| member | result | partial? |
| --- | --- | --- |
| `int()` | the code point's value | total |
| `rune()` | identity | total |
| `byte()` | the same symbol as one octet | ASCII, **or an escape** — raises otherwise |
| `string()` / `runes()` | the symbol as text; an escape gives its octet | total (takes no default) |

```go
'A'.int()          // 65
'є'.int()          // 1108
'A'.byte()         // byte(65)
'є'.byte()         // raises: cannot convert rune to byte
'є'.byte(b'?')     // byte(63) — the default covers the partial edge
'A'.string()       // "A"
'є'.string()       // "є" — every code point has a text form
'є'.runes()        // u"є" — the text targets compose through string
```

`byte()` is encoding-based: it succeeds iff the symbol's UTF-8 representation is a single octet — exactly
ASCII. `U+0080`–`U+00FF` do *not* convert (their UTF-8 form is two octets); the Latin-1 numeric reading is
`.int()` explicitly.

There is deliberately no `bool()`, `float()`, or `decimal()` (through `.int()` — `int` is the sole gateway)
and no `bytes()` (write `.string().bytes()`). Text never parses into `rune` — `"65".rune()` raises; write
`"65".int().rune()`.

### Render — `format([spec])`

Total on every rune; the default verb is the symbol itself:

```go
'є'.format()       // "є"      — default verb is c
'є'.format("d")    // "1108"   — the code point in decimal
'є'.format("x")    // "454"    — hex, no 0x prefix on runes
'є'.format("U")    // "U+0454" — the U+ notation
'є'.format("q")    // "'є'"    — quoted literal
```

### Truthiness

`is_true()` — NUL is falsy, everything else truthy (`!!x` ⟺ `x != rune()`):

```go
'\x00'.is_true()   // false
'a'.is_true()      // true
```

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar — kept so generic code never type-errors:

```go
'a'.copy()     // 'a'
'a'.freeze()   // 'a'
```

## Migration notes

- **Overflow now raises.** Rune arithmetic that left the code-point space (or landed in the surrogate
  range) used to be able to surface as `U+FFFD`; it is now a catchable error at the operation, never a
  silent replacement character.
- **`bool()`/`float()`/`decimal()` are gone** — the spelling is through `.int()`.
- **`repeat()` is gone** from all scalars — promote through the one-symbol string:
  `'a'.string().repeat(n)` or `u"".pad_end(n, 'a')`.
- **`byte()` is ASCII-only by rule**, not by accident: one octet is a symbol's representation only when the
  UTF-8 form is that one octet.

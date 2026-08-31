# byte

One octet: an unsigned 8-bit value, 0–255.

## Overview

`byte` is an ordinal scalar with text content. It is two things at once:

- **A number-like value** for binary data — and **the only modular type in the language**: `byte` arithmetic
  wraps modulo 256. Every other numeric type (`int`, `float`, `decimal`, `rune`) raises on overflow; wrapping
  is `byte`'s documented character, matching Go/Rust `uint8` bit-for-bit.
- **A symbol, but only in ASCII.** As text content, an octet reads as a symbol only in `0x00`–`0x7F`;
  `0x80`–`0xFF` alone is not valid UTF-8 and has no symbol. The Latin-1 reading is never assumed — it is
  reachable explicitly through `.int()`.

`byte` values are immutable. The element type of [`bytes`](bytes.md).

## Literals and Construction

```go
b := b'A'          // byte(65)
nl := b'\n'        // byte(10) — byte-sized escapes accepted
ff := b'\xFF'      // byte(255)
```

A `b'...'` literal must resolve to exactly one octet: empty contents, multi-character contents, and code
points above 255 are compile errors.

```go
byte()              // byte(0) — the zero value
byte(65)            // byte(65) — conversion from int, raises outside 0..255
byte(256)           // raises: cannot convert int to byte
(256).byte(b'\x00') // byte(0) — the member's optional default rescues the failure
```

Promotion into a sequence is the count constructor's job (there is no `repeat()` on scalars):

```go
bytes(b'A', 3)     // bytes([65, 65, 65])
```

## Operators

### Arithmetic — wraps mod 256

`+` and `-` accept a `byte` or a plain `int` on either side; the result is always a `byte`, reduced mod 256
for *any* magnitude of the `int` operand:

```go
byte(10) + byte(3)   // byte(13)
b'\xFF' + 1          // byte(0)   — wraps; every other numeric type raises here
b'\x00' - 1          // byte(255)
byte(0) - 300        // byte(212) — any magnitude, still mod 256
1 + b'\xFF'          // byte(0)   — either operand order
-b'\x01'             // byte(255) — unary minus is the additive inverse mod 256
```

| operator | with `byte` | with `int` | with `float` / `decimal` |
| --- | --- | --- | --- |
| `+` `-` | `byte`, wraps | `byte`, wraps | raises |
| `*` `/` `%` | raises | raises | raises |

### Bitwise

`& | ^ &^` are same-type only; the shift count additionally accepts a plain `int`:

```go
b'\x0F' & b'\x03'    // byte(3)
b'\x0F' | b'\x30'    // byte(63)
b'\x0F' ^ b'\xFF'    // byte(240)
^b'\x00'             // byte(255) — unary complement
b'\x01' << 3         // byte(8)   — int shift count is fine
b'\x01' & 5          // raises: byte & int
```

### Comparison

`==`/`!=`/`<`/`<=`/`>`/`>=` compare `byte` with the other numeric-valued scalars by numeric value —
`byte`, `int`, `rune`, `bool` (as 0/1), and `float`. Next to *text*, equality reads the canonical text —
the symbol — exactly like `rune` (ordering against text raises):

```go
b'A' == 65           // true
b'A' < 66            // true
65 == b'A'           // true
b'A' < b'B'          // true
b'A' == 'A'          // true  — same code point
b'A' < 65.5          // true
"A" == b'A'          // true  — equality against text reads the symbol, like rune
"65" == b'A'         // false — the text is the symbol, never the number
```

### As text content

Next to the text types, a `byte` is a *symbol*, and only ASCII octets have one. A scalar on the left takes
the sequence's type:

```go
"abc" + b'a'         // "abca"
b'a' + "bc"          // "abc"  — scalar on the left, result is still a string
u"ab" + b'c'         // u"abc"
"abc" + byte(200)    // raises: an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got 200
u"ab" + byte(200)    // raises — same rule on runes
```

Into `bytes` a byte is always exactly one octet — no symbol question arises:

```go
bytes("ab") + b'\xFF'   // bytes([97, 98, 255]) — any octet, unconditionally
```

The same acceptance applies to search arguments:

```go
"abc".contains(b'a')       // true
"abc".index(b'c')          // 2
"abc".contains(byte(200))  // raises — no symbol exists
bytes("abc").contains(b'b') // true — always fine on bytes
```

The symbol is the `byte`'s **canonical text**: every convert-to-string surface answers it (matching
`.string()`), and a high octet — having no symbol — raises there. Dict keys, `join`, and record membership
all read it:

```go
d := dict(); d[b'A'] = 1
d.keys()                   // ["A"] — a byte key stores under its symbol
d[b'\xFF'] = 1             // raises: (key assign) expected string, got byte
["x", b'A'].join("-")      // "x-A"
["x", b'\xFF'].join("-")   // raises: cannot convert byte to string
b'A' in {A: 1}             // true — a record key is the symbol too
```

Display renders — `format()`, f-strings — are a different question and stay *numeric*; see
[Render](#render--formatspec) below.

## Members

The full roster: `byte` `int` `rune` `string` `runes` `copy` `freeze` `format` `is_true`. No sequence
members (`len`, `repeat`, …) — a `byte` has no elements.

### Conversions

Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given
(the default is not type-checked).

| member | result | partial? |
| --- | --- | --- |
| `int()` | the octet's value, 0–255 | total |
| `byte()` | identity | total |
| `rune()` | the same symbol | ASCII only — raises above `0x7F` |
| `string()` | the symbol, as text | ASCII only — raises above `0x7F` |
| `runes()` | the symbol, as runes | ASCII only — raises above `0x7F` |

```go
b'A'.int()             // 65
byte(200).int()        // 200 — total, the numeric reading always exists
b'A'.rune()            // 'A'
byte(200).rune()       // raises: cannot convert byte to rune
byte(200).rune('?')    // '?' — the default covers the partial edge
b'A'.string()          // "A"  — the SYMBOL, not the number
byte(200).string()     // raises: cannot convert byte to string
byte(200).string("?")  // "?"
b'A'.runes()           // u"A"
```

The `rune`/`string`/`runes` conversions are encoding-based: they succeed iff the octet is the complete
UTF-8 representation of a symbol on its own — exactly ASCII. There is deliberately no `bool()`, `float()`,
`decimal()`, or `bytes()` member: `int` is the sole gateway to the numeric and boolean domains
(`b.int().float()`, `b'0'.int().bool()`), and `string` is the gateway to octets (`b.string().bytes()`).
Text never parses directly into `byte` either — `"65".byte()` raises; write `"65".int().byte()`.

### Render — `format([spec])`

`format()` is the render, total on every octet; it shows the *number*, and the symbol spelling is the `c`
verb:

```go
b'A'.format()       // "65"  — default verb is d, always works
byte(200).format()  // "200"
f"{b'A'}"           // "65"  — an f-string is a render, not a conversion
format(b'A')        // "65"
b'A'.format("c")    // "A"   — ASCII character rendering
b'A'.format("q")    // "'A'" — quoted literal
b'A'.format("x")    // "0x41"
b'A'.format("b")    // "0b1000001"
```

`.string()` answers "what symbol is this?" (partial); `format()` answers "render this value" (total). Both
exist because they are different questions.

### Truthiness

`is_true()` — the zero octet is falsy, everything else truthy (`!!x` ⟺ `x != byte()`):

```go
b'\x00'.is_true()   // false
b'A'.is_true()      // true
!!b'\x00'           // false
```

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar — kept so generic code never type-errors:

```go
b'A'.copy()     // byte(65)
b'A'.freeze()   // byte(65)
```

## Migration notes

- **`b'A'.string()` used to answer `"65"` — the number. It now answers `"A"` — the symbol**, and raises
  above `0x7F`. Every convert-to-string surface follows: a `b'A'` dict key now stores `"A"` (was `"65"`),
  `join` renders the symbol, and `"A" == b'A'` is `true`. The old render spelling is `format()`:
  `b'A'.format()` → `"65"`, total on every octet.
- **Wrapping is the documented contract**, and `byte` is the *only* type with it — every other numeric
  raises on overflow. Mixed `byte`/`int` arithmetic reduces into the byte ring: the result is a `byte`,
  never an `int`, whichever side the `int` stood on.
- **`bool()` is gone** — write `b.int().bool()`. This also removes the old `b'0'` → `true` trap.
- **`float()`/`decimal()` are gone** — through `.int()`.
- **`repeat()` is gone** from all scalars — the promotion spelling is the count constructor
  `bytes(b'A', 3)`.

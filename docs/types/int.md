# int

Checked 64-bit signed integers: arithmetic that overflows raises, it never wraps.

## Overview

`int` is a 64-bit signed integer (-9,223,372,036,854,775,808 to 9,223,372,036,854,775,807; the bounds are
available as `math.max_int` / `math.min_int`). It is a **checked** numeric: any arithmetic whose mathematical
result does not fit in 64 bits raises a catchable error instead of silently wrapping. Wide modular arithmetic is
not part of the language — [`byte`](byte.md) is the only type with wrap-around arithmetic.

`int` belongs to the numeric family alongside [`float`](float.md) and [`decimal`](decimal.md): it pairs with both
in arithmetic and compares exactly against both. Use `int` for counts, indexes, identifiers, and timestamps; use
`decimal` for money and `float` for measurements.

Like every numeric type, `int` answers the sentinel predicates `is_nan()` / `is_inf()` — constantly `false`,
since an `int` can never hold such a state. That uniformity lets generic numeric code interrogate any number
without switching on its concrete type.

## Literals

```go
i = 42            // decimal
h = 0x2a          // hexadecimal, 42
o = 0o755         // octal, 493
b = 0b1010        // binary, 10
big = 1_000_000   // underscores group digits in any base
```

A literal with a decimal point, an exponent, or an `f` suffix is a [`float`](float.md); a `d` suffix makes a
[`decimal`](decimal.md).

## Arithmetic and operators

| operator | meaning | notes |
| --- | --- | --- |
| `+` `-` `*` | add, subtract, multiply | raise on overflow |
| `/` | division, truncated toward zero | `7 / 2` → `3`, `-7 / 2` → `-3`; raises on `/ 0` |
| `%` | remainder (sign follows the dividend) | `-7 % 2` → `-1`; raises on `% 0` |
| `&` `\|` `^` `&^` | bitwise and, or, xor, and-not | never overflow |
| `<<` `>>` | shift left / right | see the shift rules below |
| `-x` | negation | raises on `-math.min_int` |
| `^x` | bitwise complement | `^5` → `-6` |

### Overflow raises, always

Every arithmetic overflow raises a catchable error of kind `invalid_value` with the message `int overflow`:

```go
math = import("math")

math.max_int + 1            // Error: int overflow
math.min_int - 1            // Error: int overflow
math.min_int * -1           // Error: int overflow
-math.min_int               // Error: int overflow — negation is arithmetic too
math.min_int / -1           // Error: int overflow — the one division that overflows
3037000500 * 3037000500     // Error: int overflow
```

The error is an ordinary raise — `recover()` catches it:

```go
fmt = import("fmt")

checked_add = func(a, b) res {
    defer func() {
        e = recover()
        if e != undefined { res = undefined; fmt.println("overflow: ", string(e)) }
    }()
    res = a + b
}
```

### Division by zero

`/ 0` and `% 0` raise a catchable error of kind `division_by_zero`:

```go
1 / 0    // Error: division_by_zero
1 % 0    // Error: division_by_zero
```

### Shift rules

`<<` raises whenever the result would lose information — a value bit or the sign bit shifted out, or an invalid
count. `>>` is an arithmetic shift: for counts of 64 and above it saturates at the sign bit (no information is
invented), and it raises only on a negative count.

```go
1 << 62      // 4611686018427387904
-1 << 1      // -2 — sign preserved, nothing lost
1 << 63      // Error: int overflow — the bit lands on the sign
1 << 64      // Error: int overflow — count out of range
5 << -1      // Error: int overflow — negative count

-8 >> 1      // -4
5 >> 100     // 0  — saturates at the sign bit
-5 >> 100    // -1 — saturates at the sign bit
5 >> -1      // Error: int overflow — negative count
```

### Pairing with float and decimal

`int` mixes freely with the other two numerics; the result takes the wider representation:

```go
1 + 1.5             // 2.5   — float (float arithmetic rules apply: 1 / 0.0 raises)
10 / 4.0            // 2.5   — float
3 + 1.5d            // 4.5d  — decimal
10 / 4d             // 2.5d  — decimal
```

`float` and `decimal` never mix with *each other* in arithmetic — `int` is the only operand type both accept.

## Comparisons and cross-type pairing

`==` and the orderings are **exact mathematical comparisons** across the whole numeric family — no operand is
silently rounded to make the comparison cheap:

```go
1 == 1.0                                   // true
1 == 1d                                    // true
2 < 2.5                                    // true
2 < 2.5d                                   // true
9007199254740993 == 9007199254740992.0     // false — exact, no rounding to float
9007199254740993 == 9007199254740993.0     // false — that float value IS ...992
9007199254740993 < 9007199254740994.0      // true
```

`bool`, `byte`, and `rune` widen to their integer value for comparison (`true == 1`, `65 == 'A'`, `5 < 'A'` are
all `true`). Equality against a `string` compares the int's canonical text form; ordering against text does not
exist:

```go
5 == "5"     // true  — canonical form
5 == "05"    // false — "05" is not the canonical rendering of 5
5 < "6"      // Error: int < string — numeric-vs-text ordering is undefined
```

## Members

### Numeric members

```go
(-5).abs()      // 5
(5).abs()       // 5
(-5).sign()     // -1
(0).sign()      // 0
(5).sign()      // 1
```

**Edge case:** `math.min_int.abs()` currently answers `math.min_int` unchanged (the true absolute value does not
fit in 64 bits). Do not rely on it; guard the minimum explicitly where it can occur.

### Sentinel predicates

Constant `false` — an `int` can never be NaN or infinite. They exist so generic numeric code can ask the question
of any number:

```go
(5).is_nan()    // false, always
(5).is_inf()    // false, always
```

`int` has no `is_zero()` / `is_negative()` / `is_positive()` members — write the comparison (`x == 0`, `x < 0`,
`x > 0`).

### Truthiness

Truthiness is inequality with the type's zero, in both spellings — the member and the free builtin:

```go
(0).is_true()    // false
(7).is_true()    // true
is_true(0)       // false
is_true(-3)      // true
!!0              // false
```

### copy / freeze

Identity no-ops — an `int` is always immutable — kept so generic code can `x.copy()` any value without a type
error:

```go
(5).copy()      // 5
(5).freeze()    // 5
```

### format

`format([spec])` renders the value; with no spec it is the plain decimal rendering. Verbs: `d` (decimal,
default), `b` / `o` / `x` / `X` (binary / octal / hex, prefixed `0b` / `0o` / `0x`), `c` (the code point as a
character), `q` (the code point as a quoted character literal). Grouping (`,` for decimal, `_` for the other
bases), sign, and zero-padding compose with them:

```go
(255).format("x")          // "0xff"
(255).format("X")          // "0xFF"
(255).format("b")          // "0b11111111"
(255).format("o")          // "0o377"
(1234567).format(",d")     // "1,234,567"
(42).format("+d")          // "+42"
(42).format("08d")         // "00000042"
(65).format("c")           // "A"
(10).format("q")           // "'\n'"
```

### No sequence members

`int` is a scalar: it has no `len()`, no elements, and no sequence members. In particular `repeat` does not exist
on any scalar — promotion into a sequence is the count constructors' job (`T(x, count)` builds `count` copies
of `x`):

```go
(5).repeat(2)     // Error: type int has no method repeat
array(5, 3)       // [5, 5, 5] — the promotion spelling: three copies of 5
bytes(b'a', 3)    // bytes([97, 97, 97])
```

## Conversions

Every conversion follows one contract: `x.T()` answers a valid `T` or raises a catchable error (kind
`conversion`); `x.T(default)` answers the default instead of raising. The free form `T(x)` is the same
conversion — the default slot is the member's only; `undefined.T(d)` → `d` is the standard maybe-missing
spelling.

| target | behavior |
| --- | --- |
| `int` | identity — the same value |
| `float` | the nearest float64; exact up to 2⁵³, silently approximate above (in-range resolution loss) |
| `decimal` | exact |
| `bool` | `false` for 0, `true` otherwise |
| `string` / `runes` | the decimal rendering (total — takes no default) |
| `byte` | value in 0–255, else raise-or-default |
| `rune` | a valid Unicode code point (0–0x10FFFF, surrogates excluded), else raise-or-default |
| `time` / `time_ms` / `time_micro` / `time_nano` | a unix timestamp in the named encoding (total — take no default) |

```go
(65).float()        // 65.0
(65).decimal()      // 65d
(2).bool()          // true
(65).string()       // "65"
(65).runes()        // u"65"
(65).byte()         // byte(65)
(300).byte()        // Error: cannot convert int to byte
(300).byte(b'0')    // byte(48) — the default answers instead
(65).rune()         // 'A'
(55296).rune()      // Error: cannot convert int to rune — a surrogate is not a code point
(55296).rune('?')   // '?'
(1114112).rune()    // Error: cannot convert int to rune — past 0x10FFFF
```

### The time family

In conversion context an int is a **unix timestamp**, never a duration (the duration reading belongs to operator
context — `t + n` adds nanoseconds; see [time](time.md)). Each member names the encoding it reads and inverts the
`time` accessor with the matching suffix (`time_ms` ↔ `unix_ms`, and the unsuffixed `time()` ↔ `unix()` /
`int()`). All produce UTC:

```go
(0).time()                              // time("1970-01-01T00:00:00Z")
(1704067200).time()                     // time("2024-01-01T00:00:00Z")
(1704067200123).time_ms()               // time("2024-01-01T00:00:00.123Z")
(1704067200123456).time_micro()         // time("2024-01-01T00:00:00.123456Z")
(1704067200123456789).time_nano()       // time("2024-01-01T00:00:00.123456789Z")
```

### Parsing text into int

The free `int(x)` (equivalently `"...".int()`) parses the canonical decimal form only — no hex
prefixes, no underscores, no fractions; the member's optional default rescues bad data:

```go
int("123")            // 123
int("12.5")           // Error: cannot convert string to int — parse, not truncate
int("0x2a")           // Error: cannot convert string to int
"abc".int(0)          // 0 — the member's default rescues bad data
undefined.int(5)      // 5 — the maybe-missing form
int(undefined)        // Error: cannot convert undefined to int: value is missing
```

The default rescues *data*, never a program error — a wrong argument type raises even with a default present.

## Migration notes

- **Overflow used to wrap silently; it now raises.** Every `+ - * / << -x` that leaves the 64-bit range is a
  catchable `invalid_value` raise (`int overflow`). Code that relied on wrap-around must move to explicit checks
  or to `byte`, the only modular type.
- **The `**` exponentiation operator no longer exists.** Use `math.pow(a, b)` (answers a `float`) or explicit
  multiplication.
- **`repeat()` was removed from scalars.** `(5).repeat(n)` is gone; write `array(5, n)` (or `bytes(fill, n)`)
  to build a filled sequence.
- **Conversions no longer answer silent zeros.** A failed conversion raises or takes the explicit
  `x.T(default)`; check errors instead of sentinel values.

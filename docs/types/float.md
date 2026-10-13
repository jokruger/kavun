# float

IEEE 754 double-precision floats with an arithmetic quarantine: no arithmetic result is ever NaN or ±Inf — it
raises instead.

## Overview

`float` is a 64-bit IEEE 754 binary float. Kavun quarantines its error states: an arithmetic operation whose
result would be NaN or ±Inf **raises a catchable error** instead of letting the sentinel flow through a
calculation silently. The sentinel *values* themselves remain first-class: they can arrive from parses
(`float("inf")`), from the host, or from JSON, and they can be stored, compared, and sorted — only arithmetic
refuses them. `is_nan()` / `is_inf()` interrogate them without triggering the raise being avoided.

`float` belongs to the numeric family alongside [`int`](int.md) and [`decimal`](decimal.md). Use `float` for
measurements and physics-style computation where binary rounding is acceptable; use `decimal` for money and any
result that must be exact in base 10 (`0.1 + 0.2 == 0.3` is `false` on floats and `true` on decimals).

## Literals

```go
x = 1.5          // decimal point
k = 1e3          // exponent, 1000.0
m = 2.5e-3       // 0.0025
n = 3f           // f suffix: a float-typed 3
u = 1_000.25     // underscores group digits
h = 0x1.8p1      // hex float with binary exponent, 3.0
```

## Rendering

`string()` and the default `format()` render the shortest decimal string that round-trips — which is why binary
rounding is visible in full rather than hidden:

```go
string(1.5)          // "1.5"
string(0.1 + 0.2)    // "0.30000000000000004" — the true stored value
string(1e21)         // "1000000000000000000000"
string(-0.0)         // "-0"
```

The **display** form — what a container shows, and what a doc example writes — always carries a decimal point,
so a whole float never reads back as an `int`:

```go
format("{0}", [[3.0, 3]])   // "[3.0, 3]" — the float keeps its point, the int has none
format("{0}", [[0.0]])      // "[0.0]"
```

The **conversion** stays bare on purpose: `3 == 3.0` is `true`, so the two must produce the same key text and
the same `join` output. `(3.0).string()` is `"3"`, `f"{3.0}"` is `"3"`, and `json.encode` is unchanged.

## Arithmetic and operators

| operator | meaning | notes |
| --- | --- | --- |
| `+` `-` `*` `/` | IEEE arithmetic | result checked by the quarantine |
| `%` | floating-point remainder | `5.5 % 2` → `1.5`; checked too |
| `-x` | negation | arithmetic — checked (see below) |

### The quarantine: NaN/Inf results raise

Every arithmetic result is checked; a would-be NaN or ±Inf raises a catchable error of kind `invalid_value`:

```go
1.0 / 0.0       // Error: float overflow or division by zero
-1.0 / 0.0      // Error: float overflow or division by zero
0.0 / 0.0       // Error: invalid float arithmetic (NaN result)
1e308 * 10.0    // Error: float overflow or division by zero — overflow to Inf
1.0 % 0.0       // Error: invalid float arithmetic (NaN result)
```

The rule applies to sentinel *operands* too — arithmetic refuses them even though the values are storable:

```go
inf = float("inf")
nan = float("nan")

inf - inf       // Error: invalid float arithmetic (NaN result)
inf + 1.0       // Error: float overflow or division by zero
nan + 1.0       // Error: invalid float arithmetic (NaN result)
-inf            // Error — unary minus is arithmetic; negate no sentinel
```

### Pairing with int

`float` pairs with `int` in arithmetic; the result is `float` under the same quarantine (`1 / 0.0` raises).
`float` **never pairs with `decimal` in arithmetic** — only in comparisons. Convert explicitly
(see [decimal](decimal.md)):

```go
1.5 + 1       // 2.5
1.5 + 1d      // Error: float + decimal — convert explicitly: 1.5.decimal() + 1d
```

## NaN and Inf values

The sentinels arrive from parses and hosts, never from Kavun arithmetic:

```go
inf = float("inf")       // +Inf ("Infinity" and case variants parse too)
ninf = float("-inf")     // -Inf
nan = float("nan")       // NaN

inf.is_inf()    // true
nan.is_nan()    // true
(1.5).is_nan()  // false
```

`is_inf()` takes no sign argument — negative infinity is a composition:

```go
ninf.is_inf() && ninf.sign() < 0    // true — "is it -Inf"
```

### Total-order comparisons

Comparison treats the whole float domain as one **total order**: NaN is the unique minimum (below even `-Inf`)
and equal only to another NaN. This keeps sorting deterministic instead of IEEE-unordered:

```go
nan == nan                     // true — reflexive, unlike raw IEEE
nan < float("-inf")            // true — NaN is the unique minimum
inf > 1e308                    // true
[2.0, nan, 1.0, inf].sort()    // [NaN, 1.0, 2.0, +Inf]
nan == 1.0                     // false
nan < 1                        // false — NaN orders below numbers, not against them piecewise
```

### Truthiness

Truthiness is inequality with `0.0`. NaN is an error state, not a domain value — a boolean context **raises**
rather than answering:

```go
(0.0).is_true()     // false
(-0.0).is_true()    // false
(0.5).is_true()     // true
!!0.0               // false
is_true(float("nan"))          // Error: float NaN is neither true nor false in a boolean context
float("nan").is_true()         // same raise — member and free spelling agree
if float("nan") { ... }        // raises the same way
```

`float("inf").is_true()` is `true` — Inf is nonzero and has a definite answer.

## Members

### Numeric members

```go
(-2.5).abs()            // 2.5
(-2.5).sign()           // -1
(0.0).sign()            // 0
float("nan").sign()     // 0
(0.0).is_zero()         // true
(-2.5).is_negative()    // true
(2.5).is_positive()     // true
```

Negative zero compares equal to zero and reads as zero everywhere:

```go
(-0.0).is_zero()        // true
(-0.0).is_negative()    // false
(-0.0).sign()           // 0
```

### Arithmetic members

A numeric operation is a `float` member exactly when [`decimal`](decimal.md) has it — same name, same reading
(pinned by a parity test). Like the operators, no member produces NaN or ±Inf: such a result **raises**
`invalid_value` naming the member.

```go
16.0.sqrt()               // 4.0
(-1.0).sqrt()             // Error: (sqrt) the result is not a number
2.0.pow(10)               // 1024.0     — the exponent may be an int or a float
2.0.pow(0.5)              // 1.4142135623730951
0.0.pow(-1)               // Error: division by zero
1.5.copy_sign(-2)         // -1.5
5.0.clamp(0, 2)           // 2.0        — clamp(lo, hi) needs lo <= hi
2.5.negate()              // -2.5
1.0.next_up() > 1.0       // true       — the next representable float
2.0.is_integer()          // true       (NaN and ±Inf are not integers)
5.5.quo_rem(2)            // [2.0, 1.5] — quotient toward zero, remainder with the receiver's sign
1.0.quo_rem(0)            // Error: division by zero
```

`exp()`, `ln()`, `log2()` and `log10()` are float-only: their result is inexact, so `decimal` spells them with a
scale and a mode (`exp_round(scale, mode)`, …).

```go
1.0.exp()                 // 2.718281828459045
1.0.ln()                  // 0.0
8.0.log2()                // 3.0
1000.0.log10()            // 3.0
0.0.ln()                  // Error: (ln) the result is infinite
```

The rest of float math — trigonometry, `cbrt`, `hypot`, the special functions — is in the [`math`](../stdlib.md#math)
module.

### Rounding

`round(places, mode)` and its fixed-mode twins `round_half_even(places)`, `round_half_up`, `round_half_down`,
`round_up`, `round_down`, `round_ceiling`, `round_floor`, plus `round_to_multiple(m, mode)` and
`round_significant(k, mode)`, are **decimal's contract**: `places` is required (negative rounds to tens,
hundreds, …), and `mode` is one of `half_even`, `half_up`, `half_down`, `up`, `down`, `ceiling`, `floor`.

A float rounds **the digits it prints** — its shortest decimal text, the one `string()` shows — not its exact
binary value. So rounding does what the reader expects from the number they see:

```go
2.675.round(2, "half_up")          // 2.68  (2.675 is stored as 2.67499999…; it prints, and rounds, as 2.675)
(0.1 + 0.2).round(1, "half_even")  // 0.3
1234.5.round(-2, "half_up")        // 1200.0
3.7.round_floor(0)                 // 3.0   — floor / ceil / trunc are round_floor / round_ceiling / round_down at 0
1.234.round_to_multiple(0.05, "half_up")    // 1.25
123456.0.round_significant(2, "half_even")  // 120000.0
1.5.round(2, "bankers")            // Error: unknown rounding mode "bankers", expected one of: ceiling, …
```

Float rounds those digits itself, with the same seven modes as `decimal` (and, wherever `decimal` can hold the value,
exactly the same answer), so no magnitude is out of range: `1e-30.round(2, "half_up")` is `0.0`, and
`1e300.round(2, "half_up")` is `1e300` unchanged. Only a result past float's own range raises
(`1.7976931348623157e308.round(-308, "up")` would be `2e308`); NaN and ±Inf have no digits to round and raise too.

### Durations

A float builds an int **duration in nanoseconds**, rounded to the nearest one — the same members as
[int](int.md#durations): `1.5.hours()` is `5400000000000`, `0.25.seconds()` is `250000000`. NaN, ±Inf and a
result past int's range raise.

### Sentinel predicates

`is_nan()` / `is_inf()` are the safe way to interrogate a value that arithmetic would refuse; every numeric type
answers them (constantly `false` on `int` and for `is_inf` on `decimal`), so generic code never type-switches.

### copy / freeze

Identity no-ops (a `float` is always immutable), kept for generic code: `(1.5).copy()` → `1.5`.

### format

Verbs `f` / `F` (fixed-point, default precision 6), `e` / `E` (scientific), `g` / `G` (shortest, the default),
`%` (×100 with a percent sign); precision, grouping, sign, and zero-padding compose. NaN/Inf render as text:

```go
(1234.5678).format(".2f")     // "1234.57"
(1234.5678).format(",.2f")    // "1,234.57"
(0.1234).format(".1%")        // "12.3%"
(12345.678).format("e")       // "1.234568e+04"
(1.5).format("+.1f")          // "+1.5"
float("nan").format("f")      // "NaN"
float("inf").format("f")      // "Inf"
```

### No sequence members

Scalars have no `len()`, no elements, and no `repeat`:

```go
(1.5).repeat(2)    // Error: type float has no method repeat
```

## Comparisons and cross-type pairing

Cross-type `==` / `<` against `int` and `decimal` are **exact mathematical comparisons** — the float's true
stored value is compared, with no rounding of either side:

```go
1 == 1.0                              // true
0.1d == 0.1                           // false — float 0.1 is really 0.1000000000000000055511…
0.1d < 0.1                            // true  — the decimal is below the float's true value
9007199254740993 == 9007199254740993.0   // false — that float literal rounds to …992
```

`bool` / `byte` / `rune` widen to their integer value. Equality against a `string` compares the canonical text
form (`1.5 == "1.5"` → `true`, `float("nan") == "NaN"` → `true`); ordering against text raises.

## Conversions

`x.T()` answers a valid `T` or raises (kind `conversion`); `x.T(default)` answers the default instead. The free
form `T(x)` is the same conversion — the default slot is the member's only; `undefined.T(d)` → `d` covers
maybe-missing data.

| target | behavior |
| --- | --- |
| `float` | identity |
| `int` | truncation toward zero for in-range values (documented resolution loss); NaN, ±Inf, and out-of-range raise-or-default |
| `decimal` | exact-enough shortest decimal (`(0.1).decimal()` → `0.1d`); NaN/±Inf raise-or-default |
| `bool` | zero test; NaN raises-or-defaults |
| `string` / `runes` | the shortest round-trip rendering (total — takes no default) |
| `time` | unix timestamp as sec.frac — **lossy**, see below |

```go
(1.9).int()               // 1  — truncation toward zero
(-1.9).int()              // -1
float("nan").int()        // Error: cannot convert float to int
float("nan").int(0)       // 0
(1e300).int()             // Error: cannot convert float to int — out of range
(1.5).decimal()           // 1.5d
float("inf").decimal()    // Error: cannot convert float to decimal
(0.5).bool()              // true
(1.5).string()            // "1.5"
(1.5).runes()             // u"1.5"
```

Truncation is a documented in-range resolution loss of *conversion*. **Count slots are stricter**: an argument
position that means a count, width, or edit position accepts a float only losslessly —

```go
"ab".repeat(2.0)    // "abab" — 2.0 converts losslessly
"ab".repeat(1.5)    // Error: (repeat) argument first must be a whole number, got 1.5
```

### time

A float in conversion context is a unix timestamp read as `seconds.fraction` — and float64 precision only reaches
about a microsecond for present-day timestamps:

```go
(0.5).time()             // time("1970-01-01T00:00:00.5Z")
(1704067200.123).time()  // time("2024-01-01T00:00:00.122999907Z") — float rounding, not a bug
```

Use `decimal` (exact to the nanosecond) or `int` `.time_nano()` when the sub-second part must be right.

### Parsing text into float

```go
float("1.5")           // 1.5
float("inf")           // +Inf — a parse is not arithmetic; the value is legal
float("nan")           // NaN
float("abc")           // Error: cannot convert string to float
float("1e309")         // Error: cannot convert string to float — not representable
"abc".float(0.0)       // 0.0 — the member's default rescues bad data
undefined.float(1.0)   // 1.0 — the maybe-missing form
float(3)               // 3.0
```

There is no `byte()` or `rune()` conversion on `float` — `int` is the sole gateway between the numeric family and
the ordinal scalars (`x.int().byte()`).

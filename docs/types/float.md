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

`float` has **no** `sqrt` / `next_up` / `next_down` / `negate` members — those exist on [`decimal`](decimal.md)
only. For float math use the `math` module (`math.sqrt(2.0)` → `1.4142135623730951`, `math.pow`, …) or the
operators (`-x` negates).

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

## Migration notes

- **NaN and ±Inf used to flow through arithmetic silently; arithmetic on or into them now raises.** The values
  are still representable (parses, hosts, JSON) — check with `is_nan()` / `is_inf()`, which never raise.
- **NaN truthiness now raises.** `if x { … }` on a NaN float is a catchable error, not `false`.
- **Comparisons are a total order.** `nan == nan` is `true` and NaN sorts as the unique minimum, making
  `sort()` on float arrays deterministic; IEEE "unordered" semantics are gone.
- **Cross-type comparisons are exact.** `int`/`decimal` vs `float` comparisons no longer round either operand;
  very large integers that used to compare equal to nearby floats now compare correctly.
- **`repeat()` was removed from scalars**, and count-valued arguments accept floats only losslessly
  (`repeat(1.5)` raises where `repeat(2.0)` converts).

# decimal

128-bit base-10 decimals (dec128) for exact monetary arithmetic.

## Overview

`decimal` is a 128-bit decimal floating-point number: up to 38 significant digits with a fractional scale of 0 to
19 digits. Arithmetic is exact in base 10 — `0.1d + 0.2d == 0.3d` is `true`, where the float spelling is famously
`false`. Money and anything else that must be exact in base 10 belongs in `decimal`; measurements and physics
belong in [`float`](float.md).

`decimal` belongs to the numeric family alongside [`int`](int.md) and `float`: it pairs with `int` in arithmetic
and compares exactly against both. It has no Inf representation at all (`is_inf()` is constantly `false`), and a
NaN exists only as an error state a **host** can hand in — no operation in the language produces one, because
every path that would raises instead (see [NaN as an error state](#nan-as-an-error-state)).

## Literals and construction

A base-10 numeric literal with a `d` suffix is a decimal:

```go
price = 19.99d
two = 2d
neg = -2.5d        // unary minus, exact
```

The free `decimal(x)` constructor (equivalently `x.decimal()`) converts and parses; the member's optional
`x.decimal(default)` rescues bad data:

```go
decimal()                  // 0d — the zero value
decimal("0.1")             // 0.1d — parsing text is the lossless entry point
decimal(3)                 // 3d
decimal(1.5)               // 1.5d — from float, shortest decimal reading
decimal("abc")             // Error: cannot convert string to decimal — parse raises, no NaN sentinel
"abc".decimal(0d)          // 0d — the member's default rescues bad data
undefined.decimal(0d)      // 0d — the maybe-missing form
decimal("inf")             // Error: cannot convert string to decimal — the type has no Inf
decimal("1e3")             // 1000d — scientific notation parses, in either case ("1E3" too)
```

### Scientific notation and scale

Both the mantissa's own digits and the exponent contribute to the resulting scale, so the notation carries
precision rather than discarding it:

```go
decimal("1.0e-3").format("s")   // "0.0010" — scale 4: mantissa scale 1, shifted 3 more places
decimal("2.5E2").scale()        // 0 — 250d; the exponent absorbed the fractional digit
decimal("1e-25")                // Error: cannot convert string to decimal — past the scale-19 ceiling
```

## Arithmetic and operators

| operator | meaning | notes |
| --- | --- | --- |
| `+` `-` `*` | add, subtract, multiply | exact; raise `invalid_value` past 38 digits |
| `/` | division | full available precision (scale 19); `/ 0d` raises `division_by_zero` |
| `%` | remainder | `10d % 3d` → `1d`; `% 0d` raises `division_by_zero` |
| `-x` | negation | exact, never overflows |

```go
0.1d + 0.2d            // 0.3d — exact
0.1d + 0.2d == 0.3d    // true
0.1 + 0.2 == 0.3       // false — the float contrast
1d / 3d                // 0.3333333333333333333d — 19 fractional digits
```

### Scale of a result

Each value carries a scale (count of stored fractional digits, observable via `scale()` and `format("s")`).
Addition keeps the wider operand's scale, multiplication adds scales, division answers the full 19:

```go
(decimal("0.10") + decimal("0.20")).format("s")    // "0.30" — scale 2
(decimal("1.5") * decimal("2.00")).scale()          // 3 — scales add
(decimal("10.00") / 4d).format("s")                 // "2.5000000000000000000" — scale 19
```

A `d`-suffixed literal carries the scale it was written with, and two literals that differ only in trailing
zeros are two different constants:

```go
(1.5d).scale()     // 1
(1.50d).scale()    // 2
1.5d == 1.50d      // true — numeric equality is scale-blind
```

Use `rescale_*()` / `canonical()` or the rounding family to tidy a result for storage or display.

### Rounding at the precision ceiling

A decimal holds up to 38 significant digits with at most 19 fractional places. When an exact result does not
fit, the scale is reduced to the largest that still holds the integer part, and the discarded digits are
**rounded — not reported**:

```go
1d / 3d                             // 0.3333333333333333333d — 19 places, the rest dropped
decimal("0.0000000001") * decimal("0.0000000001")
                                    // 0.0000000000000000000d — the exact 1e-20 needs 20 places,
                                    // so it rounds to zero at scale 19
```

This is ordinary and expected: no fixed-point type can represent a third, and the same ceiling applies to a
very small product. A result **cannot report whether it was rounded** — the type carries a value and a scale,
with nowhere to record that digits were lost. Only a result whose *integer* part does not fit raises
(see [Overflow raises](#overflow-raises)).

The practical answer is to fix the scale where it matters rather than relying on the ceiling: state the scale
and the policy at each point a result is pinned down, with `rescale_bank(2)`, `div_round_bank(y, 2)` and the
rest of the [`rescale_*` / `div_round_*` families](#rounding-to-an-exact-scale--the-rescale_-family). A
calculation carried at a working scale well below 19 is unaffected by the ceiling.

### Overflow raises

A result that no longer fits 128 bits raises a catchable `invalid_value` error:

```go
x = decimal("99999999999999999999999999999999999999")
x + x    // 199999999999999999999999999999999999998d — still fits
x * x    // Error: invalid_value: (*) overflow
```

### Division by zero

`x / 0d` and `x % 0d` raise `division_by_zero`, the same kind `int` answers for `1 / 0`. There is nothing to
check after the division:

```go
1d / 0d              // Error: division_by_zero: division by zero
0d / 0d              // Error: division_by_zero: division by zero
1d % 0d              // Error: division_by_zero: division by zero
```

### decimal never mixes with float

Arithmetic between `decimal` and `float` raises — there is no silently-chosen loser of precision. Convert
explicitly on the side you mean:

```go
1.5d + 1.0    // Error: decimal + float
1.5 + 1d      // Error: float + decimal
1.5d + 1      // 2.5d — int pairs fine
1.5d * 2      // 3d
1.5d + (1.0).decimal()    // 2.5d — explicit, decimal wins
```

Comparisons across the pair are allowed (and exact) — see below.

## NaN as an error state

dec128 signals every failure the same way — it answers NaN — and Kavun treats that as an **error state, not a
value**. Unlike an IEEE NaN it even compares equal to itself, so it behaves as an ordinary number everywhere
downstream and a wrong answer would propagate silently through a whole script.

So **no operation in the language produces one**. Every path that would raises instead:

```go
decimal("abc")       // Error: conversion — the parse
1d / 0d              // Error: division_by_zero
0d % 0d              // Error: division_by_zero
(-1d).sqrt()         // Error: invalid_value: (sqrt) square root of negative number
x * x                // Error: invalid_value: (*) overflow  — past the 128-bit coefficient
(1e39).decimal()     // Error: conversion — the float is finite but too large for dec128
```

That mirrors `float`, whose arithmetic already raises on a NaN or ±Inf result.

`is_nan()` and `error_details()` remain because a **host** can bind a NaN decimal through the embedding API —
nothing in the language can. For a decimal a script produced, `is_nan()` is always `false` and
`error_details()` is always `undefined`. On a host-supplied NaN:

```go
n.is_nan()           // true
n.error_details()    // error("...") — why this NaN exists
n.is_true()          // Error: decimal NaN is neither true nor false in a boolean context
n.int()              // Error: cannot convert decimal to int
n.sign()             // 0
```

In comparisons such a NaN follows the numeric family's total order: the unique minimum, equal only to another
NaN, so sorting stays deterministic.

## Comparisons and cross-type pairing

`==` and the orderings are **exact mathematical comparisons**. Trailing zeros never matter — equality compares
numeric value, not scale:

```go
2.50d == 2.5d      // true — scale is representation, not identity
1d == 1            // true
1d < 1.5           // true — float comparison is allowed (only arithmetic is refused)
0.1d == 0.1        // false — float 0.1 is really 0.1000000000000000055511…
0.1d < 0.1         // true  — exactly below the float's true value
(0.1).decimal() == 0.1d    // true — the float's shortest decimal reading is 0.1
```

`bool` / `byte` / `rune` widen to their integer value. Equality against a `string` compares the decimal's own text
form, which carries its scale; ordering against text raises:

```go
decimal("1.50") == "1.50"    // true — the scale is part of the rendering
decimal("1.50") == "1.5"     // false — a different text, though the same NUMBER
decimal("1.50") == 1.5d      // true — numeric equality is scale-blind, text equality is not
```

## Members

### The rounding family

Every rounding member takes the target scale (0–19) as its one required argument and answers a `decimal`. The
names state the tie-breaking / direction policy exactly; here is the whole family on `2.5` / `-2.5` at scale 0:

| member | policy | `2.5d` | `-2.5d` |
| --- | --- | --- | --- |
| `round_half_away_from_zero(n)` | ties away from zero ("schoolbook") | `3d` | `-3d` |
| `round_half_toward_zero(n)` | ties toward zero | `2d` | `-2d` |
| `round_bank(n)` | ties to even (banker's) | `2d` | `-2d` |
| `round_up(n)` | always toward +∞ (ceiling) | `3d` | `-2d` |
| `round_down(n)` | always toward −∞ (floor) | `2d` | `-3d` |
| `round_away_from_zero(n)` | any fraction rounds away from zero | `3d` | `-3d` |
| `round_toward_zero(n)` | any fraction drops (= `trunc`) | `2d` | `-2d` |
| `trunc(n)` | drop digits past scale n | `2d` | `-2d` |

```go
(3.5d).round_bank(0)                     // 4d — ties go to the even neighbor
(2.345d).round_half_away_from_zero(2)    // 2.35d
(2.4d).round_up(0)                       // 3d — direction applies to any fraction, not just ties
(2.5d).round_half_away_from_zero(-1)     // Error: (round_half_away_from_zero) scale must be between 0 and 19
```

There is **no plain `round()`, `floor()`, or `ceil()`** — every rounding spells its policy: schoolbook rounding
is `round_half_away_from_zero(n)`, floor is `round_down(0)`, ceiling is `round_up(0)`. The same seven policy
names recur in every family below, so learning them once is enough.

### Rounding to an exact scale — the `rescale_*` family

`round_*(n)` rounds to **at most** n places and leaves a shorter value alone; `rescale(n)` pads and truncates
but never rounds. Neither says *"exactly n places, rounded"* — the shape a monetary result usually needs. That
is `rescale_*(n)`:

```go
(1.5d).round_bank(2).format("s")      // "1.5"  — at most 2 places; already shorter, so unchanged
(1.5d).rescale(2).format("s")         // "1.50" — exactly 2 places, but truncates when narrowing
(1.5d).rescale_bank(2).format("s")    // "1.50" — exactly 2 places, rounding when narrowing
(2.345d).rescale_bank(2)              // 2.34d — ties to even
(2.345d).rescale_half_away_from_zero(2)   // 2.35d
```

`rescale_toward_zero(n)` is `rescale(n)` under another name — truncation is one of the seven policies.

| member | ties / direction | `2.345d` → 2 | `-2.345d` → 2 |
| --- | --- | --- | --- |
| `rescale_half_away_from_zero(n)` | ties away from zero | `2.35d` | `-2.35d` |
| `rescale_half_toward_zero(n)` | ties toward zero | `2.34d` | `-2.34d` |
| `rescale_bank(n)` | ties to even | `2.34d` | `-2.34d` |
| `rescale_up(n)` | toward +∞ | `2.35d` | `-2.34d` |
| `rescale_down(n)` | toward −∞ | `2.34d` | `-2.35d` |
| `rescale_away_from_zero(n)` | away from zero | `2.35d` | `-2.35d` |
| `rescale_toward_zero(n)` | toward zero (= `rescale`) | `2.34d` | `-2.34d` |

### Arithmetic straight to a scale — `div_round_*`, `mul_round_*`, `sqrt_round_*`

These divide, multiply or take a square root and land on **exactly** the requested scale in one call. The other
operand of `div_round_*` / `mul_round_*` is a `decimal` or an `int` — the same domain the `/` and `*` operators
accept, and `float` is refused here for the same reason it is refused there.

```go
(10d).div_round_bank(4d, 2).format("s")     // "2.50" — a money result carrying its cents
((10d) / (4d)).round_bank(2).format("s")    // "2.5"  — composition stops at "at most 2"
(7d).div_round_bank(3d, 2)                  // 2.33d
(1.005d).mul_round_bank(1d, 2)              // 1.00d — ties to even
(1.005d).mul_round_half_away_from_zero(1d, 2)   // 1.01d
(2d).sqrt_round_bank(4)                     // 1.4142d
(250000.00d).mul_round_bank(0.0425d, 2)     // 10625.00d — the shape a monetary line item wants
```

The scale is the difference that always shows against composing an operator with `round_*`. (The rounding
decision is also made against the exact result rather than an already-truncated scale-19 intermediate; that can
change the last digit in principle, but it is vanishingly rare — no divergence appeared in 4M randomized cases.)

All three families raise rather than answering a sentinel:

```go
(1d).div_round_bank(0d, 2)          // Error: division_by_zero
(-2d).sqrt_round_bank(2)            // Error: (sqrt_round_bank) square root of negative number
(1d).rescale_bank(20)               // Error: (rescale_bank) scale must be between 0 and 19
(1d).div_round_bank(3.0, 2)         // Error: (div_round_bank) argument first expects type decimal or int, got float
(1.23456d).rescale_bank(2.5)        // Error: (rescale_bank) argument scale must be a whole number, got 2.5
```

### Scale machinery

```go
decimal("1.50").scale()        // 2 — stored fractional digits
decimal("1.50").string()       // "1.50" — every rendering carries the scale
decimal("1.50").format("!")    // "1.50" trimmed to "1.5" — the '!' flag is the trimmed reading
(1.5d).rescale(3).format("s")  // "1.500" — widen the scale
(1.55d).rescale(1)             // 1.5d — narrowing TRUNCATES toward zero…
(-1.55d).rescale(1)            // -1.5d — …in both directions
(1.500d).canonical()           // 1.5d, scale 1 — the minimal equal representation
(1.5d).rescale(40)             // Error: (rescale) scale must be between 0 and 19
```

`rescale(n)` never rounds. When narrowing should round, reach for `rescale_*(n)` — it is the one spelling that
guarantees exactly n fractional digits *and* applies a policy.

### Other numeric members

```go
(-2.5d).abs()       // 2.5d
(-2.5d).sign()      // -1 (int); 0 for zero, 1 for positive
(2.5d).negate()     // -2.5d — the member spelling of unary minus
(2d).sqrt()         // 1.4142135623730950488d
(-1d).sqrt()        // Error: invalid_value: (sqrt) square root of negative number
(2d).pow(10)        // 1024d — integer exponent
(2d).pow(-1)        // 0.5d — a negative exponent is the reciprocal
(1.50d).pow(2)      // 2.2500d — the scale multiplies: scale(base) x n
(0d).pow(-1)        // Error: division_by_zero
(10d).pow(39)       // Error: (pow) overflow — 10d.pow(38) is the largest that fits
(1d).next_up()      // 2d — one unit in the last place of the CURRENT scale…
decimal("1.50").next_up()      // 1.51d — …so the step depends on scale
decimal("1.50").next_down()    // 1.49d
```

`pow(n)` takes an integer exponent and is what compound interest is written with:

```go
rate = 0.00416667d                                  // 5% APR, monthly
(1d + rate).pow(360)                                // 4.4677496530564604675d
(250000.00d) * rate * (1d + rate).pow(360)          // 4653.9096117251905340295d
```

### Predicates

```go
(0d).is_zero()           // true — scale-blind: 0.00d is zero too
(-1.5d).is_negative()    // true
(1.5d).is_positive()     // true
(1.5d).is_nan()          // false
(1.5d).is_inf()          // false, ALWAYS — the type has no Inf representation
```

`is_nan()` / `is_inf()` exist on every numeric type so generic code never type-switches; on `decimal`, `is_inf()`
is the constant-`false` end of that contract.

### Truthiness

Inequality with zero, member and free spelling alike; NaN raises:

```go
(0d).is_true()      // false
(0.5d).is_true()    // true
is_true(0d)         // false
1d / 0d             // Error: division_by_zero — there is no decimal NaN to ask about
```

### copy / freeze

Identity no-ops (a `decimal` is always immutable), kept for generic code: `(1.5d).copy()` → `1.5d`.

### format

Default rendering is the fixed-point form **at the value's own scale** — the same text `json.encode` emits. Verbs:
`f` / `F` (fixed, default precision 6, rounds half-away-from-zero), `s` (the explicit spelling of the default),
`%` (×100 with percent sign), `e` / `E` (scientific, default precision 6), `g` / `G` (the shorter of the two
readings, fixed and scientific). The `v` verb shows the literal form with the `d` suffix, scale included, so it
round-trips.

**Scientific output is exact.** The digits come from the decimal's own coefficient, so a precision past the
17th significant digit shows the value rather than the nearest `float64`:

```go
(2d/3d).format(".17e")      // "6.66666666666666667e-01"
(2d/3d).format("g")         // "0.6666666666666666666" — every digit
decimal("0.0000000000000000001").format("g")   // "1e-19" — scientific is the shorter reading here
decimal("123456789012345678901234567890").format(".25e")
                            // "1.2345678901234567890123457e+29"
(9.99d).format(".1e")       // "1.0e+01" — the carry moves the exponent
```

Ties round half away from zero, like the `f` verb: `(8.5d).format(".0e")` is `"9e+00"`. A `g` precision counts
**significant** digits, not fractional ones — `(2d/3d).format(".5g")` is `"0.66667"`.

The `!` flag is the trimmed reading, accepted **only** on the default verb and `s` — everywhere else a precision
already governs the digits, and `!` is a parse error:

```go
(1.500d).format()           // "1.500"
(1.500d).format("s")        // "1.500" — same thing, said explicitly
(1.500d).format("!")        // "1.5"
(1.500d).format("v")        // "1.500d"
(1234.5d).format(",.2f")    // "1,234.50"
(0.125d).format(".1%")      // "12.5%"
(1.5d).format("+.2f")       // "+1.50"
(1.500d).format(".2!f")     // Error: type decimal does not support format spec
```

### No sequence members

Scalars have no `len()`, no elements, and no `repeat`:

```go
(1.5d).repeat(2)    // Error: type decimal has no method repeat
```

## Conversions

`x.T()` answers a valid `T` or raises (kind `conversion`); `x.T(default)` answers the default instead;
`undefined.T(d)` → `d` covers maybe-missing data.

| target | behavior |
| --- | --- |
| `decimal` | identity |
| `int` | truncation toward zero for in-range values (documented resolution loss); out-of-range and NaN raise-or-default |
| `float` | nearest float64 — approximate by nature |
| `bool` | zero test; NaN raises-or-defaults |
| `string` / `runes` | rendering at the value's own scale, no `d` suffix (total — takes no default); `canonical()` first for the trimmed reading |
| `time` | unix timestamp as sec.frac, **exact to the nanosecond** |

```go
(1.9d).int()      // 1 — truncation toward zero
(-1.9d).int()     // -1
decimal("100000000000000000000").int()     // Error: cannot convert decimal to int
decimal("100000000000000000000").int(0)    // 0
(1.5d).float()    // 1.5
(2d).bool()       // true
(1.50d).string()  // "1.50" — the scale rides along
(1.50d).canonical().string()   // "1.5" — the trimmed reading
(1.5d).runes()    // u"1.5"
```

There is no `byte()` or `rune()` conversion — `int` is the sole gateway to the ordinal scalars
(`x.int().byte()`).

### time

A decimal in conversion context is a unix timestamp read as `seconds.fraction` — the **exact** path, unlike
`float`'s (dec128 is base-10, so every digit survives to the nanosecond):

```go
decimal("1704067200.123456789").time()    // time("2024-01-01T00:00:00.123456789Z")
```

## Migration notes

The `dec128` upgrade (v1.0.20 → v1.1.2) and the surface work built on it changed the following. Nothing here is
a deprecation — the old spellings are gone.

### Scale is preserved in every rendering

`json.encode`, `string()`, `runes()`, `fmt.println`, f-strings and the default `format()` verb all render at the
value's own scale. Previously all of them trimmed trailing zeros:

```go
json.encode(1.50d)     // "1.50"   — was "1.5"
(1.500d).string()      // "1.500"  — was "1.5"
f"{1.500d}"            // "1.500"  — was "1.5"
(1.500d).format("v")   // "1.500d" — was "1.5d"
```

`format("s")` is unchanged and is now the explicit spelling of the default. The new `!` flag is the trimmed
reading (`f"{1.500d:!}"` → `"1.5"`), and `canonical()` remains its value-level counterpart. **If you hash, sign,
diff or golden-test serialized decimals, the bytes have changed.**

### Equality against text inverted

Text comparison goes through the same rendering, so it now compares against the scale-preserving form:

```go
decimal("1.50") == "1.50"    // true  — was false
decimal("1.50") == "1.5"     // false — was true
decimal("1.50") == 1.5d      // true  — numeric equality is still scale-blind
```

### `1.5d` and `1.50d` are two constants

The compiler used to key decimal constants on their trimmed text, so the two collapsed into one and whichever
literal appeared **first in the file** decided the scale for both. They are now distinct, and a literal's scale
no longer depends on declaration order.

### Scientific notation parses, and scientific output is exact

`decimal("1e3")` answers `1000d` where it used to raise. Both the mantissa's digits and the exponent feed the
resulting scale, so `decimal("1.0e-3").scale()` is `4`.

The `e` / `E` / `g` / `G` verbs no longer route through `float64`, so their digits past the 17th significant
place have changed — they were previously fabricated from the nearest double:

```go
(2d/3d).format(".17e")    // "6.66666666666666667e-01" — was "6.66666666666666630e-01"
```

`g` / `G` now mean *the shorter of the two exact readings*, fixed and scientific, rather than a float64
shortest form; a `g` precision counts significant digits.

### A fractional scale or exponent raises

`rescale(2.5)`, `round_bank(2.5)` and the like silently truncated the argument to `2`. Every scale and exponent
argument now follows the same rule as every other count-shaped argument in the language — a lossless spelling is
accepted, a fractional one raises:

```go
(1.23456d).rescale(2.0)    // 1.23d
(1.23456d).rescale(2.5)    // Error: (rescale) argument scale must be a whole number, got 2.5
```

### Arithmetic rounds at the precision ceiling instead of raising

Up to dec128 v1.0.20 an operation whose exact result exceeded 19 decimal places answered `NaN`, which Kavun
turned into a raise. It now rounds instead — see [Rounding at the precision
ceiling](#rounding-at-the-precision-ceiling). This is what makes `pow` usable: `(1d + rate).pow(360)` raised
before and computes now. The trade is that a result no longer reports whether it was rounded, and a nonzero
product below 1e-19 becomes zero rather than raising.

Only a result whose **integer** part does not fit still raises.

### New members

Additive, nothing removed:

- `rescale_*(n)` — exactly *n* places, rounded, in all seven policies
- `div_round_*(y, n)`, `mul_round_*(y, n)`, `sqrt_round_*(n)` — arithmetic straight to a scale
- `pow(n)` — integer exponent, negative for the reciprocal

### Binary decoding refuses a NaN payload

`decimal` values decoded through the binary codec are now checked for NaN. dec128's `UnmarshalBinary` accepts a
NaN payload and reports no error, which made the codec the one door through which a NaN `decimal` could enter a
script; no operation in the language can produce one.

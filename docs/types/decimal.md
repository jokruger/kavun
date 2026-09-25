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
| `/` | division | up to 19 places; an exact quotient keeps its natural scale (`10d / 4d` → `2.5d`); `/ 0d` raises `division_by_zero` |
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
Addition keeps the wider operand's scale and multiplication adds scales. Division answers up to 19 places, but an
**exact** quotient keeps its natural scale — the dividend's scale minus the divisor's, or 0 — so it does not
sprout trailing zeros:

```go
(decimal("0.10") + decimal("0.20")).format("s")    // "0.30" — scale 2
(decimal("1.5") * decimal("2.00")).scale()          // 3 — scales add
(decimal("10.00") / 4d).format("s")                 // "2.50" — exact, natural scale 2 - 0
(1d / 2d).format("s")                               // "0.5"
(1d / 3d).scale()                                   // 19 — inexact, so every place is used
```

A `d`-suffixed literal carries the scale it was written with, and two literals that differ only in trailing
zeros are two different constants:

```go
(1.5d).scale()     // 1
(1.50d).scale()    // 2
1.5d == 1.50d      // true — numeric equality is scale-blind
```

Use `round(n, mode)` to pin a result to a scale, `rescale(n)` to change the scale without changing the value,
and `canonical()` to drop trailing zeros.

### Rounding at the precision ceiling

A decimal holds up to 38 significant digits with at most 19 fractional places. When an exact result does not
fit, the scale is reduced to the largest that still holds the integer part, and the discarded digits are
**truncated toward zero — not reported**:

```go
1d / 3d                             // 0.3333333333333333333d — 19 places, the rest dropped
decimal("0.0000000001") * decimal("0.0000000001")
                                    // 0.0000000000000000000d — the exact 1e-20 needs 20 places,
                                    // so it rounds to zero at scale 19
```

This is ordinary and expected, and it is the same rule as `int`'s `1 / 2 == 0`: a decimal moves in steps of its
smallest unit, and a result finer than that step cannot be held. No fixed-point type can represent a third, and
the same ceiling applies to a very small product. The result does not record that digits were lost; only a
result whose *integer* part does not fit raises (see [Overflow raises](#overflow-raises)).

The practical answer is to keep intermediates wide and round **once, at the boundary**, stating the scale and
the rounding mode at each point a result is pinned down: `round(2, mode)`, `div_round(y, 2, mode)` and the rest
of the [`*_round` members](#arithmetic-straight-to-a-scale--the-_round-members). A calculation carried at a
working scale well below 19 is unaffected by the ceiling. To check that a value is *already* on a grid without
changing it, `rescale(n)` raises instead of dropping a digit.

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
1.5d * 2      // 3.0d — the scale rides along
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

### Rounding modes

Every member that rounds takes the **mode as a string**, because in a real application the rounding rule is
configuration, not code. The seven names are Python's `decimal` modes, lowercased and without the `ROUND_`
prefix — so `down` means *toward zero*, as it does in Python, and the directions toward −∞ / +∞ are `floor` /
`ceiling`:

| mode | rule | `2.5d` → 0 | `-2.5d` → 0 | `2.345d` → 2 | `-2.345d` → 2 | Python |
| --- | --- | --- | --- | --- | --- | --- |
| `"ceiling"` | toward +∞ | `3d` | `-2d` | `2.35d` | `-2.34d` | `ROUND_CEILING` |
| `"floor"` | toward −∞ | `2d` | `-3d` | `2.34d` | `-2.35d` | `ROUND_FLOOR` |
| `"down"` | toward zero (truncate) | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_DOWN` |
| `"up"` | away from zero | `3d` | `-3d` | `2.35d` | `-2.35d` | `ROUND_UP` |
| `"half_down"` | nearest, ties toward zero | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_HALF_DOWN` |
| `"half_up"` | nearest, ties away from zero ("schoolbook") | `3d` | `-3d` | `2.35d` | `-2.35d` | `ROUND_HALF_UP` |
| `"half_even"` | nearest, ties to even (banker's) | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_HALF_EVEN` |

The name must match exactly — lowercase, no prefix. Anything else raises and lists the valid names; there is no
fallback mode:

```go
(1d).round(2, "bank")               // Error: (round) unknown rounding mode "bank", expected one of: ceiling, floor, down, up, half_down, half_up, half_even
(1d).round(2, "ROUND_HALF_EVEN")    // Error: (round) unknown rounding mode "ROUND_HALF_EVEN", …
(1d).round(2, 3)                    // Error: (round) argument mode expects type string, got int
```

### round, its twins, and rescale

`round(n, mode)` answers **exactly** `n` fractional places — the shape a monetary result needs, and what Python's
`round(Decimal, n)` answers. A negative `n` rounds to tens, hundreds, … and answers a whole number:

```go
rule = "half_even"                     // typically read from configuration
(2.345d).round(2, rule)                // 2.34d
(1.5d).round(2, rule).format("s")      // "1.50" — exactly two places, padded
(1234.5d).round(-2, rule)              // 1200d
(2.5d).round(-1, "half_up")            // 0d
(1d).round(20, "up")                   // Error: (round) places must be between -38 and 19
```

Each mode also has a **twin** that spells it in the name, for the common case where the rule is fixed:
`round_ceiling(n)`, `round_floor(n)`, `round_down(n)`, `round_up(n)`, `round_half_down(n)`, `round_half_up(n)`,
`round_half_even(n)`. A twin answers exactly what `round(n, "<mode>")` answers:

```go
(2.345d).round_half_up(2)     // 2.35d — same as (2.345d).round(2, "half_up")
(2.4d).round_ceiling(0)       // 3d — a direction applies to any fraction, not just ties
(-2.9d).round_down(0)         // -2d — toward zero
```

`rescale(n)` changes the scale **without changing the value**: it pads, and it narrows only when the digits it
drops are zeros. A value that would change raises — `round(n, mode)` is the spelling that changes it, and it
names how. That makes `rescale(n)` the check that a value is already on a grid:

```go
(1.5d).rescale(3).format("s")     // "1.500"
(1.500d).rescale(1).format("s")   // "1.5" — only zeros dropped
(1.55d).rescale(1)                // Error: (rescale) 1.55 has non-zero digits past scale 1; use round(n, mode) to round it
```

Two other grids:

```go
(2.37d).round_to_multiple(0.05d, "half_up")    // 2.35d — Swiss/Swedish cash rounding; the result has m's scale
(183.47d).round_to_multiple(10, "ceiling")     // 190d — "the next whole 10"
(1.09875d).round_significant(5, "half_even")   // 1.0988d — at most 5 significant digits, an FX quote
(9.99d).round_significant(2, "half_up")        // 10.0d — the carry adds a digit no scale can drop
(2.37d).round_to_multiple(0, "half_up")        // Error: (round_to_multiple) the multiple must be positive, got 0
```

There is **no plain `round()` with a default mode, no `floor()`, `ceil()` or `trunc()`**: every rounding states
its mode. Floor is `round_floor(0)`, ceiling `round_ceiling(0)`, truncation `round_down(n)`.

### Arithmetic straight to a scale — the `*_round` members

Every member whose last two arguments are `(scale, mode)` ends in `_round`. Each performs its operation with the
intermediate held **exactly** and makes **one** rounding decision, landing on exactly the requested scale — where
composing an operator with `round` rounds twice (once at the 19-place ceiling, once at `round`) and keeps the
operator's scale. The other operands are `decimal` or `int`, the domain the `/` and `*` operators accept; `float`
is refused here for the same reason it is refused there.

| member | computes | typical use |
| --- | --- | --- |
| `div_round(y, n, mode)` | `x / y` | a rate per period |
| `mul_round(y, n, mode)` | `x * y` | a line item |
| `mul_percent_round(r, n, mode)` | `x * r / 100` | VAT, fees — the division by 100 is a move of the point |
| `mul_add_round(b, c, n, mode)` | `x * b + c` | principal × rate + fee |
| `mul_div_round(b, c, n, mode)` | `x * b / c` | proration, day counts (31/365) |
| `sqrt_round(n, mode)` | `√x` | |
| `pow_round(k, n, mode)` | `x^k`, integer `k` | compounding — computed with guard digits |
| `nth_root_round(k, n, mode)` | `x^(1/k)` | the monthly factor of an annual rate |
| `pow_rational_round(p, q, n, mode)` | `x^(p/q)` | five months of a yearly rate |
| `exp_round(n, mode)`, `ln_round(n, mode)` | `eˣ`, `ln x` | continuous compounding |
| `log10_round(n, mode)`, `log2_round(n, mode)` | `log₁₀ x`, `log₂ x` | |

```go
(10d).div_round(4d, 2, "half_even")                 // 2.50d — a money result carrying its cents
10d / 4d                                            // 2.5d  — the operator keeps the natural scale
(7d).div_round(3, 2, "up")                          // 2.34d
(100.00d).mul_percent_round(7.5d, 2, "half_even")   // 7.50d
(250000.00d).mul_add_round(0.0425d, 12.345d, 2, "half_even")   // 10637.34d — 10637.345 is an exact tie
(1000d).mul_div_round(31, 365, 2, "half_up")        // 84.93d — one rounding on the exact numerator
(2d).sqrt_round(4, "half_even")                     // 1.4142d
(1.000164383561643836d).pow_round(3650, 19, "half_up")   // 1.8220289545384488980d — ten years of daily rests
(1.126825d).nth_root_round(12, 10, "half_up")       // 1.0099999977d
(1.06d).pow_rational_round(5, 12, 19, "half_up")    // 1.0245758393924285985d
(-8d).pow_rational_round(2, 6, 0, "half_up")        // -2d — 2/6 is reduced to 1/3 first
(1d).exp_round(10, "half_even")                     // 2.7182818285d
(1000d).log10_round(4, "half_even")                 // 3.0000d — an exact power of the base is exact
```

`mul_div_round` is the one of these that cannot be composed at all: scaling by a ratio whose decimal expansion
never ends (1/3, 31/365) has no exact decimal factor to multiply by first. `pow_round` against `pow`: `pow(k)`
truncates at the 19-place ceiling at every step of the power, `pow_round` carries guard digits and rounds once, so
over thousands of periods they can differ in the last places. `nth_root_round` and `pow_rational_round` are
**correctly rounded**; `exp_round`, `ln_round`, `log10_round` and `log2_round` are the only members that are not
exact — they are **faithfully rounded**, within one unit in the last place of the correctly rounded value.

All of them raise rather than answering a sentinel:

```go
(1d).div_round(0d, 2, "up")                  // Error: division_by_zero
(1d).mul_div_round(1, 0, 2, "up")            // Error: division_by_zero
(0d).pow_round(-1, 4, "half_up")             // Error: division_by_zero
(-2d).sqrt_round(2, "half_even")             // Error: (sqrt_round) square root of negative number
(0d).ln_round(4, "half_even")                // Error: (ln_round) argument outside the domain of the function
(-4d).nth_root_round(2, 2, "half_up")        // Error: (nth_root_round) argument outside the domain of the function
(4d).nth_root_round(16385, 2, "half_up")     // Error: (nth_root_round) degree must be at most 16384, got 16385
(1d).div_round(3.0, 2, "up")                 // Error: (div_round) argument first expects type decimal or int, got float
(1d).div_round(3d, 20, "up")                 // Error: (div_round) scale must be between 0 and 19
```

The degree limit (and the same limit on a reduced `p` or `q` of `pow_rational_round`) is a cost ceiling: 16384
covers 40 years of daily rests.

### Splitting an amount

`split` and `allocate` divide an amount into shares that **sum to exactly the amount** — a reconciliation is an
equality, not a tolerance. Both answer a new, mutable array.

| member | shares | the leftover quanta go to |
| --- | --- | --- |
| `split(k, n)` | `k` equal shares at scale `n` | the shares with the largest remainders, ties to the lowest index |
| `allocate(ratios, n)` | proportional to `ratios` (an array of `decimal`/`int`) | the same |
| `split_residual(k, n, index, mode)` | every share rounded with `mode` … | … except the one at `index`, which takes what is left |
| `allocate_residual(ratios, n, index, mode)` | the same, proportionally | the same |

The `_residual` forms are the convention of an amortization schedule or a syndicated facility: the last
installment, or the lead bank, absorbs the rounding. `index` counts from the end when negative, as an array index
does.

```go
(1000.00d).split(7, 2)
// [142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.85d, 142.85d]
(1000.00d).split_residual(7, 2, -1, "half_up")
// [142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.84d]
(100.00d).allocate([1, 1, 1], 2)                  // [33.34d, 33.33d, 33.33d]
(100.00d).allocate_residual([1, 1, 1], 2, -1, "half_up")   // [33.33d, 33.33d, 33.34d]
(0.05d).split(3, 2)                               // [0.02d, 0.02d, 0.01d]
(1000.00d).split(7, 2).sum() == 1000.00d          // true
```

The target scale must be at least the amount's own: splitting 1.005 into cents would have to round first, and
the split refuses to do that silently.

```go
(1.005d).split(3, 2)                               // Error: (split) scale 2 is below the amount's own scale 3; round the amount first
(1.005d).round(2, "half_even").split(3, 2)         // [0.34d, 0.33d, 0.33d]
(1d).split(0, 2)                                   // Error: (split) argument count must be positive, got 0
(1000.00d).split_residual(7, 2, 7, "half_up")      // Error: (split_residual) 7 out of range [0, 6]
(100d).allocate([1, -1], 2)                        // Error: (allocate) ratio at index 1 is negative
(100d).allocate([0, 0], 2)                         // Error: (allocate) ratios must not all be zero
(100d).allocate([], 2)                             // Error: (allocate) ratios must not be empty
```

### Exact helpers

```go
(-7.5d).quo_rem(2)               // [-3d, -1.5d] — quotient truncated toward zero, as Python's divmod on Decimal
q, r := (17d).quo_rem(5)         // q = 3d, r = 2d; q * y + r == x exactly
(1d).quo_rem(0)                  // Error: division_by_zero
(5.25d).scale_by_pow10(-2)       // 0.0525d — a percent to a fraction, as a move of the point, never rounded
(1.5d).scale_by_pow10(3)         // 1500d
(1d).scale_by_pow10(60)          // Error: (scale_by_pow10) overflow
(1.5d).copy_sign(-2)             // -1.5d — the magnitude of x, the sign of y
(3d).clamp(0, 2.00d)             // 2.00d — numeric comparison; the answer keeps its own scale
(1.5d).clamp(0, 2.00d)           // 1.5d
(1d).clamp(2, 0)                 // Error: (clamp) lower bound 2 is above upper bound 0
```

### Shape

For checking a value against the column it has to live in, where it is computed rather than at write time:

```go
(1.000d).is_integer()             // true — nothing after the point but zeros
(1.5d).is_integer()               // false
(1.50d).significant_digits()      // 3 — trailing zeros are digits of the representation
(-123.45d).integer_digits()       // 3
(0.99d).integer_digits()          // 0
(1.50d).can_fit(3, 1)             // true — would NUMERIC(3, 1) hold it exactly? trailing zeros need no place
(123.456d).can_fit(5, 2)          // false
(1d).can_fit(2, 3)                // Error: (can_fit) scale must be between 0 and the precision 2
```

### Scale machinery

```go
decimal("1.50").scale()        // 2 — stored fractional digits
decimal("1.50").string()       // "1.50" — every rendering carries the scale
decimal("1.50").format("!")    // "1.5" — the '!' flag is the trimmed reading
(1.5d).rescale(3).format("s")  // "1.500" — widen the scale
(1.500d).canonical()           // 1.5d, scale 1 — the minimal equal representation
(1.5d).rescale(40)             // Error: (rescale) scale must be between 0 and 19
```

### Other numeric members

```go
(-2.5d).abs()       // 2.5d
(-2.5d).sign()      // -1 (int); 0 for zero, 1 for positive
(2.5d).negate()     // -2.5d — the member spelling of unary minus
(2d).sqrt()         // 1.4142135623730950488d
(-1d).sqrt()        // Error: invalid_value: (sqrt) square root of negative number
(2d).pow(10)        // 1024d — integer exponent
(2d).pow(-1)        // 0.5000000000000000000d — a negative exponent is the reciprocal, taken at 19 places
(1.50d).pow(2)      // 2.2500d — the scale multiplies: scale(base) x n
(0d).pow(-1)        // Error: division_by_zero
(10d).pow(39)       // Error: (pow) overflow — 10d.pow(38) is the largest that fits
(1d).next_up()      // 2d — one unit in the last place of the CURRENT scale…
decimal("1.50").next_up()      // 1.51d — …so the step depends on scale
decimal("1.50").next_down()    // 1.49d
```

`pow(n)` takes an integer exponent and truncates at the 19-place ceiling as it goes, like the operators; for a
long compounding chain, `pow_round(n, scale, mode)` carries guard digits and rounds once (see
[the `*_round` members](#arithmetic-straight-to-a-scale--the-_round-members)):

```go
rate = 0.00416667d                                  // 5% APR, monthly
(1d + rate).pow(360)                                // 4.4677496530564605163d
(250000.00d) * rate * (1d + rate).pow(360)          // 4653.9096117251905848629d
(1d + rate).pow_round(360, 10, "half_even")         // 4.4677496531d — guard digits, one rounding
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

A precision past the 19-place ceiling pads with zeros — a decimal has no digits there, so writing them is exact:

```go
(1d).format(".25f")         // "1.0000000000000000000000000"
```

The `f`, `%` and `e` verbs round **for display**, always half away from zero. A business rounding rule belongs in
the value, before formatting — `f"{total.round(2, rule):,.2f}"` — so the stored number and the printed one agree
whatever the rule is.

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

The `dec128` upgrade (v1.0.20 → v1.4.0) and the surface work built on it changed the following. Nothing here is
a deprecation — the old spellings are gone.

### Rounding takes a mode string, named as in Python

Every rounding member now takes the rounding mode as a string argument, or spells it as a `round_<mode>` suffix,
using Python's `decimal` names (see [Rounding modes](#rounding-modes)). The per-mode member families are gone.

**Two old names keep their spelling but change meaning.** `round_down` and `round_up` used to be floor and
ceiling; they now mean toward zero and away from zero, as Python's `ROUND_DOWN` / `ROUND_UP` do. A script that
calls them still runs, with different results on negative values (`round_down`) and on every non-exact value
(`round_up`) — check each call:

| before | now |
| --- | --- |
| `round_down(n)` — floor | `round_floor(n)` |
| `round_up(n)` — ceiling | `round_ceiling(n)` |
| `round_toward_zero(n)`, `trunc(n)` | `round_down(n)` |
| `round_away_from_zero(n)` | `round_up(n)` |
| `round_half_toward_zero(n)` | `round_half_down(n)` |
| `round_half_away_from_zero(n)` | `round_half_up(n)` |
| `round_bank(n)` | `round_half_even(n)` |
| `rescale_<policy>(n)` | `round(n, mode)` or `round_<mode>(n)` |
| `div_round_<policy>(y, n)`, `mul_round_<policy>(y, n)`, `sqrt_round_<policy>(n)` | `div_round(y, n, mode)`, `mul_round(y, n, mode)`, `sqrt_round(n, mode)` |

### `round` answers exactly n places

The rounding members used to answer *at most* `n` places and leave a shorter value alone. They now answer
**exactly** `n`, as Python's `round(Decimal, n)` does:

```go
(1.5d).round_half_even(2).format("s")    // "1.50" — round_bank(2) answered "1.5"
```

A negative `n` is new: it rounds to tens, hundreds, …

### `rescale(n)` no longer truncates

`rescale(n)` used to drop digits toward zero when narrowing. It is now lossless and **raises** instead of
changing the value; `round(n, "down")` is the truncating spelling:

```go
(1.55d).rescale(1)          // Error: (rescale) 1.55 has non-zero digits past scale 1 — was 1.5d
(1.55d).round(1, "down")    // 1.5d
```

### An exact quotient keeps its natural scale

`/` used to answer every quotient at 19 places. An exact quotient now keeps the dividend's scale minus the
divisor's (or 0); an inexact one still uses all 19. `sqrt()` of a perfect square does the same:

```go
(decimal("10.00") / 4d).format("s")    // "2.50" — was "2.5000000000000000000"
(4d).sqrt().format("v")                // "2d"
```

### A format precision past 19 places pads

`(1d).format(".25f")` used to stop at 19 places; it now pads to the 25 asked for.

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

`rescale(2.5)`, `round_half_even(2.5)` and the like silently truncated the argument to `2`. Every scale and exponent
argument now follows the same rule as every other count-shaped argument in the language — a lossless spelling is
accepted, a fractional one raises:

```go
(1.23000d).rescale(2.0)    // 1.23d
(1.23456d).rescale(2.5)    // Error: (rescale) argument scale must be a whole number, got 2.5
```

### Arithmetic truncates at the precision ceiling instead of raising

Up to dec128 v1.0.20 an operation whose exact result exceeded 19 decimal places answered `NaN`, which Kavun
turned into a raise. It now truncates toward zero instead — see [Rounding at the precision
ceiling](#rounding-at-the-precision-ceiling). This is what makes `pow` usable: `(1d + rate).pow(360)` raised
before and computes now. The trade is that a result no longer reports whether it was rounded, and a nonzero
product below 1e-19 becomes zero rather than raising.

Only a result whose **integer** part does not fit still raises.

### New members

- `round(n, mode)` and the seven `round_<mode>(n)` twins; `round_to_multiple(m, mode)`, `round_significant(k, mode)`
- `div_round`, `mul_round`, `mul_percent_round`, `mul_add_round`, `mul_div_round`, `sqrt_round`, `pow_round`,
  `nth_root_round`, `pow_rational_round`, `exp_round`, `ln_round`, `log10_round`, `log2_round`
- `split`, `split_residual`, `allocate`, `allocate_residual`
- `quo_rem`, `scale_by_pow10`, `copy_sign`, `clamp`
- `is_integer`, `significant_digits`, `integer_digits`, `can_fit`
- `pow(n)` — integer exponent, negative for the reciprocal

### Binary decoding refuses a NaN payload

`decimal` values decoded through the binary codec are now checked for NaN. dec128's `UnmarshalBinary` accepts a
NaN payload and reports no error, which made the codec the one door through which a NaN `decimal` could enter a
script; no operation in the language can produce one.

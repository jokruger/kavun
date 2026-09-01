# decimal

128-bit base-10 decimals (dec128) for exact monetary arithmetic.

## Overview

`decimal` is a 128-bit decimal floating-point number: up to 38 significant digits with a fractional scale of 0 to
19 digits. Arithmetic is exact in base 10 — `0.1d + 0.2d == 0.3d` is `true`, where the float spelling is famously
`false`. Money and anything else that must be exact in base 10 belongs in `decimal`; measurements and physics
belong in [`float`](float.md).

`decimal` belongs to the numeric family alongside [`int`](int.md) and `float`: it pairs with `int` in arithmetic
and compares exactly against both. It has no Inf representation at all (`is_inf()` is constantly `false`), and a
NaN exists only as an interrogable error state — never as a value Kavun arithmetic is supposed to produce (see
[NaN as an error state](#nan-as-an-error-state)).

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
decimal("1e3")             // Error: scientific notation does not parse (yet)
```

## Arithmetic and operators

| operator | meaning | notes |
| --- | --- | --- |
| `+` `-` `*` | add, subtract, multiply | exact; raise `decimal overflow` past 38 digits |
| `/` | division | full available precision (scale 19); `/ 0d` currently answers NaN — see below |
| `%` | remainder | `10d % 3d` → `1d` |
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

(These examples parse from text: a `d`-suffixed literal also carries its written scale, but numerically equal
constants in one program share one stored representation, so a scale-sensitive literal can be
surprising — parse from text where the scale matters.)

Use `rescale()` / `canonical()` or the rounding family to tidy a result for storage or display.

### Overflow raises

A result that no longer fits 128 bits raises a catchable `invalid_value` error:

```go
x = decimal("99999999999999999999999999999999999999")
x + x    // 199999999999999999999999999999999999998d — still fits
x * x    // Error: decimal overflow
```

### Division by zero — a known gap

`x / 0d` and `x % 0d` currently answer a **NaN decimal instead of raising** (slated to become a raise). Until then,
check quotients where a zero divisor is possible:

```go
q = 1d / 0d          // decimal("NaN") — no raise today
q.is_nan()           // true
q.error_details()    // error("division by zero")
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

dec128 has a NaN representation, but Kavun treats it as an error state: parses never produce it (they raise), and
the only arithmetic that reaches it today is the division-by-zero gap above (plus `sqrt` of a negative). Once
present it propagates through arithmetic, so interrogate suspect values:

```go
n = 1d / 0d          // decimal("NaN")
n + 1d               // decimal("NaN") — propagates silently today
n.is_nan()           // true
n.error_details()    // error("division by zero") — why this NaN exists
n.is_true()          // Error: decimal NaN is neither true nor false in a boolean context
n.int()              // Error: cannot convert decimal to int
n.bool()             // Error: cannot convert decimal to bool
n.sign()             // 0
(-1d).sqrt()         // decimal("NaN")
```

In comparisons NaN follows the numeric family's total order: the unique minimum, equal only to another NaN
(`n == n` → `true`, `n < -1d` → `true`), so sorting stays deterministic.

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

`bool` / `byte` / `rune` widen to their integer value. Equality against a `string` compares the canonical
(trailing-zero-trimmed) text form; ordering against text raises:

```go
decimal("1.50") == "1.5"     // true — canonical form
decimal("1.50") == "1.50"    // false — "1.50" is not the canonical rendering
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
is `round_half_away_from_zero(n)`, floor is `round_down(0)`, ceiling is `round_up(0)`.

### Scale machinery

```go
decimal("1.50").scale()        // 2 — stored fractional digits
decimal("1.50").format("s")    // "1.50" — the scale-preserving rendering
decimal("1.50").string()       // "1.5" — canonical rendering trims trailing zeros
(1.5d).rescale(3).format("s")  // "1.500" — widen the scale
(1.55d).rescale(1)             // 1.5d — narrowing TRUNCATES toward zero…
(-1.55d).rescale(1)            // -1.5d — …in both directions
(1.500d).canonical()           // 1.5d, scale 1 — the minimal equal representation
(1.5d).rescale(40)             // Error: (rescale) scale must be between 0 and 19
```

`rescale(n)` never rounds — apply a rounding member first when narrowing should round
(`x.round_half_away_from_zero(2)` already leaves scale 2).

### Other numeric members

```go
(-2.5d).abs()       // 2.5d
(-2.5d).sign()      // -1 (int); 0 for zero and NaN, 1 for positive
(2.5d).negate()     // -2.5d — the member spelling of unary minus
(2d).sqrt()         // 1.4142135623730950488d
(-1d).sqrt()        // decimal("NaN") — interrogate with is_nan()
(1d).next_up()      // 2d — one unit in the last place of the CURRENT scale…
decimal("1.50").next_up()      // 1.51d — …so the step depends on scale
decimal("1.50").next_down()    // 1.49d
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
(1d / 0d).is_true() // Error: decimal NaN is neither true nor false in a boolean context
```

### copy / freeze

Identity no-ops (a `decimal` is always immutable), kept for generic code: `(1.5d).copy()` → `1.5d`.

### format

Default rendering is the canonical fixed-point form (trailing zeros trimmed). Verbs: `f` / `F` (fixed, default
precision 6, rounds half-away-from-zero), `s` (scale-preserving), `%` (×100 with percent sign), `e` / `E` / `g` /
`G` (scientific/shortest — via float64, adequate for display, not full precision). The `v` verb shows the literal
form with the `d` suffix:

```go
(1234.5d).format(",.2f")    // "1,234.50"
decimal("1.50").format("s") // "1.50"
(0.125d).format(".1%")      // "12.5%"
(1.5d).format("+.2f")       // "+1.50"
(1234.5d).format("v")       // "1234.5d"
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
| `string` / `runes` | canonical rendering, trailing zeros trimmed, no `d` suffix (total — takes no default) |
| `time` | unix timestamp as sec.frac, **exact to the nanosecond** |

```go
(1.9d).int()      // 1 — truncation toward zero
(-1.9d).int()     // -1
decimal("100000000000000000000").int()     // Error: cannot convert decimal to int
decimal("100000000000000000000").int(0)    // 0
(1.5d).float()    // 1.5
(2d).bool()       // true
(1.50d).string()  // "1.5"
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

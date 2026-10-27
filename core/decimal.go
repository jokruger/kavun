package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/seq"
)

const decimalTypeName = "decimal"

func NewStaticDecimalValue(d *dec128.Dec128) Value {
	return Value{Type: value.Decimal, Immutable: true, Ptr: unsafe.Pointer(d)}
}

func NewDecimalValue(d dec128.Dec128) Value {
	return Value{Type: value.Decimal, Immutable: true, Ptr: unsafe.Pointer(&d)}
}

// 128-bit base-10 decimals (dec128) for exact monetary arithmetic.
//
// ## Overview
//
// `decimal` is a 128-bit decimal floating-point number: up to 38 significant digits with a fractional scale of 0 to
// 19 digits. Arithmetic is exact in base 10 — `0.1d + 0.2d == 0.3d` is `true`, where the float spelling is famously
// `false`. Money and anything else that must be exact in base 10 belongs in `decimal`; measurements and physics
// belong in [`float`](float.md).
//
// `decimal` belongs to the numeric family alongside [`int`](int.md) and `float`: it pairs with `int` in arithmetic
// and compares exactly against both. It has no Inf representation at all (`is_inf()` is constantly `false`), and a
// NaN exists only as an error state a **host** can hand in — no operation in the language produces one, because
// every path that would raises instead (see [NaN as an error state](#nan-as-an-error-state)).
//
// ## Literals and construction
//
// A base-10 numeric literal with a `d` suffix is a decimal:
//
// ```go
// price = 19.99d
// two = 2d
// neg = -2.5d        // unary minus, exact
// big = 1.5e3d       // => 1500d — exponent form, same value and scale as decimal("1.5e3")
// ```
//
// The free `decimal(x)` constructor (equivalently `x.decimal()`) converts and parses; the member's optional
// `x.decimal(default)` rescues bad data:
//
// ```go
// decimal()                  // => 0d — the zero value
// decimal("0.1")             // => 0.1d — parsing text is the lossless entry point
// decimal(3)                 // => 3d
// decimal(1.5)               // => 1.5d — from float, shortest decimal reading
// decimal("abc")             // Error: conversion: cannot convert string to decimal — parse raises, no NaN sentinel
// "abc".decimal(0d)          // => 0d — the member's default rescues bad data
// undefined.decimal(0d)      // => 0d — the maybe-missing form
// decimal("inf")             // Error: conversion: cannot convert string to decimal — the type has no Inf
// decimal("1e3")             // => 1000d — scientific notation parses, in either case ("1E3" too)
// ```
//
// ### Scientific notation and scale
//
// Both the mantissa's own digits and the exponent contribute to the resulting scale, so the notation carries
// precision rather than discarding it:
//
// ```go
// decimal("1.0e-3").format("s")   // => "0.0010" — scale 4: mantissa scale 1, shifted 3 more places
// decimal("2.5E2").scale()        // => 0 — 250d; the exponent absorbed the fractional digit
// decimal("1e-25")                // Error: conversion: cannot convert string to decimal — past the scale-19 ceiling
// ```
//
// ## Arithmetic and operators
//
// | operator | meaning | notes |
// | --- | --- | --- |
// | `+` `-` `*` | add, subtract, multiply | exact; raise `invalid_value` past 38 digits |
// | `/` | division | up to 19 places; an exact quotient keeps its natural scale (`10d / 4d` → `2.5d`); `/ 0d` raises `division_by_zero` |
// | `%` | remainder | `10d % 3d` → `1d`; `% 0d` raises `division_by_zero` |
// | `-x` | negation | exact, never overflows |
//
// ```go
// 0.1d + 0.2d            // => 0.3d — exact
// 0.1d + 0.2d == 0.3d    // => true
// 0.1 + 0.2 == 0.3       // => false — the float contrast
// 1d / 3d                // => 0.3333333333333333333d — 19 fractional digits
// ```
//
// ### Scale of a result
//
// Each value carries a scale (count of stored fractional digits, observable via `scale()` and `format("s")`).
// Addition keeps the wider operand's scale and multiplication adds scales. Division answers up to 19 places, but an
// **exact** quotient keeps its natural scale — the dividend's scale minus the divisor's, or 0 — so it does not
// sprout trailing zeros:
//
// ```go
// (decimal("0.10") + decimal("0.20")).format("s")    // => "0.30" — scale 2
// (decimal("1.5") * decimal("2.00")).scale()          // => 3 — scales add
// (decimal("10.00") / 4d).format("s")                 // => "2.50" — exact, natural scale 2 - 0
// (1d / 2d).format("s")                               // => "0.5"
// (1d / 3d).scale()                                   // => 19 — inexact, so every place is used
// ```
//
// A `d`-suffixed literal carries the scale it was written with, and two literals that differ only in trailing
// zeros are two different constants:
//
// ```go
// (1.5d).scale()     // => 1
// (1.50d).scale()    // => 2
// 1.5d == 1.50d      // => true — numeric equality is scale-blind
// ```
//
// Use `round(n, mode)` to pin a result to a scale, `rescale(n)` to change the scale without changing the value,
// and `canonical()` to drop trailing zeros.
//
// ### Rounding at the precision ceiling
//
// A decimal holds up to 38 significant digits with at most 19 fractional places. When an exact result does not
// fit, the scale is reduced to the largest that still holds the integer part, and the discarded digits are
// **truncated toward zero — not reported**:
//
// ```go
//
//	1d / 3d                             // => 0.3333333333333333333d — 19 places, the rest dropped
//	decimal("0.0000000001") * decimal("0.0000000001")
//	                                    // 0.0000000000000000000d — the exact 1e-20 needs 20 places,
//	                                    // so it rounds to zero at scale 19
//
// ```
//
// This is ordinary and expected, and it is the same rule as `int`'s `1 / 2 == 0`: a decimal moves in steps of its
// smallest unit, and a result finer than that step cannot be held. No fixed-point type can represent a third, and
// the same ceiling applies to a very small product. The result does not record that digits were lost; only a
// result whose *integer* part does not fit raises (see [Overflow raises](#overflow-raises)).
//
// The practical answer is to keep intermediates wide and round **once, at the boundary**, stating the scale and
// the rounding mode at each point a result is pinned down: `round(2, mode)`, `div_round(y, 2, mode)` and the rest
// of the [`*_round` members](#arithmetic-straight-to-a-scale--the-_round-members). A calculation carried at a
// working scale well below 19 is unaffected by the ceiling. To check that a value is *already* on a grid without
// changing it, `rescale(n)` raises instead of dropping a digit.
//
// ### Overflow raises
//
// A result that no longer fits 128 bits raises a catchable `invalid_value` error:
//
// ```go
// x = decimal("99999999999999999999999999999999999999")
// x + x    // => 199999999999999999999999999999999999998d — still fits
// x * x    // Error: invalid_value: (*) overflow
// ```
//
// ### Division by zero
//
// `x / 0d` and `x % 0d` raise `division_by_zero`, the same kind `int` answers for `1 / 0`. There is nothing to
// check after the division:
//
// ```go
// 1d / 0d              // Error: division_by_zero: division by zero
// 0d / 0d              // Error: division_by_zero: division by zero
// 1d % 0d              // Error: division_by_zero: division by zero
// ```
//
// ### decimal never mixes with float
//
// Arithmetic between `decimal` and `float` raises — there is no silently-chosen loser of precision. Convert
// explicitly on the side you mean:
//
// ```go
// 1.5d + 1.0    // Error: invalid_binary_operator: decimal + float
// 1.5 + 1d      // Error: invalid_binary_operator: float + decimal
// 1.5d + 1      // => 2.5d — int pairs fine
// 1.5d * 2      // => 3.0d — the scale rides along
// 1.5d + (1.0).decimal()    // => 2.5d — explicit, decimal wins
// ```
//
// Comparisons across the pair are allowed (and exact) — see below.
//
// ## NaN as an error state
//
// dec128 signals every failure the same way — it answers NaN — and Kavun treats that as an **error state, not a
// value**. Unlike an IEEE NaN it even compares equal to itself, so it behaves as an ordinary number everywhere
// downstream and a wrong answer would propagate silently through a whole script.
//
// So **no operation in the language produces one**. Every path that would raises instead:
//
// ```go
// x := 99999999999999999999d   // 20 significant digits
// decimal("abc")       // Error: conversion — the parse
// 1d / 0d              // Error: division_by_zero
// 0d % 0d              // Error: division_by_zero
// (-1d).sqrt()         // Error: invalid_value: (sqrt) square root of negative number
// x * x                // Error: invalid_value: (*) overflow  — past the 128-bit coefficient
// (1e39).decimal()     // Error: conversion — the float is finite but too large for dec128
// ```
//
// That mirrors `float`, whose arithmetic already raises on a NaN or ±Inf result.
//
// `is_nan()` and `error_details()` remain because a **host** can bind a NaN decimal through the embedding API —
// nothing in the language can. For a decimal a script produced, `is_nan()` is always `false` and
// `error_details()` is always `undefined`. On a host-supplied NaN:
//
// ```go norun — needs a NaN decimal, which only a host can bind
// n.is_nan()           // => true
// n.error_details()    // => error("...") — why this NaN exists
// n.is_true()          // Error: invalid_value: decimal NaN is neither true nor false in a boolean context
// n.int()              // Error: conversion: cannot convert decimal to int
// n.sign()             // => 0
// ```
//
// In comparisons such a NaN follows the numeric family's total order: the unique minimum, equal only to another
// NaN, so sorting stays deterministic.
//
// ## Comparisons and cross-type pairing
//
// `==` and the orderings are **exact mathematical comparisons**. Trailing zeros never matter — equality compares
// numeric value, not scale:
//
// ```go
// 2.50d == 2.5d      // => true — scale is representation, not identity
// 1d == 1            // => true
// 1d < 1.5           // => true — float comparison is allowed (only arithmetic is refused)
// 0.1d == 0.1        // => false — float 0.1 is really 0.1000000000000000055511…
// 0.1d < 0.1         // => true — exactly below the float's true value
// (0.1).decimal() == 0.1d    // => true — the float's shortest decimal reading is 0.1
// ```
//
// `bool` / `byte` / `rune` widen to their integer value. Equality against a `string` compares the decimal's own text
// form, which carries its scale; ordering against text raises:
//
// ```go
// decimal("1.50") == "1.50"    // => true — the scale is part of the rendering
// decimal("1.50") == "1.5"     // => false — a different text, though the same NUMBER
// decimal("1.50") == 1.5d      // => true — numeric equality is scale-blind, text equality is not
// ```
//
// ## Rounding modes
//
// Every member that rounds takes the **mode as a string**, because in a real application the rounding rule is
// configuration, not code. The seven names are Python's `decimal` modes, lowercased and without the `ROUND_`
// prefix — so `down` means *toward zero*, as it does in Python, and the directions toward −∞ / +∞ are `floor` /
// `ceiling`:
//
// | mode | rule | `2.5d` → 0 | `-2.5d` → 0 | `2.345d` → 2 | `-2.345d` → 2 | Python |
// | --- | --- | --- | --- | --- | --- | --- |
// | `"ceiling"` | toward +∞ | `3d` | `-2d` | `2.35d` | `-2.34d` | `ROUND_CEILING` |
// | `"floor"` | toward −∞ | `2d` | `-3d` | `2.34d` | `-2.35d` | `ROUND_FLOOR` |
// | `"down"` | toward zero (truncate) | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_DOWN` |
// | `"up"` | away from zero | `3d` | `-3d` | `2.35d` | `-2.35d` | `ROUND_UP` |
// | `"half_down"` | nearest, ties toward zero | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_HALF_DOWN` |
// | `"half_up"` | nearest, ties away from zero ("schoolbook") | `3d` | `-3d` | `2.35d` | `-2.35d` | `ROUND_HALF_UP` |
// | `"half_even"` | nearest, ties to even (banker's) | `2d` | `-2d` | `2.34d` | `-2.34d` | `ROUND_HALF_EVEN` |
//
// The name must match exactly — lowercase, no prefix. Anything else raises and lists the valid names; there is no
// fallback mode:
//
// ```go
// (1d).round(2, "bank")               // Error: invalid_value: (round) unknown rounding mode "bank", expected one of: ceiling, floor, down, up, half_down, half_up, half_even
// (1d).round(2, "ROUND_HALF_EVEN")    // Error: invalid_value: (round) unknown rounding mode "ROUND_HALF_EVEN", …
// (1d).round(2, 3)                    // Error: invalid_argument_type: (round) argument mode expects type string, got int
// ```
//
// ## Arithmetic straight to a scale — the `*_round` members
//
// Every member whose last two arguments are `(scale, mode)` ends in `_round`. Each performs its operation with the
// intermediate held **exactly** and makes **one** rounding decision, landing on exactly the requested scale — where
// composing an operator with `round` rounds twice (once at the 19-place ceiling, once at `round`) and keeps the
// operator's scale. The other operands are `decimal` or `int`, the domain the `/` and `*` operators accept; `float`
// is refused here for the same reason it is refused there.
//
// | member | computes | typical use |
// | --- | --- | --- |
// | `div_round(y, n, mode)` | `x / y` | a rate per period |
// | `mul_round(y, n, mode)` | `x * y` | a line item |
// | `mul_percent_round(r, n, mode)` | `x * r / 100` | VAT, fees — the division by 100 is a move of the point |
// | `mul_add_round(b, c, n, mode)` | `x * b + c` | principal × rate + fee |
// | `mul_div_round(b, c, n, mode)` | `x * b / c` | proration, day counts (31/365) |
// | `sqrt_round(n, mode)` | `√x` | |
// | `pow_round(k, n, mode)` | `x^k`, integer `k` | compounding — computed with guard digits |
// | `nth_root_round(k, n, mode)` | `x^(1/k)` | the monthly factor of an annual rate |
// | `pow_rational_round(p, q, n, mode)` | `x^(p/q)` | five months of a yearly rate |
// | `exp_round(n, mode)`, `ln_round(n, mode)` | `eˣ`, `ln x` | continuous compounding |
// | `log10_round(n, mode)`, `log2_round(n, mode)` | `log₁₀ x`, `log₂ x` | |
//
// ```go
// (10d).div_round(4d, 2, "half_even")                 // => 2.50d — a money result carrying its cents
// 10d / 4d                                            // => 2.5d — the operator keeps the natural scale
// (7d).div_round(3, 2, "up")                          // => 2.34d
// (100.00d).mul_percent_round(7.5d, 2, "half_even")   // => 7.50d
// (250000.00d).mul_add_round(0.0425d, 12.345d, 2, "half_even")   // => 10637.34d — 10637.345 is an exact tie
// (1000d).mul_div_round(31, 365, 2, "half_up")        // => 84.93d — one rounding on the exact numerator
// (2d).sqrt_round(4, "half_even")                     // => 1.4142d
// (1.000164383561643836d).pow_round(3650, 19, "half_up")   // => 1.8220289545384488980d — ten years of daily rests
// (1.126825d).nth_root_round(12, 10, "half_up")       // => 1.0099999977d
// (1.06d).pow_rational_round(5, 12, 19, "half_up")    // => 1.0245758393924285985d
// (-8d).pow_rational_round(2, 6, 0, "half_up")        // => -2d — 2/6 is reduced to 1/3 first
// (1d).exp_round(10, "half_even")                     // => 2.7182818285d
// (1000d).log10_round(4, "half_even")                 // => 3.0000d — an exact power of the base is exact
// ```
//
// `mul_div_round` is the one of these that cannot be composed at all: scaling by a ratio whose decimal expansion
// never ends (1/3, 31/365) has no exact decimal factor to multiply by first. `pow_round` against `pow`: `pow(k)`
// truncates at the 19-place ceiling at every step of the power, `pow_round` carries guard digits and rounds once, so
// over thousands of periods they can differ in the last places. `nth_root_round` and `pow_rational_round` are
// **correctly rounded**; `exp_round`, `ln_round`, `log10_round` and `log2_round` are the only members that are not
// exact — they are **faithfully rounded**, within one unit in the last place of the correctly rounded value.
//
// All of them raise rather than answering a sentinel:
//
// ```go
// (1d).div_round(0d, 2, "up")                  // Error: division_by_zero
// (1d).mul_div_round(1, 0, 2, "up")            // Error: division_by_zero
// (0d).pow_round(-1, 4, "half_up")             // Error: division_by_zero
// (-2d).sqrt_round(2, "half_even")             // Error: invalid_value: (sqrt_round) square root of negative number
// (0d).ln_round(4, "half_even")                // Error: invalid_value: (ln_round) argument outside the domain of the function
// (-4d).nth_root_round(2, 2, "half_up")        // Error: invalid_value: (nth_root_round) argument outside the domain of the function
// (4d).nth_root_round(16385, 2, "half_up")     // Error: invalid_value: (nth_root_round) degree must be at most 16384, got 16385
// (1d).div_round(3.0, 2, "up")                 // Error: invalid_argument_type: (div_round) argument first expects type decimal or int, got float
// (1d).div_round(3d, 20, "up")                 // Error: invalid_value: (div_round) scale must be between 0 and 19
// ```
//
// The degree limit (and the same limit on a reduced `p` or `q` of `pow_rational_round`) is a cost ceiling: 16384
// covers 40 years of daily rests.
//
// ## Splitting an amount
//
// `split` and `allocate` divide an amount into shares that **sum to exactly the amount** — a reconciliation is an
// equality, not a tolerance. Both answer a new, mutable array.
//
// | member | shares | the leftover quanta go to |
// | --- | --- | --- |
// | `split(k, n)` | `k` equal shares at scale `n` | the shares with the largest remainders, ties to the lowest index |
// | `allocate(ratios, n)` | proportional to `ratios` (an array of `decimal`/`int`) | the same |
// | `split_residual(k, n, index, mode)` | every share rounded with `mode` … | … except the one at `index`, which takes what is left |
// | `allocate_residual(ratios, n, index, mode)` | the same, proportionally | the same |
//
// The `_residual` forms are the convention of an amortization schedule or a syndicated facility: the last
// installment, or the lead bank, absorbs the rounding. `index` counts from the end when negative, as an array index
// does.
//
// ```go
// (1000.00d).split(7, 2)
// // [142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.85d, 142.85d]
// (1000.00d).split_residual(7, 2, -1, "half_up")
// // [142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.86d, 142.84d]
// (100.00d).allocate([1, 1, 1], 2)                  // => [33.34d, 33.33d, 33.33d]
// (100.00d).allocate_residual([1, 1, 1], 2, -1, "half_up")   // => [33.33d, 33.33d, 33.34d]
// (0.05d).split(3, 2)                               // => [0.02d, 0.02d, 0.01d]
// (1000.00d).split(7, 2).sum() == 1000.00d          // => true
// ```
//
// The target scale must be at least the amount's own: splitting 1.005 into cents would have to round first, and
// the split refuses to do that silently.
//
// ```go
// (1.005d).split(3, 2)                               // Error: invalid_value: (split) scale 2 is below the amount's own scale 3; round the amount first
// (1.005d).round(2, "half_even").split(3, 2)         // => [0.34d, 0.33d, 0.33d]
// (1d).split(0, 2)                                   // Error: invalid_value: (split) argument count must be positive, got 0
// (1000.00d).split_residual(7, 2, 7, "half_up")      // Error: index_out_of_bounds: (split_residual) 7 out of range [0, 6]
// (100d).allocate([1, -1], 2)                        // Error: invalid_value: (allocate) ratio at index 1 is negative
// (100d).allocate([0, 0], 2)                         // Error: invalid_value: (allocate) ratios must not all be zero
// (100d).allocate([], 2)                             // Error: invalid_value: (allocate) ratios must not be empty
// ```
//
// ## No sequence members
//
// Scalars have no `len()`, no elements, and no `repeat`:
//
// ```go
// (1.5d).repeat(2)    // Error: invalid_method: type decimal has no method repeat
// ```
//
// ## Conversions
//
// `x.T()` answers a valid `T` or raises (kind `conversion`); `x.T(default)` answers the default instead;
// `undefined.T(d)` → `d` covers maybe-missing data.
//
// | target | behavior |
// | --- | --- |
// | `decimal` | identity |
// | `int` | truncation toward zero for in-range values (documented resolution loss); out-of-range and NaN raise-or-default |
// | `float` | nearest float64 — approximate by nature |
// | `bool` | zero test; NaN raises-or-defaults |
// | `string` / `runes` | rendering at the value's own scale, no `d` suffix (total — the default is never used); `canonical()` first for the trimmed reading |
// | `time` | unix timestamp as sec.frac, **exact to the nanosecond** |
//
// There is no `byte()` or `rune()` conversion — `int` is the sole gateway to the ordinal scalars
// (`x.int().byte()`).
//
//kavun:type decimal

var TypeDecimal = ValueTypeDescr{
	Name:         ConstHook(decimalTypeName),                                                          // PURE by contract
	String:       decimalTypeString,                                                                   // PURE by contract
	Format:       decimalTypeFormat,                                                                   // PURE by contract
	Interface:    func(v Value) any { return *(*dec128.Dec128)(v.Ptr) },                               // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return (*dec128.Dec128)(v.Ptr).MarshalJSON() },      // PURE by contract
	EncodeBinary: decimalTypeEncodeBinary,                                                             // PURE by contract
	DecodeBinary: decimalTypeDecodeBinary,                                                             // IMPURE by contract (mutates target)
	IsTrue:       decimalTypeIsTrue,                                                                   // PURE by contract
	Equal:        decimalTypeEqual,                                                                    // PURE by contract
	UnaryOp:      decimalTypeUnaryOp,                                                                  // PURE by contract
	BinaryOp:     decimalTypeBinaryOp,                                                                 // PURE by contract
	Len:          ConstHook(int64(1)),                                                                 // PURE by contract
	AsString:     func(v Value) (string, bool) { return (*dec128.Dec128)(v.Ptr).StringFixed(), true }, // PURE by contract
	AsInt:        decimalTypeAsInt,                                                                    // PURE by contract
	AsFloat:      decimalTypeAsFloat,                                                                  // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *(*dec128.Dec128)(v.Ptr), true },       // PURE by contract
	AsTime:       decimalTypeAsTime,                                                                   // PURE by contract
	AsBool:       decimalTypeAsBool,                                                                   // PURE by contract

	Methods: decimalMethods,
}

// decimal NaN is an error state, not a domain value; a boolean context refuses
// the question rather than answering it (as arithmetic and conversion already do)
func decimalTypeIsTrue(v Value) (bool, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return false, errs.NewInvalidValueError("decimal NaN is neither true nor false in a boolean context")
	}
	return !o.IsZero(), nil
}

func decimalTypeEncodeBinary(v Value) ([]byte, error) {
	return (*dec128.Dec128)(v.Ptr).MarshalBinary()
}

func decimalTypeDecodeBinary(v *Value, data []byte) error {
	var d dec128.Dec128
	if err := d.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("failed to decode decimal: %w", err)
	}
	// UnmarshalBinary accepts a NaN payload and reports no error, so the error alone is not enough of a check.
	// No operation in the language can produce a NaN decimal; letting the codec introduce one would put a value
	// into a script that every other path is built to rule out.
	if d.IsNaN() {
		return fmt.Errorf("failed to decode decimal: encoded value is not a number")
	}
	*v = NewDecimalValue(d)
	return nil
}

// The repr carries the SCALE: 1.50d and 1.5d are two different constants (they hash apart in the static pool),
// so a reading that collapsed them would not round-trip.
func decimalTypeString(v Value) string {
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return `decimal("NaN")`
	}
	return o.StringFixed() + "d"
}

func decimalTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return decimalTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(decimalTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	d := *(*dec128.Dec128)(v.Ptr)

	// NaN bypasses digit shaping.
	if d.IsNaN() {
		body := "NaN"
		switch sp.Verb {
		case 'F', 'E', 'G':
			body = "NAN"
		}
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
	}

	verb := sp.Verb
	prec := -1
	if sp.HasPrec {
		prec = int(sp.Precision)
	} else {
		switch verb {
		case 'f', 'F', '%', 'e', 'E':
			prec = 6
		}
	}

	negative := d.IsNegative()
	abs := d.Abs()
	var raw string // magnitude string, no leading sign

	switch verb {
	case 0, 's':
		// A decimal's scale is part of the value — "1.50" states cents precision, "1.5" does not — so the
		// default rendering keeps it, matching json.encode. '!' asks for the trimmed reading; 's' is the
		// explicit spelling of this same default, kept so existing format strings still say what they mean.
		if sp.Bare {
			raw = abs.String()
		} else {
			raw = abs.StringFixed()
		}

	case 'f', 'F':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = decimalFixedString(abs, prec)

	case '%':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = decimalFixedString(abs.Mul(dec128.FromInt64(100)), prec) + "%"

	case 'e', 'E':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		sig, exp := decimalSigDigits(abs, prec+1)
		raw = decimalSciFromSig(sig, exp, verb == 'E')

	case 'g', 'G':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		// "shortest" means the shorter of the two EXACT readings, fixed and scientific — not the shortest
		// approximation. 1e-19 stays "1e-19" rather than spelling out nineteen zeros, and a value that reads
		// better in full stays in full.
		want := 0 // 0 = every digit the value has
		if sp.HasPrec {
			want = max(int(sp.Precision), 1) // a 'g' precision counts SIGNIFICANT digits, as it does in Go
		}
		sig, exp := decimalSigDigits(abs, want)
		sig = strings.TrimRight(sig, "0")
		if sig == "" {
			sig = "0"
		}
		fixed := decimalFixedFromSig(sig, exp)
		sci := decimalSciFromSig(sig, exp, verb == 'G')
		raw = fixed
		if len(sci) < len(fixed) {
			raw = sci
		}

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	// 'z' coerce-zero: drop sign when the formatted magnitude is numerically zero.
	if sp.CoerceZero && negative && isAllZeroMagnitude(strings.TrimSuffix(raw, "%")) {
		negative = false
	}

	if sp.Grouping != 0 {
		if sp.Grouping != ',' && sp.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		hasPct := strings.HasSuffix(raw, "%")
		if hasPct {
			raw = raw[:len(raw)-1]
		}
		raw = groupFloatIntegral(raw, sp.Grouping)
		if hasPct {
			raw += "%"
		}
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

// decimalSigDigits splits a NON-NEGATIVE decimal into its significant decimal digits and the exponent of the
// first one: the value is sig[0] . sig[1:] x 10^exp. When want > 0 exactly that many digits are returned,
// rounded half-away-from-zero; want <= 0 keeps every digit the value has, with trailing zeros trimmed.
//
// The digits come from the decimal's own coefficient. Nothing here goes through float64, which is the whole
// point: a float64 round trip invents digits past the 17th significant place.
//
// PURE by contract.
func decimalSigDigits(d dec128.Dec128, want int) (string, int) {
	text := d.StringFixed()
	intPart, fracPart := text, ""
	if i := strings.IndexByte(text, '.'); i >= 0 {
		intPart, fracPart = text[:i], text[i+1:]
	}
	digits := intPart + fracPart

	first := -1
	for i := 0; i < len(digits); i++ {
		if digits[i] != '0' {
			first = i
			break
		}
	}
	if first < 0 { // the value is zero: no significant digit to anchor an exponent on
		return strings.Repeat("0", max(want, 1)), 0
	}

	exp := len(intPart) - 1 - first
	sig := digits[first:]

	if want <= 0 {
		sig = strings.TrimRight(sig, "0")
		if sig == "" {
			sig = "0"
		}
		return sig, exp
	}
	if len(sig) <= want {
		return sig + strings.Repeat("0", want-len(sig)), exp
	}

	head := []byte(sig[:want])
	if sig[want] >= '5' {
		i := len(head) - 1
		for ; i >= 0; i-- {
			if head[i] != '9' {
				head[i]++
				break
			}
			head[i] = '0'
		}
		if i < 0 {
			// the carry ran off the front (999 -> 1000): one more place, same digit count
			head = append([]byte{'1'}, head[:want-1]...)
			exp++
		}
	}
	return string(head), exp
}

// decimalSciFromSig renders significant digits + exponent as d.dddde±XX, with the exponent zero-padded to at
// least two digits as strconv.FormatFloat does.
//
// PURE by contract.
func decimalSciFromSig(sig string, exp int, upper bool) string {
	var b strings.Builder
	b.WriteByte(sig[0])
	if len(sig) > 1 {
		b.WriteByte('.')
		b.WriteString(sig[1:])
	}
	if upper {
		b.WriteByte('E')
	} else {
		b.WriteByte('e')
	}
	if exp < 0 {
		b.WriteByte('-')
		exp = -exp
	} else {
		b.WriteByte('+')
	}
	if exp < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(exp))
	return b.String()
}

// decimalFixedFromSig renders significant digits + exponent in plain positional notation.
//
// PURE by contract.
func decimalFixedFromSig(sig string, exp int) string {
	if exp >= 0 {
		if len(sig) <= exp+1 {
			return sig + strings.Repeat("0", exp+1-len(sig))
		}
		return sig[:exp+1] + "." + sig[exp+1:]
	}
	return "0." + strings.Repeat("0", -exp-1) + sig
}

// decimalFixedString renders a non-negative Dec128 in fixed-point notation with exactly prec fractional digits (no
// trailing-zero trim). If prec < 0, the canonical representation is returned (trailing zeros trimmed).
//
// A precision past MaxScale is honoured by padding: a decimal never has more than MaxScale places, so every digit
// beyond is a zero and writing it is exact. Only the rounding step is bounded by the type.
func decimalFixedString(d dec128.Dec128, prec int) string {
	if prec < 0 {
		return d.String()
	}
	rounded := d.RoundHalfAwayFromZero(uint8(min(prec, int(dec128.MaxScale))))
	s := rounded.String()
	dot := strings.IndexByte(s, '.')
	var intp, fracp string
	if dot < 0 {
		intp, fracp = s, ""
	} else {
		intp, fracp = s[:dot], s[dot+1:]
	}
	if len(fracp) < prec {
		fracp += strings.Repeat("0", prec-len(fracp))
	} else if len(fracp) > prec {
		fracp = fracp[:prec]
	}
	if prec == 0 {
		return intp
	}
	return intp + "." + fracp
}

// PURE by contract. A decimal in conversion context is a unix timestamp read as sec.frac: the
// integer part is seconds since epoch, the fraction is the sub-second part. Always UTC, like every
// other int-shaped conversion.
//
// This is the EXACT sec.frac path -- dec128 is base-10, so decimal("1704067200.123456789") converts
// with every digit intact, where the float64 spelling of the same number cannot. Anything finer than
// nanoseconds truncates (that is the resolution of a time value, not of the decimal). NaN and
// out-of-int64-range values decline, surfacing as time(x) -> undefined or the time(x, fallback)
// default.
func decimalTypeAsTime(v Value) (time.Time, bool) {
	d := *(*dec128.Dec128)(v.Ptr)
	if d.IsNaN() {
		return time.Time{}, false
	}
	whole := d.Trunc(0)
	sec, err := whole.Int64()
	if err != nil {
		return time.Time{}, false
	}
	nsec, err := d.Sub(whole).MulInt64(1_000_000_000).Trunc(0).Int64()
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, nsec).UTC(), true
}

func decimalTypeAsInt(v Value) (int64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsBool(v Value) (bool, bool) {
	// the conversion is a zero check; NaN is an error state and declines
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return false, false
	}
	return !o.IsZero(), true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Decimal:
		l := (*dec128.Dec128)(v.Ptr)
		r := (*dec128.Dec128)(other.Ptr)
		return l.Equal(*r)
	case value.Int, value.Rune, value.Byte, value.Bool:
		l := (*dec128.Dec128)(v.Ptr)
		r, _ := other.AsInt() // always succeeds and is exact for Int/Rune/Byte/Bool
		return l.Equal(dec128.FromInt64(r))
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// decimalResult is the SINGLE guard on producing a decimal from a dec128 computation.
//
// dec128 signals every failure the same way: it answers NaN. That is an ERROR STATE, not a value — unlike an IEEE
// NaN it even compares equal to itself, so it behaves as an ordinary number everywhere downstream and a wrong
// answer propagates silently through a whole rules script. Kavun has one failure mode, so a NaN result raises
// here instead of travelling on, matching what float arithmetic already does. A NaN that carries dec128's
// division-by-zero reason raises division_by_zero, the kind `int` answers for `1 / 0`.
//
// PURE by contract.
func decimalResult(name string, d dec128.Dec128) (Value, error) {
	if !d.IsNaN() {
		return NewDecimalValue(d), nil
	}
	if errors.Is(d.ErrorDetails(), state.DivisionByZero.Error()) {
		return Undefined, errs.NewDivisionByZeroError()
	}
	if details := d.ErrorDetails(); details != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, details.Error()))
	}
	return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) decimal result is not a number", name))
}

// decimalArithResult is decimalResult for the binary operators. A zero divisor is reported as division_by_zero,
// the same kind `int` answers for `1 / 0`, rather than as a generic bad value.
//
// PURE by contract.
func decimalArithResult(d dec128.Dec128, op token.Token, rZero bool) (Value, error) {
	if d.IsNaN() && rZero && (op == tokens.Quo || op == tokens.Rem) {
		return Undefined, errs.NewDivisionByZeroError()
	}
	return decimalResult(op.String(), d)
}

func decimalTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		r := (*dec128.Dec128)(v.Ptr)
		switch other.Type {
		case value.Bool, value.Byte, value.Rune:
			l := dec128.FromInt64(int64(other.Data))
			switch op {
			case tokens.Less:
				return BoolValue(l.LessThan(*r)), nil
			case tokens.Greater:
				return BoolValue(l.GreaterThan(*r)), nil
			case tokens.LessEq:
				return BoolValue(l.LessThanOrEqual(*r)), nil
			case tokens.GreaterEq:
				return BoolValue(l.GreaterThanOrEqual(*r)), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	l := (*dec128.Dec128)(v.Ptr)
	switch other.Type {
	case value.Decimal:
		r := *(*dec128.Dec128)(other.Ptr)
		switch op {
		case tokens.Add:
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case tokens.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case tokens.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case tokens.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case tokens.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Int:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case tokens.Add:
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case tokens.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case tokens.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case tokens.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case tokens.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Bool, value.Byte, value.Rune:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func decimalTypeUnaryOp(v Value, op token.Token) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch op {
	case tokens.Sub:
		return decimalResult("-", o.Neg())

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

// decimalScaleArg reads a target-scale argument: a whole number in 0..dec128.MaxScale. It goes through
// parseIntArg, so a lossless float or decimal spelling is accepted (rescale(2.0)) and a fractional one raises
// rather than silently truncating (rescale(2.5)), exactly as every other count-shaped argument behaves.
//
// PURE by contract.
func decimalScaleArg(name, pos string, a Value) (uint8, error) {
	scale, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if scale < 0 || scale > int64(dec128.MaxScale) {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
	}
	return uint8(scale), nil
}

// decimalOperandArg reads the other operand of a two-operand member. It accepts exactly what the arithmetic
// OPERATORS accept — decimal and int — so div_round(y, n, mode) and `/` agree on their domain. float is refused
// here for the same reason `decimal / float` is refused: there is no automatic winner between the two
// representations.
//
// PURE by contract.
func decimalOperandArg(name, pos string, a Value) (dec128.Dec128, error) {
	switch a.Type {
	case value.Decimal:
		return *(*dec128.Dec128)(a.Ptr), nil
	case value.Int:
		return dec128.FromInt64(int64(a.Data)), nil
	}
	return dec128.Dec128{}, errs.NewInvalidArgumentTypeError(name, pos, "decimal or int", a.TypeName())
}

// decimalModeArg reads a rounding-mode argument (roundingModeArg) as the dec128 mode with the same reading.
//
// PURE by contract.
func decimalModeArg(name, pos string, a Value) (dec128.RoundingMode, error) {
	m, err := roundingModeArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	return m.dec128(), nil
}

// decimalPlacesArg reads round()'s place count: 0..MaxScale fractional places, or a negative count that rounds to
// tens, hundreds, … down to the 10^38 place, past which no coefficient reaches.
//
// PURE by contract.
func decimalPlacesArg(name, pos string, a Value) (int, error) {
	n, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if n < -38 || n > int64(dec128.MaxScale) {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) places must be between -38 and %d", name, dec128.MaxScale))
	}
	return int(n), nil
}

// decimalRoundTo is round(n, mode): EXACTLY n places for n >= 0 (1.5 at 2 is 1.50, as Python's round(Decimal, n)
// answers), and a whole number at scale 0 for a negative n (1234.5 at -2 is 1200).
//
// PURE by contract.
func decimalRoundTo(d dec128.Dec128, places int, mode dec128.RoundingMode) dec128.Dec128 {
	if places >= 0 {
		return d.RescaleRound(uint8(places), mode)
	}
	return d.RoundToPlaces(int8(places), mode)
}

// decimalCountArg reads a strictly positive int-shaped argument (a root degree, a share count, a digit count).
//
// PURE by contract.
func decimalCountArg(name, pos string, a Value) (int64, error) {
	n, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) argument %s must be positive, got %d", name, pos, n))
	}
	return n, nil
}

// decimalMaxRootDegree is dec128's own bound on a root degree (and on a reduced pow_rational exponent), repeated
// here so an over-large degree raises with a message that names the limit rather than a generic domain error. It
// is a measured cost ceiling — 40 years of daily rests fit, a century does not.
const decimalMaxRootDegree = 16384

// decimalScaleModeArgs reads the trailing (scale, mode) pair every *_round member ends with, starting at args[i].
//
// PURE by contract.
func decimalScaleModeArgs(name string, args []Value, i int) (uint8, dec128.RoundingMode, error) {
	scale, err := decimalScaleArg(name, "scale", args[i])
	if err != nil {
		return 0, 0, err
	}
	mode, err := decimalModeArg(name, "mode", args[i+1])
	if err != nil {
		return 0, 0, err
	}
	return scale, mode, nil
}

// decimalRatiosArg reads allocate()'s ratios: a non-empty array of decimal or int, none negative and not all zero.
// These are checked here, before dec128 is called, because its Allocate answers a bare `false` with no reason.
//
// PURE by contract.
func decimalRatiosArg(name string, a Value) ([]dec128.Dec128, error) {
	if a.Type != value.Array {
		return nil, errs.NewInvalidArgumentTypeError(name, "ratios", "array", a.TypeName())
	}
	elems := (*Array)(a.Ptr).Elements
	if len(elems) == 0 {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratios must not be empty", name))
	}
	if len(elems) > seq.MaxSequenceLen {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) result would be %d elements, past the %d limit", name, len(elems), seq.MaxSequenceLen))
	}
	ratios := make([]dec128.Dec128, len(elems))
	allZero := true
	for i, e := range elems {
		r, err := decimalOperandArg(name, "ratios", e)
		if err != nil {
			return nil, err
		}
		if r.IsNegative() {
			return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratio at index %d is negative", name, i))
		}
		if !r.IsZero() {
			allZero = false
		}
		ratios[i] = r
	}
	if allZero {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratios must not all be zero", name))
	}
	return ratios, nil
}

// decimalSharesCountArg reads split()'s share count: positive, and within the sequence-allocation ceiling, because
// the count sizes the array that dec128 builds.
//
// PURE by contract.
func decimalSharesCountArg(name string, a Value) (int, error) {
	n, err := decimalCountArg(name, "count", a)
	if err != nil {
		return 0, err
	}
	if n > seq.MaxSequenceLen {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) result would be %d elements, past the %d limit", name, n, seq.MaxSequenceLen))
	}
	return int(n), nil
}

// decimalResidualIndexArg reads the index of the share that absorbs the rounding. Like an array index, a negative
// one counts from the end (-1 is the last share).
//
// PURE by contract.
func decimalResidualIndexArg(name string, a Value, count int) (int, error) {
	idx, err := parseIntArg(name, "index", a)
	if err != nil {
		return 0, err
	}
	i := idx
	if i < 0 {
		i += int64(count)
	}
	if i < 0 || i >= int64(count) {
		return 0, errs.NewIndexOutOfBoundsError(name, int(idx), count-1)
	}
	return int(i), nil
}

// decimalShares finishes allocate/split: dec128's `ok == false` is turned into a raise, naming the one cause the
// argument checks cannot rule out in advance cheaply (the amount's own scale), and otherwise the representation
// limit. The shares come back as a new, mutable array.
//
// PURE by contract.
func decimalShares(name string, d dec128.Dec128, scale uint8, shares []dec128.Dec128, ok bool) (Value, error) {
	if !ok {
		if scale < d.Scale() {
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf(
				"(%s) scale %d is below the amount's own scale %d; round the amount first", name, scale, d.Scale()))
		}
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the shares cannot be represented at scale %d", name, scale))
	}
	out := make([]Value, len(shares))
	for i, s := range shares {
		out[i] = NewDecimalValue(s)
	}
	return NewArrayValue(out, false), nil
}

// The decimal argument rules, exported for the modules that compute in decimal (fin): one vocabulary, one set of
// messages. decimalOperandArg accepts decimal or int (float raises); ScaleModeArgs reads the trailing
// (scale, mode) pair at args[i]; decimalResult turns a NaN result into its raise (division_by_zero or
// invalid_value).

// PURE by contract.
func DecimalOperandArg(name, pos string, a Value) (dec128.Dec128, error) {
	return decimalOperandArg(name, pos, a)
}

// PURE by contract.
func ScaleModeArgs(name string, args []Value, i int) (uint8, dec128.RoundingMode, error) {
	return decimalScaleModeArgs(name, args, i)
}

// PURE by contract.
func DecimalResult(name string, d dec128.Dec128) (Value, error) {
	return decimalResult(name, d)
}

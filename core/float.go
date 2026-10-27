package core

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/num"
)

const floatTypeName = "float"

func FloatValue(f float64) Value {
	return Value{
		Type:      value.Float,
		Immutable: true,
		Data:      math.Float64bits(f),
	}
}

// IEEE 754 double-precision floats with an arithmetic quarantine: no arithmetic result is ever NaN or ±Inf — it
// raises instead.
//
// ## Overview
//
// `float` is a 64-bit IEEE 754 binary float. Kavun quarantines its error states: an arithmetic operation whose
// result would be NaN or ±Inf **raises a catchable error** instead of letting the sentinel flow through a
// calculation silently. The sentinel *values* themselves remain first-class: they can arrive from parses
// (`float("inf")`), from the host, or from JSON, and they can be stored, compared, and sorted — only arithmetic
// refuses them. `is_nan()` / `is_inf()` interrogate them without triggering the raise being avoided.
//
// `float` belongs to the numeric family alongside [`int`](int.md) and [`decimal`](decimal.md). Use `float` for
// measurements and physics-style computation where binary rounding is acceptable; use `decimal` for money and any
// result that must be exact in base 10 (`0.1 + 0.2 == 0.3` is `false` on floats and `true` on decimals).
//
// A numeric operation is a `float` member exactly when [`decimal`](decimal.md) has it — same name, same reading
// (pinned by a parity test). Like the operators, no member produces NaN or ±Inf: such a result **raises**
// `invalid_value` naming the member.
//
// The rest of float math — trigonometry, `cbrt`, `hypot`, the special functions — is in the
// [`math`](../stdlib/math.md)
// module.
//
// ## Literals
//
// ```go
// x = 1.5          // decimal point
// k = 1e3          // exponent, 1000.0
// m = 2.5e-3       // => 0.0025
// n = 3f           // f suffix: a float-typed 3
// u = 1_000.25     // underscores group digits
// h = 0x1.8p1      // hex float with binary exponent, 3.0
// ```
//
// ## Rendering
//
// `string()` and the default `format()` render the shortest decimal string that round-trips — which is why binary
// rounding is visible in full rather than hidden:
//
// ```go
// string(1.5)          // => "1.5"
// string(0.1 + 0.2)    // => "0.30000000000000004" — the true stored value
// string(1e21)         // => "1000000000000000000000"
// string(-0.0)         // => "-0"
// ```
//
// The **display** form — what a container shows, and what a doc example writes — always carries a decimal point,
// so a whole float never reads back as an `int`:
//
// ```go
// format("{0}", [[3.0, 3]])   // => "[3.0, 3]" — the float keeps its point, the int has none
// format("{0}", [[0.0]])      // => "[0.0]"
// ```
//
// The **conversion** stays bare on purpose: `3 == 3.0` is `true`, so the two must produce the same key text and
// the same `join` output. `(3.0).string()` is `"3"`, `f"{3.0}"` is `"3"`, and `json.encode` is unchanged.
//
// ## Arithmetic and operators
//
// | operator | meaning | notes |
// | --- | --- | --- |
// | `+` `-` `*` `/` | IEEE arithmetic | result checked by the quarantine |
// | `%` | floating-point remainder | `5.5 % 2` → `1.5`; checked too |
// | `-x` | negation | arithmetic — checked (see below) |
//
// ### The quarantine: NaN/Inf results raise
//
// Every arithmetic result is checked; a would-be NaN or ±Inf raises a catchable error of kind `invalid_value`:
//
// ```go
// 1.0 / 0.0       // Error: invalid_value: float overflow or division by zero
// -1.0 / 0.0      // Error: invalid_value: float overflow or division by zero
// 0.0 / 0.0       // Error: invalid_value: invalid float arithmetic (NaN result)
// 1e308 * 10.0    // Error: invalid_value: float overflow or division by zero — overflow to Inf
// 1.0 % 0.0       // Error: invalid_value: invalid float arithmetic (NaN result)
// ```
//
// The rule applies to sentinel *operands* too — arithmetic refuses them even though the values are storable:
//
// ```go
// inf = float("inf")
// nan = float("nan")
//
// inf - inf       // Error: invalid_value: invalid float arithmetic (NaN result)
// inf + 1.0       // Error: invalid_value: float overflow or division by zero
// nan + 1.0       // Error: invalid_value: invalid float arithmetic (NaN result)
// -inf            // Error: invalid_value — unary minus is arithmetic; negate no sentinel
// ```
//
// ### Pairing with int
//
// `float` pairs with `int` in arithmetic; the result is `float` under the same quarantine (`1 / 0.0` raises).
// `float` **never pairs with `decimal` in arithmetic** — only in comparisons. Convert explicitly
// (see [decimal](decimal.md)):
//
// ```go
// 1.5 + 1       // => 2.5
// 1.5 + 1d      // Error: invalid_binary_operator: float + decimal — convert explicitly: 1.5.decimal() + 1d
// ```
//
// ## NaN and Inf values
//
// The sentinels arrive from parses and hosts, never from Kavun arithmetic:
//
// ```go
// inf = float("inf")       // +Inf ("Infinity" and case variants parse too)
// ninf = float("-inf")     // => -Inf
// nan = float("nan")       // => NaN
//
// inf.is_inf()    // => true
// nan.is_nan()    // => true
// (1.5).is_nan()  // => false
// ```
//
// `is_inf()` takes no sign argument — negative infinity is a composition:
//
// ```go
// ninf := float("-inf")
// ninf.is_inf() && ninf.sign() < 0    // => true — "is it -Inf"
// ```
//
// ### Total-order comparisons
//
// Comparison treats the whole float domain as one **total order**: NaN is the unique minimum (below even `-Inf`)
// and equal only to another NaN. This keeps sorting deterministic instead of IEEE-unordered:
//
// ```go
// inf := float("inf")
// nan := float("nan")
// nan == nan                     // => true — reflexive, unlike raw IEEE
// nan < float("-inf")            // => true — NaN is the unique minimum
// inf > 1e308                    // => true
// [2.0, nan, 1.0, inf].sort()    // => [NaN, 1.0, 2.0, +Inf]
// nan == 1.0                     // => false
// nan < 1                        // => false — NaN orders below numbers, not against them piecewise
// ```
//
// ### Truthiness
//
// Truthiness is inequality with `0.0`. NaN is an error state, not a domain value — a boolean context **raises**
// rather than answering:
//
// ```go
// (0.0).is_true()     // => false
// (-0.0).is_true()    // => false
// (0.5).is_true()     // => true
// !!0.0               // => false
// is_true(float("nan"))          // Error: invalid_value: float NaN is neither true nor false in a boolean context
// float("nan").is_true()         // Error: invalid_value — the same raise: member and free spelling agree
// if float("nan") {}             // Error: invalid_value — and the same raise in a condition
// ```
//
// `float("inf").is_true()` is `true` — Inf is nonzero and has a definite answer.
//
// ## Comparisons and cross-type pairing
//
// Cross-type `==` / `<` against `int` and `decimal` are **exact mathematical comparisons** — the float's true
// stored value is compared, with no rounding of either side:
//
// ```go
// 1 == 1.0                              // => true
// 0.1d == 0.1                           // => false — float 0.1 is really 0.1000000000000000055511…
// 0.1d < 0.1                            // => true — the decimal is below the float's true value
// 9007199254740993 == 9007199254740993.0   // => false — that float literal rounds to …992
// ```
//
// `bool` / `byte` / `rune` widen to their integer value. Equality against a `string` compares the canonical text
// form (`1.5 == "1.5"` → `true`, `float("nan") == "NaN"` → `true`); ordering against text raises.
//
// ## No sequence members
//
// Scalars have no `len()`, no elements, and no `repeat`:
//
// ```go
// (1.5).repeat(2)    // Error: invalid_method: type float has no method repeat
// ```
//
// ## Conversions
//
// `x.T()` answers a valid `T` or raises (kind `conversion`); `x.T(default)` answers the default instead. The free
// form `T(x)` is the same conversion — the default slot is the member's only; `undefined.T(d)` → `d` covers
// maybe-missing data.
//
// | target | behavior |
// | --- | --- |
// | `float` | identity |
// | `int` | truncation toward zero for in-range values (documented resolution loss); NaN, ±Inf, and out-of-range raise-or-default |
// | `decimal` | exact-enough shortest decimal (`(0.1).decimal()` → `0.1d`); NaN/±Inf raise-or-default |
// | `bool` | zero test; NaN raises-or-defaults |
// | `string` / `runes` | the shortest round-trip rendering (total — the default is never used) |
// | `time` | unix timestamp as sec.frac — **lossy**, see [`time`](#time) |
//
// Truncation is a documented in-range resolution loss of *conversion*. **Count slots are stricter**: an argument
// position that means a count, width, or edit position accepts a float only losslessly —
//
// ```go
// "ab".repeat(2.0)    // => "abab" — 2.0 converts losslessly
// "ab".repeat(1.5)    // Error: invalid_value: (repeat) argument first must be a whole number, got 1.5
// ```
//
// ## Durations
//
// A float builds an int **duration in nanoseconds**, rounded to the nearest one — the same members as
// [int](int.md#durations): `1.5.hours()` is `5400000000000`, `0.25.seconds()` is `250000000`. NaN, ±Inf and a
// result past int's range raise.
//
// ## Parsing text into float
//
// ```go
// float("1.5")           // => 1.5
// float("inf")           // => +Inf — a parse is not arithmetic; the value is legal
// float("nan")           // => NaN
// float("abc")           // Error: conversion: cannot convert string to float
// float("1e309")         // Error: conversion: cannot convert string to float — not representable
// "abc".float(0.0)       // => 0.0 — the member's default rescues bad data
// undefined.float(1.0)   // => 1.0 — the maybe-missing form
// float(3)               // => 3.0
// ```
//
// There is no `byte()` or `rune()` conversion on `float` — `int` is the sole gateway between the numeric family and
// the ordinal scalars (`x.int().byte()`).
//
//kavun:type float

var TypeFloat = ValueTypeDescr{
	Name:         ConstHook(floatTypeName),                                  // PURE by contract
	String:       floatTypeString,                                           // PURE by contract
	Format:       floatTypeFormat,                                           // PURE by contract
	Interface:    func(v Value) any { return math.Float64frombits(v.Data) }, // PURE by contract
	EncodeJSON:   floatTypeEncodeJSON,                                       // PURE by contract
	EncodeBinary: floatTypeEncodeBinary,                                     // PURE by contract
	DecodeBinary: floatTypeDecodeBinary,                                     // IMPURE by contract (mutates target)
	IsTrue:       floatTypeIsTrue,                                           // PURE by contract
	Len:          ConstHook(int64(1)),                                       // PURE by contract
	Equal:        floatTypeEqual,                                            // PURE by contract
	UnaryOp:      floatTypeUnaryOp,                                          // PURE by contract
	BinaryOp:     floatTypeBinaryOp,                                         // PURE by contract
	AsInt:        floatTypeAsInt,                                            // PURE by contract
	AsFloat:      floatTypeAsFloat,                                          // PURE by contract
	AsDecimal:    floatTypeAsDecimal,                                        // PURE by contract
	AsBool:       floatTypeAsBool,                                           // PURE by contract
	AsString:     floatTypeAsString,                                         // PURE by contract
	AsTime:       floatTypeAsTime,                                           // PURE by contract

	Methods: floatMethods,
}

func floatTypeIsTrue(v Value) (bool, error) {
	// truthiness is inequality with the type's zero value, so 0.0 is falsy like
	// 0 and decimal(0); NaN is an error state, not a domain value, and a boolean
	// context refuses the question rather than answering it
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) {
		return false, errs.NewInvalidValueError("float NaN is neither true nor false in a boolean context")
	}
	return f != 0, nil
}

func floatTypeString(v Value) string {
	f := math.Float64frombits(v.Data)
	s := strconv.FormatFloat(f, 'f', -1, 64)
	// A whole float keeps its point in the DISPLAY form, so `[3.0]` never reads back as an int.
	// The text CONVERSION (AsString, .string(), a dict key, join) stays bare on purpose: `3 == 3.0`
	// is true, so the two must render the same key text.
	if !math.IsInf(f, 0) && !math.IsNaN(f) && !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func floatTypeEncodeJSON(v Value) ([]byte, error) {
	var y []byte

	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) {
		return nil, errs.NewJSONEncodingError("float Inf has no JSON representation")
	}
	if math.IsNaN(f) {
		return nil, errs.NewJSONEncodingError("float NaN has no JSON representation")
	}

	// Convert as if by ES6 number to string conversion. This matches most other JSON generators.
	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	y = strconv.AppendFloat(y, f, fmt, -1, 64)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(y)
		if n >= 4 && y[n-4] == 'e' && y[n-3] == '-' && y[n-2] == '0' {
			y[n-2] = y[n-1]
			y = y[:n-1]
		}
	}

	return y, nil
}

func floatTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v.Data)
	return b, nil
}

func floatTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("float: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.LittleEndian.Uint64(data)
	return nil
}

func floatTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(floatTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	f := math.Float64frombits(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'g'
	}

	if sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	var (
		fmtVerb byte
		upper   bool
		percent bool
	)
	switch verb {
	case 'f':
		fmtVerb = 'f'
	case 'F':
		fmtVerb = 'f'
		upper = true
	case 'e':
		fmtVerb = 'e'
	case 'E':
		fmtVerb = 'E'
	case 'g':
		fmtVerb = 'g'
	case 'G':
		fmtVerb = 'G'
		upper = true
	case '%':
		fmtVerb = 'f'
		percent = true
	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	prec := -1
	if sp.HasPrec {
		prec = int(sp.Precision)
	} else {
		switch fmtVerb {
		case 'f':
			prec = 6
		case 'e', 'E':
			prec = 6
		case 'g', 'G':
			prec = -1
		}
	}

	if percent {
		f *= 100
	}

	negative := math.Signbit(f) && !math.IsNaN(f)

	// Special values: NaN / ±Inf bypass digit-shaping (no grouping, no zero-pad).
	if math.IsNaN(f) || math.IsInf(f, 0) {
		var body string
		switch {
		case math.IsNaN(f):
			body = "NaN"
			if upper {
				body = "NAN"
			}
		default: // Inf
			body = "Inf"
			if upper {
				body = "INF"
			}
			if negative {
				body = "-" + body
			} else {
				body = fspec.SignPrefix(sp.Sign, false) + body
			}
		}
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
	}

	// Render the magnitude; strconv emits its own leading '-' for negatives, which we strip and re-emit explicitly so
	// that grouping / sign-aware split work uniformly.
	raw := strings.TrimPrefix(strconv.FormatFloat(f, fmtVerb, prec, 64), "-")
	if upper {
		raw = strings.ToUpper(raw)
	}

	// 'z' flag: coerce -0 (and -0.000…) to +0 once rounding has produced an all-zero magnitude.
	if sp.CoerceZero && negative && isAllZeroMagnitude(raw) {
		negative = false
	}

	// Grouping applies to the integral part only.
	if sp.Grouping != 0 {
		if sp.Grouping != ',' && sp.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = groupFloatIntegral(raw, sp.Grouping)
	}

	if percent {
		raw += "%"
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

// isAllZeroMagnitude reports whether a magnitude string (no leading sign) numerically equals zero. It accepts forms
// like "0", "0.000", "0e+00", "0.0E-05".
func isAllZeroMagnitude(s string) bool {
	for _, r := range s {
		switch r {
		case '0', '.':
			continue
		case 'e', 'E':
			return true // remainder is the exponent; mantissa was all zeros
		default:
			return false
		}
	}
	return true
}

// groupFloatIntegral inserts sep into the integral part of a magnitude string (no leading sign). The integral part is
// everything up to the first '.', 'e' or 'E'.
func groupFloatIntegral(s string, sep byte) string {
	end := len(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', 'e', 'E':
			end = i
			i = len(s)
		}
	}
	if end == 0 {
		return s
	}
	return fspec.GroupDigits(s[:end], sep, 3) + s[end:]
}

func floatTypeAsInt(v Value) (int64, bool) {
	// truncation toward zero for in-range values is documented resolution loss;
	// NaN, ±Inf and values outside int64's range have no counterpart and decline
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) || math.IsInf(f, 0) || f >= math.MaxInt64 || f < math.MinInt64 {
		return 0, false
	}
	return int64(f), true
}

func floatTypeAsFloat(v Value) (float64, bool) {
	return math.Float64frombits(v.Data), true
}

func floatTypeAsBool(v Value) (bool, bool) {
	// the conversion is a zero check (the bool -> int edge read backwards);
	// NaN is an error state and declines
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) {
		return false, false
	}
	return f != 0, true
}

// PURE by contract. A float in conversion context is a unix timestamp read as sec.frac: the integer
// part is seconds since epoch, the fraction is the sub-second part (the encoding Python's
// time.time() produces). Always UTC, like every other int-shaped conversion.
//
// LOSSY, unavoidably: float64 has ~15-16 significant digits and a present-day timestamp already
// spends 10 on the seconds, so the fraction survives to roughly microsecond precision and not
// exactly -- 1704067200.123 lands on 1704067200.122999906. Use decimal (exact to nanoseconds) or
// int.time_nano() when the sub-second part has to be right. NaN/Inf and out-of-int64-range values
// decline, which surfaces as time(x) -> undefined or the time(x, fallback) default.
func floatTypeAsTime(v Value) (time.Time, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return time.Time{}, false
	}
	sec := math.Trunc(f)
	if sec >= 9.223372036854776e18 || sec <= -9.223372036854776e18 {
		return time.Time{}, false
	}
	nsec := math.Round((f - sec) * 1e9)
	return time.Unix(int64(sec), int64(nsec)).UTC(), true
}

func floatTypeAsString(v Value) (string, bool) {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), true
}

func floatTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return dec128.NaN(state.NaN), false
	}
	return dec128.FromFloat64(f), true
}

// compareExactAndFloat compares an exact, always-finite rational value against a float64 f,
// implementing the resolved ordering policy for the whole numeric family (docs/types.md):
// any real value is less than +Inf and greater than -Inf; nothing is ordered against NaN. exact
// itself is never NaN/Inf here — it comes from int/bool/byte/rune, or an already-NaN-checked
// decimal (see compareDecimalAndFloat). Returns a normal -1/0/1 Cmp result, or the sentinel -2 for
// "not ordered" (f is NaN).
func compareExactAndFloat(exact *big.Rat, f float64) int {
	switch {
	case math.IsNaN(f):
		return -2
	case math.IsInf(f, 1):
		return -1
	case math.IsInf(f, -1):
		return 1
	default:
		return exact.Cmp(new(big.Rat).SetFloat64(f))
	}
}

// compareDecimalAndFloat is compareExactAndFloat's decimal-specific counterpart: unlike int/bool/
// byte/rune, decimal itself can be NaN, which carries its own total-order placement — the unique
// minimum, equal only to another NaN (mirroring dec128's own Compare/Equal convention, and float's
// eventual same-type NaN change — see docs/types.md's "Resolved: NaN / Inf policy") — rather
// than "ordering against it is always false."
func compareDecimalAndFloat(d *dec128.Dec128, f float64) int {
	if d.IsNaN() {
		if math.IsNaN(f) {
			return 0 // both "NaN" in their respective domains: equal in the total order
		}
		return -1 // decimal NaN is the unique minimum, less than any real float (finite or ±Inf)
	}
	return compareExactAndFloat(num.DecimalToExactRat(d), f)
}

// compareFloatTotalOrder implements the resolved NaN total-order convention for float's own
// same-type comparisons (docs/types.md, "Resolved: NaN / Inf policy"): NaN is the unique
// minimum, sorting below even -Inf, and equal only to another NaN — matching dec128's own
// Compare/Equal convention, rather than IEEE-754 unordered semantics (which made array.sort() on a
// float array containing NaN silently non-deterministic — see core/array.go's sort comparator).
// Inf needs no special-casing: Go's native operators already order it correctly.
func compareFloatTotalOrder(a, b float64) int {
	aNaN, bNaN := math.IsNaN(a), math.IsNaN(b)
	switch {
	case aNaN && bNaN:
		return 0
	case aNaN:
		return -1
	case bNaN:
		return 1
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// exactOrderFloat implements < > <= >= for "exact OP f", given cmp from compareExactAndFloat/
// compareDecimalAndFloat (the -2 NaN sentinel always resolves to false, for every op).
func exactOrderFloat(cmp int, op token.Token) (Value, error) {
	if cmp == -2 {
		return BoolValue(false), nil
	}
	switch op {
	case tokens.Less:
		return BoolValue(cmp < 0), nil
	case tokens.Greater:
		return BoolValue(cmp > 0), nil
	case tokens.LessEq:
		return BoolValue(cmp <= 0), nil
	default: // tokens.GreaterEq
		return BoolValue(cmp >= 0), nil
	}
}

// floatOrderExact is exactOrderFloat's mirror, for "f OP exact" — float as the left operand instead
// of the right (float's own non-reflected branch, where v is float and other is the exact side).
func floatOrderExact(cmp int, op token.Token) (Value, error) {
	if cmp == -2 {
		return BoolValue(false), nil
	}
	switch op {
	case tokens.Less:
		return BoolValue(cmp > 0), nil
	case tokens.Greater:
		return BoolValue(cmp < 0), nil
	case tokens.LessEq:
		return BoolValue(cmp >= 0), nil
	default: // tokens.GreaterEq
		return BoolValue(cmp <= 0), nil
	}
}

func floatTypeEqual(v Value, other Value, final bool) bool {
	f := math.Float64frombits(v.Data)
	switch other.Type {
	case value.Float:
		return compareFloatTotalOrder(f, math.Float64frombits(other.Data)) == 0
	case value.Bool, value.Byte, value.Rune:
		i := int64(other.Data)
		return f == float64(i)
	case value.Int:
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false // int is never NaN/Inf
		}
		i := int64(other.Data)
		fr := new(big.Rat).SetFloat64(f)
		return fr.Cmp(new(big.Rat).SetInt64(i)) == 0
	case value.Decimal:
		d := (*dec128.Dec128)(other.Ptr)
		if d.IsNaN() {
			return math.IsNaN(f)
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false
		}
		fr := new(big.Rat).SetFloat64(f)
		return fr.Cmp(num.DecimalToExactRat(d)) == 0
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func floatTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Bool, value.Byte, value.Rune:
			switch op {
			case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
				l := int64(other.Data)
				r := math.Float64frombits(v.Data)
				cmp := compareExactAndFloat(new(big.Rat).SetInt64(l), r)
				return exactOrderFloat(cmp, op)
			}
		case value.Decimal:
			switch op {
			case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
				l := (*dec128.Dec128)(other.Ptr)
				r := math.Float64frombits(v.Data)
				cmp := compareDecimalAndFloat(l, r)
				return exactOrderFloat(cmp, op)
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Float:
		l := math.Float64frombits(v.Data)
		r := math.Float64frombits(other.Data)
		switch op {
		case tokens.Add:
			return floatArithResult(l + r)
		case tokens.Sub:
			return floatArithResult(l - r)
		case tokens.Mul:
			return floatArithResult(l * r)
		case tokens.Quo:
			return floatArithResult(l / r)
		case tokens.Rem:
			return floatArithResult(math.Mod(l, r))
		case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
			cmp := compareFloatTotalOrder(l, r)
			switch op {
			case tokens.Less:
				return BoolValue(cmp < 0), nil
			case tokens.Greater:
				return BoolValue(cmp > 0), nil
			case tokens.LessEq:
				return BoolValue(cmp <= 0), nil
			default:
				return BoolValue(cmp >= 0), nil
			}
		}

	case value.Int:
		l := math.Float64frombits(v.Data)
		i := int64(other.Data)
		r := float64(i)
		switch op {
		case tokens.Add:
			return floatArithResult(l + r)
		case tokens.Sub:
			return floatArithResult(l - r)
		case tokens.Mul:
			return floatArithResult(l * r)
		case tokens.Quo:
			return floatArithResult(l / r)
		case tokens.Rem:
			return floatArithResult(math.Mod(l, r))
		case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(i), l)
			return floatOrderExact(cmp, op)
		}

	case value.Bool, value.Byte, value.Rune:
		switch op {
		case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
			l := math.Float64frombits(v.Data)
			r := int64(other.Data)
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(r), l)
			return floatOrderExact(cmp, op)
		}

	case value.Decimal:
		switch op {
		case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
			l := math.Float64frombits(v.Data)
			r := (*dec128.Dec128)(other.Ptr)
			cmp := compareDecimalAndFloat(r, l)
			return floatOrderExact(cmp, op)
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract

// floatArithResult guards every float arithmetic result: no arithmetic may
// produce NaN or ±Inf — division by zero and overflow raise, matching int's
// behaviour. The sentinels stay representable (parses, the host) and keep
// their total-order comparison semantics; only arithmetic refuses them.
func floatArithResult(f float64) (Value, error) {
	if math.IsNaN(f) {
		return Undefined, errs.NewInvalidValueError("invalid float arithmetic (NaN result)")
	}
	if math.IsInf(f, 0) {
		return Undefined, errs.NewInvalidValueError("float overflow or division by zero")
	}
	return FloatValue(f), nil
}

func floatTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case tokens.Sub: // see also fast track in VM UnaryNeg
		f := math.Float64frombits(v.Data)
		return floatArithResult(-f)
	}

	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

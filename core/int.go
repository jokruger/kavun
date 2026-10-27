package core

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/text"
)

const intTypeName = "int"

func IntValue(i int64) Value {
	return Value{
		Type:      value.Int,
		Immutable: true,
		Data:      uint64(i),
	}
}

// Checked 64-bit signed integers: arithmetic that overflows raises, it never wraps.
//
// ## Overview
//
// `int` is a 64-bit signed integer (-9,223,372,036,854,775,808 to 9,223,372,036,854,775,807; the bounds are
// available as `math.max_int` / `math.min_int`). It is a **checked** numeric: any arithmetic whose mathematical
// result does not fit in 64 bits raises a catchable error instead of silently wrapping. Wide modular arithmetic is
// not part of the language — [`byte`](byte.md) is the only type with wrap-around arithmetic.
//
// `int` belongs to the numeric family alongside [`float`](float.md) and [`decimal`](decimal.md): it pairs with both
// in arithmetic and compares exactly against both. Use `int` for counts, indexes, identifiers, and timestamps; use
// `decimal` for money and `float` for measurements.
//
// Like every numeric type, `int` answers the sentinel predicates `is_nan()` / `is_inf()` — constantly `false`,
// since an `int` can never hold such a state. That uniformity lets generic numeric code interrogate any number
// without switching on its concrete type.
//
// `int` has no `is_zero()` / `is_negative()` / `is_positive()` members — write the comparison (`x == 0`, `x < 0`,
// `x > 0`).
//
// ## Literals
//
// ```go
// i = 42            // decimal
// h = 0x2a          // hexadecimal, 42
// o = 0o755         // octal, 493
// b = 0b1010        // binary, 10
// big = 1_000_000   // underscores group digits in any base
// ```
//
// A literal with a decimal point, an exponent, or an `f` suffix is a [`float`](float.md); a `d` suffix makes a
// [`decimal`](decimal.md). A `.` followed by a letter is NOT a decimal point but a member call on the int —
// `30.seconds()`, `7.string()` need no parentheses (so `1.e5` is not a float: write `1e5` or `1.0e5`).
//
// ## Arithmetic and operators
//
// | operator | meaning | notes |
// | --- | --- | --- |
// | `+` `-` `*` | add, subtract, multiply | raise on overflow |
// | `/` | division, truncated toward zero | `7 / 2` → `3`, `-7 / 2` → `-3`; raises on `/ 0` |
// | `%` | remainder (sign follows the dividend) | `-7 % 2` → `-1`; raises on `% 0` |
// | `&` `\|` `^` `&^` | bitwise and, or, xor, and-not | never overflow |
// | `<<` `>>` | shift left / right | see the shift rules below |
// | `-x` | negation | raises on `-math.min_int` |
// | `^x` | bitwise complement | `^5` → `-6` |
//
// ### Overflow raises, always
//
// Every arithmetic overflow raises a catchable error of kind `invalid_value` with the message `int overflow`:
//
// ```go
// math = import("math")
//
// math.max_int + 1            // Error: invalid_value: int overflow
// math.min_int - 1            // Error: invalid_value: int overflow
// math.min_int * -1           // Error: invalid_value: int overflow
// -math.min_int               // Error: invalid_value: int overflow — negation is arithmetic too
// math.min_int / -1           // Error: invalid_value: int overflow — the one division that overflows
// 3037000500 * 3037000500     // Error: invalid_value: int overflow
// ```
//
// The error is an ordinary raise — `recover()` catches it:
//
// ```go
//
//	io = import("io")
//
//	checked_add = func(a, b) res {
//	    defer func() {
//	        e = recover()
//	        if e != undefined { res = undefined; io.print("overflow: ", string(e)) }
//	    }()
//	    res = a + b
//	}
//
// ```
//
// ### Division by zero
//
// `/ 0` and `% 0` raise a catchable error of kind `division_by_zero`:
//
// ```go
// 1 / 0    // Error: division_by_zero
// 1 % 0    // Error: division_by_zero
// ```
//
// ### Shift rules
//
// `<<` raises whenever the result would lose information — a value bit or the sign bit shifted out, or an invalid
// count. `>>` is an arithmetic shift: for counts of 64 and above it saturates at the sign bit (no information is
// invented), and it raises only on a negative count.
//
// ```go
// 1 << 62      // => 4611686018427387904
// -1 << 1      // => -2 — sign preserved, nothing lost
// 1 << 63      // Error: invalid_value: int overflow — the bit lands on the sign
// 1 << 64      // Error: invalid_value: int overflow — count out of range
// 5 << -1      // Error: invalid_value: int overflow — negative count
//
// -8 >> 1      // => -4
// 5 >> 100     // => 0 — saturates at the sign bit
// -5 >> 100    // => -1 — saturates at the sign bit
// 5 >> -1      // Error: invalid_value: int overflow — negative count
// ```
//
// ### Pairing with float and decimal
//
// `int` mixes freely with the other two numerics; the result takes the wider representation:
//
// ```go
// 1 + 1.5             // => 2.5 — float (float arithmetic rules apply: 1 / 0.0 raises)
// 10 / 4.0            // => 2.5 — float
// 3 + 1.5d            // => 4.5d — decimal
// 10 / 4d             // => 2.5d — decimal
// ```
//
// `float` and `decimal` never mix with *each other* in arithmetic — `int` is the only operand type both accept.
//
// ## Comparisons and cross-type pairing
//
// `==` and the orderings are **exact mathematical comparisons** across the whole numeric family — no operand is
// silently rounded to make the comparison cheap:
//
// ```go
// 1 == 1.0                                   // => true
// 1 == 1d                                    // => true
// 2 < 2.5                                    // => true
// 2 < 2.5d                                   // => true
// 9007199254740993 == 9007199254740992.0     // => false — exact, no rounding to float
// 9007199254740993 == 9007199254740993.0     // => false — that float value IS ...992
// 9007199254740993 < 9007199254740994.0      // => true
// ```
//
// `bool`, `byte`, and `rune` widen to their integer value for comparison (`true == 1`, `65 == 'A'`, `5 < 'A'` are
// all `true`). Equality against a `string` compares the int's canonical text form; ordering against text does not
// exist:
//
// ```go
// 5 == "5"     // => true — canonical form
// 5 == "05"    // => false — "05" is not the canonical rendering of 5
// 5 < "6"      // Error: invalid_binary_operator: int < string — numeric-vs-text ordering is undefined
// ```
//
// ## No sequence members
//
// `int` is a scalar: it has no `len()`, no elements, and no sequence members. In particular `repeat` does not exist
// on any scalar — promotion into a sequence is the count constructors' job (`T(x, count)` builds `count` copies
// of `x`):
//
// ```go
// (5).repeat(2)     // Error: invalid_method: type int has no method repeat
// array(5, 3)       // => [5, 5, 5] — the promotion spelling: three copies of 5
// bytes(b'a', 3)    // => bytes([97, 97, 97])
// ```
//
// ## Conversions
//
// Every conversion follows one contract: `x.T()` answers a valid `T` or raises a catchable error (kind
// `conversion`); `x.T(default)` answers the default instead of raising. The free form `T(x)` is the same
// conversion — the default slot is the member's only; `undefined.T(d)` → `d` is the standard maybe-missing
// spelling.
//
// | target | behavior |
// | --- | --- |
// | `int` | identity — the same value |
// | `float` | the nearest float64; exact up to 2⁵³, silently approximate above (in-range resolution loss) |
// | `decimal` | exact |
// | `bool` | `false` for 0, `true` otherwise |
// | `string` / `runes` | the decimal rendering (total — the default is never used) |
// | `byte` | value in 0–255, else raise-or-default |
// | `rune` | a valid Unicode code point (0–0x10FFFF, surrogates excluded), else raise-or-default |
// | `time` / `time_ms` / `time_micro` / `time_nano` | a unix timestamp in the named encoding (total — the default is never used) |
//
// ## Durations
//
// An int counts **nanoseconds** when its member says so. The build members turn a count of units into a duration,
// the `in_*` members read a duration in a unit, and `duration_string()` renders it:
//
// ```go
// t := t"2026-08-29T15:04:05.123456789Z"
// 30.seconds()                  // => 30000000000
// 2.weeks() == 14.days()        // => true
// t + 90.minutes()              // an instant 90 minutes after t
// d := 90.minutes()
// d.in_hours()                  // => 1.5 — a float
// d.in_seconds()                // => 5400.0
// d.duration_string()           // => "1h30m0s"
// 9223372037.seconds()          // Error: invalid_value: (seconds) the duration overflows int — int is checked
// ```
//
// | build (`int`, `float`) | read (`int` → `float`) |
// | --- | --- |
// | `nanoseconds()` `microseconds()` `milliseconds()` `seconds()` `minutes()` `hours()` `days()` `weeks()` | `in_microseconds()` `in_milliseconds()` `in_seconds()` `in_minutes()` `in_hours()` `in_days()` `in_weeks()` |
//
// A day is 24 hours and a week 7 days — fixed durations, not calendar units (to move by calendar days or months, use
// the [time](time.md) arithmetic members). The text form of the same reading is
// [`"1h30m".nanoseconds()`](string.md#nanoseconds); a float builds one too, rounded to the nearest nanosecond
// (`1.5.hours()`). Full words name durations; the abbreviations (`time_ms`) stay on the timestamp family.
//
// ## Parsing text into int
//
// The free `int(x)` (equivalently `"...".int()`) parses the canonical decimal form only — no hex
// prefixes, no underscores, no fractions; the member's optional default rescues bad data:
//
// ```go
// int("123")            // => 123
// int("12.5")           // Error: conversion: cannot convert string to int — parse, not truncate
// int("0x2a")           // Error: conversion: cannot convert string to int
// "abc".int(0)          // => 0 — the member's default rescues bad data
// undefined.int(5)      // => 5 — the maybe-missing form
// int(undefined)        // Error: conversion: cannot convert undefined to int: value is missing
// ```
//
// The default rescues *data*, never a program error — a wrong argument type raises even with a default present.
//
//kavun:type int

var TypeInt = ValueTypeDescr{
	Name:         ConstHook(intTypeName),                                                               // PURE by contract
	String:       func(v Value) string { return strconv.FormatInt(int64(v.Data), 10) },                 // PURE by contract
	Format:       intTypeFormat,                                                                        // PURE by contract
	Interface:    func(v Value) any { return int64(v.Data) },                                           // PURE by contract
	EncodeJSON:   intTypeEncodeJSON,                                                                    // PURE by contract
	EncodeBinary: intTypeEncodeBinary,                                                                  // PURE by contract
	DecodeBinary: intTypeDecodeBinary,                                                                  // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                              // PURE by contract
	Len:          ConstHook(int64(1)),                                                                  // PURE by contract
	Equal:        intTypeEqual,                                                                         // PURE by contract
	UnaryOp:      intTypeUnaryOp,                                                                       // PURE by contract
	BinaryOp:     intTypeBinaryOp,                                                                      // PURE by contract
	AsString:     func(v Value) (string, bool) { return strconv.FormatInt(int64(v.Data), 10), true },   // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                           // PURE by contract
	AsFloat:      func(v Value) (float64, bool) { return float64(int64(v.Data)), true },                // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return dec128.FromInt64(int64(v.Data)), true }, // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                              // PURE by contract
	AsRune:       intTypeAsRune,                                                                        // PURE by contract
	AsTime:       func(v Value) (time.Time, bool) { return time.Unix(int64(v.Data), 0).UTC(), true },   // PURE by contract
	AsDate:       intTypeAsDate,                                                                        // PURE by contract
	AsByte:       intTypeAsByte,                                                                        // PURE by contract

	Methods: intMethods,
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(int64(v.Data), 10)
	return []byte(s), nil
}

func intTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v.Data)
	return b, nil
}

func intTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("int: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.LittleEndian.Uint64(data)
	return nil
}

func intTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return strconv.FormatInt(int64(v.Data), 10), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(intTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.HasPrec || sp.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	i := int64(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'd'
	}

	// 'c' renders the code point as its text: a scalar value as its UTF-8 character, an escape as its octet; any
	// other int (a surrogate, out of range) has no character and raises.
	if verb == 'c' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		if !text.IntInRuneDomain(i) {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(text.EncodeRuneText(rune(i)), sp, fspec.AlignLeft), nil
	}

	// 'q' renders the code point as a quoted character literal: 'A', '\n', etc.
	if verb == 'q' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		if i < 0 || i > utf8.MaxRune {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(rune(i)), sp, fspec.AlignLeft), nil
	}

	var (
		base       int
		prefix     string
		groupEvery int
		upper      bool
	)
	switch verb {
	case 'd':
		base = 10
		groupEvery = 3
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
	case 'b':
		base = 2
		prefix = "0b"
		groupEvery = 4
	case 'o':
		base = 8
		prefix = "0o"
		groupEvery = 4
	case 'x':
		base = 16
		prefix = "0x"
		groupEvery = 4
	case 'X':
		base = 16
		prefix = "0x"
		groupEvery = 4
		upper = true
	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.Bare {
		prefix = ""
	}

	if sp.Grouping == ',' && base != 10 {
		return "", errs.NewUnsupportedFormatSpecMsg("',' grouping is only supported with decimal verb 'd'; use '_' for base-2/8/16")
	}

	negative := i < 0
	var u uint64
	if negative {
		// safely negate, including math.MinInt64
		u = uint64(-(i + 1)) + 1
	} else {
		u = uint64(i)
	}

	digits := strconv.FormatUint(u, base)
	if upper {
		digits = strings.ToUpper(digits)
	}
	if sp.Grouping != 0 {
		digits = fspec.GroupDigits(digits, sp.Grouping, groupEvery)
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + prefix + digits
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

func intTypeAsRune(v Value) (rune, bool) {
	// a rune is a Unicode scalar value OR one of the reserved octet escapes (see internal/text/escape.go);
	// negatives, values past MaxRune and the non-escape surrogates are excluded by the same domain
	// rule, not as special cases. The domain is exactly the set that can be encoded back to octets
	i := int64(v.Data)
	if !text.IntInRuneDomain(i) {
		return rune(i), false
	}
	return rune(i), true
}

func intTypeAsByte(v Value) (byte, bool) {
	i := int64(v.Data)
	if i < 0 || i > math.MaxUint8 {
		return byte(i), false
	}
	return byte(i), true
}

func intTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Int, value.Rune, value.Byte, value.Bool:
		return v.Data == other.Data
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func intTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Bool:
			l := int64(other.Data)
			r := int64(v.Data)
			switch op {
			case tokens.Less:
				return BoolValue(l < r), nil
			case tokens.Greater:
				return BoolValue(l > r), nil
			case tokens.LessEq:
				return BoolValue(l <= r), nil
			case tokens.GreaterEq:
				return BoolValue(l >= r), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Int:
		// int is a CHECKED numeric: overflow raises, catchable — wide modular
		// arithmetic left the language (byte is the only modular type)
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Add:
			if s, ok := IntAddChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Sub:
			if s, ok := IntSubChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Mul:
			if s, ok := IntMulChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Quo:
			if r == 0 {
				return Undefined, errs.NewDivisionByZeroError()
			}
			if l == math.MinInt64 && r == -1 {
				return Undefined, errs.NewInvalidValueError("int overflow")
			}
			return IntValue(l / r), nil
		case tokens.Rem:
			if r == 0 {
				return Undefined, errs.NewDivisionByZeroError()
			}
			return IntValue(l % r), nil
		case tokens.And:
			return IntValue(l & r), nil
		case tokens.Or:
			return IntValue(l | r), nil
		case tokens.Xor:
			return IntValue(l ^ r), nil
		case tokens.AndNot:
			return IntValue(l &^ r), nil
		case tokens.Shl:
			if s, ok := IntShlChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Shr:
			if r < 0 {
				return Undefined, errs.NewInvalidValueError("int overflow")
			}
			if r >= 64 {
				// arithmetic shift saturates at the sign bit — no information invented
				return IntValue(l >> 63), nil
			}
			return IntValue(l >> uint64(r)), nil
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}

	case value.Float:
		l := float64(int64(v.Data))
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
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(int64(v.Data)), r)
			return exactOrderFloat(cmp, op)
		}

	case value.Decimal:
		l := dec128.FromInt64(int64(v.Data))
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

	case value.Bool:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
// IntAddChecked / IntSubChecked / IntMulChecked / IntShlChecked are int's checked arithmetic — ok=false means
// the mathematical result does not fit int64 and the operation must raise instead of wrapping. Shared by the
// operator implementations here and the VM's integer fast paths.
func IntAddChecked(l, r int64) (int64, bool) {
	s := l + r
	// overflow iff the operands share a sign and the sum does not
	return s, (l >= 0) != (r >= 0) || (s >= 0) == (l >= 0)
}

func IntSubChecked(l, r int64) (int64, bool) {
	s := l - r
	return s, (l >= 0) == (r >= 0) || (s >= 0) == (l >= 0)
}

func IntMulChecked(l, r int64) (int64, bool) {
	if l == 0 || r == 0 {
		return 0, true
	}
	if l == math.MinInt64 || r == math.MinInt64 {
		if l == 1 {
			return r, true
		}
		if r == 1 {
			return l, true
		}
		return 0, false
	}
	p := l * r
	return p, p/r == l
}

func IntShlChecked(l, r int64) (int64, bool) {
	if r < 0 || r >= 64 {
		return 0, false
	}
	s := l << uint64(r)
	return s, s>>uint64(r) == l
}

func intTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case tokens.Sub: // see also fast track in VM UnaryNeg
		i := int64(v.Data)
		if i == math.MinInt64 {
			return Undefined, errs.NewInvalidValueError("int overflow")
		}
		return IntValue(-i), nil

	case tokens.Xor: // see also fast track in VM UnaryBitNot
		i := int64(v.Data)
		return IntValue(^i), nil
	}

	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// intTypeAsDate reads an int as epoch days (civil.FromDays), failing outside 0001-01-01…9999-12-31.
//
// PURE by contract
func intTypeAsDate(v Value) (civil.Date, bool) {
	n := int64(v.Data)
	if n < -1<<31 || n > 1<<31-1 {
		return civil.Date{}, false
	}
	return civil.FromDays(int32(n))
}

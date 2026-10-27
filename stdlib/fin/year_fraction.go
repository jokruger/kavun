package fin

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/jokruger/fin128/daycount"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.year_fraction is a length of time in YEARS, as an exact N1/D1 + N2/D2: the days a day-count convention
// counts over the year length it takes (fin128's daycount.Fraction). It is owned by the fin module — built only by
// fin.year_fraction(...) and fin.year_fraction_between(...), named after its constructor — and registered by it
// (core.ModuleBuilder.Type).
//
// Every value satisfies one invariant: a valid fraction whose numerators fit int32 and whose denominators are in
// 1…65535 (the second term absent, or both its parts set). That is exactly what fin.year_fraction(n1, d1[, n2, d2])
// accepts, so fin.year_fraction(f.terms()...) == f for every value, and Rational() can never overflow.

const yearFractionTypeName = "fin.year_fraction"

// MaxYearFractionDenominator bounds a term's denominator: fin128's own conventions stay at or below 366, and
// ACTFixed's uint16 caps a year length at 65535.
const MaxYearFractionDenominator = 65535

// NewYearFractionValue wraps a fraction that already satisfies the invariant (see CheckYearFraction).
func NewYearFractionValue(f daycount.Fraction) core.Value {
	return core.Value{Type: value.FinYearFraction, Immutable: true, Ptr: unsafe.Pointer(&f)}
}

// YearFractionOf unwraps a fin.year_fraction value.
func YearFractionOf(v core.Value) daycount.Fraction {
	return *(*daycount.Fraction)(v.Ptr)
}

// CheckYearFraction reports why f is not a fin.year_fraction value, or nil.
func CheckYearFraction(f daycount.Fraction) error {
	if !f.IsValid() {
		return fmt.Errorf("the year fraction is not usable")
	}
	if f.D1 < 1 || f.D1 > MaxYearFractionDenominator || f.D2 < 0 || f.D2 > MaxYearFractionDenominator {
		return fmt.Errorf("denominator must be between 1 and %d", MaxYearFractionDenominator)
	}
	if f.D2 == 0 && f.N2 != 0 {
		return fmt.Errorf("a second numerator needs a second denominator")
	}
	return nil
}

// yearFractionResult checks an arithmetic result against the invariant: an overflow (fin128's invalid zero) or a
// denominator past the bound raises instead of producing a value that could not be rebuilt.
func yearFractionResult(op string, f daycount.Fraction) (core.Value, error) {
	if err := CheckYearFraction(f); err != nil {
		return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(fin.year_fraction %s) result out of range: %s", op, err))
	}
	return NewYearFractionValue(f), nil
}

// A length of time **in years**, as an exact ratio.
//
// ## Overview
//
// A year fraction is the days a day-count convention counts over the year length it takes. It is the `f` the interest
// functions take, kept exact so an accrual never rounds the period before it multiplies. The type is named after its
// constructor (`type_name(f)` is `"fin.year_fraction"`) because it exists only for this module. It is owned by the
// [`fin`](../stdlib/fin.md) module.
//
// ```go
// fin := import("fin")
// f := fin.year_fraction_between(d"2026-01-01", d"2026-02-01", "ACT/365F")   // => fin.year_fraction(31, 365)
// fin.year_fraction_between(d"2026-02-01", d"2026-03-01", "30E/360")        // => fin.year_fraction(30, 360) — every month is 30 days
// fin.year_fraction_between(d"2027-07-01", d"2028-07-01", "ACT/ACT")        // => fin.year_fraction(184, 365, 182, 366) — split at the year end
// fin.year_fraction(31, 365)                 // stated directly: 31 counted days over a 365-day year
// fin.year_fraction()                        // => fin.year_fraction(0, 1) — the default; falsy
// fin.accrue_simple(10000, 0.05d, f, 2, "half_even")   // => 42.47d — interest for January
// ```
//
// `fin.year_fraction(n1, d1[, n2, d2])` takes `int`s: numerators within `int32`, any sign; denominators **1…65535**
// (fin128's own output stays within that, so every value can be rebuilt from its terms). A sum or scale that would
// leave those bounds raises `invalid_value`. JSON is the terms array, `[184, 365, 182, 366]`; `json.decode` gives
// an array back, which `fin.year_fraction(v...)` rebuilds.
//
// ## Operators
//
// | operator | result |
// | --- | --- |
// | `f + g` | the exact sum |
// | `f * n`, `n * f` (`n` an `int`) | scaled — e.g. a year fraction as compounding periods |
// | `f < g` (`<=` `>` `>=`), `f == g` | exact, numeric: `fin.year_fraction(31, 365) == fin.year_fraction(62, 730)` |
// | `f - g`, `f / n`, `f * f`, `f ± decimal`, `f < 1` | raise — use `f.value(…)` to leave the type |
//
//kavun:type fin.year_fraction

// TypeYearFraction is the fin.year_fraction type descriptor.
var TypeYearFraction = core.ValueTypeDescr{
	Name:         core.ConstHook(yearFractionTypeName),                                          // PURE by contract
	String:       yearFractionTypeString,                                                        // PURE by contract
	Format:       yearFractionTypeFormat,                                                        // PURE by contract
	Interface:    func(v core.Value) any { return YearFractionOf(v) },                           // PURE by contract
	EncodeJSON:   yearFractionTypeEncodeJSON,                                                    // PURE by contract
	EncodeBinary: yearFractionTypeEncodeBinary,                                                  // PURE by contract
	DecodeBinary: yearFractionTypeDecodeBinary,                                                  // IMPURE by contract (mutates target)
	IsTrue:       func(v core.Value) (bool, error) { return !YearFractionOf(v).IsZero(), nil },  // PURE by contract: falsy iff equal to the default 0/1
	Len:          core.ConstHook(int64(1)),                                                      // PURE by contract
	Equal:        yearFractionTypeEqual,                                                         // PURE by contract
	BinaryOp:     yearFractionTypeBinaryOp,                                                      // PURE by contract
	AsString:     func(v core.Value) (string, bool) { return YearFractionOf(v).String(), true }, // PURE by contract

	Methods: yearFractionMethods,
}

// yearFractionTermsOf answers the constructor's arguments: [n1, d1] or [n1, d1, n2, d2].
func yearFractionTermsOf(f daycount.Fraction) []int64 {
	if f.D2 == 0 {
		return []int64{int64(f.N1), int64(f.D1)}
	}
	return []int64{int64(f.N1), int64(f.D1), int64(f.N2), int64(f.D2)}
}

// PURE by contract
func yearFractionTypeString(v core.Value) string {
	t := yearFractionTermsOf(YearFractionOf(v))
	s := "fin.year_fraction("
	for i, n := range t {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(n)
	}
	return s + ")"
}

// PURE by contract
func yearFractionTypeFormat(v core.Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return yearFractionTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(yearFractionTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 || sp.HasUnconsumedTail() || sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(YearFractionOf(v).String(), sp, fspec.AlignLeft), nil
}

// PURE by contract: the terms array, which fin.year_fraction(v...) reads back.
func yearFractionTypeEncodeJSON(v core.Value) ([]byte, error) {
	t := yearFractionTermsOf(YearFractionOf(v))
	out := []byte{'['}
	for i, n := range t {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, fmt.Sprint(n)...)
	}
	return append(out, ']'), nil
}

// PURE by contract
func yearFractionTypeEncodeBinary(v core.Value) ([]byte, error) {
	f := YearFractionOf(v)
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:], uint32(f.N1))
	binary.LittleEndian.PutUint32(b[4:], uint32(f.D1))
	binary.LittleEndian.PutUint32(b[8:], uint32(f.N2))
	binary.LittleEndian.PutUint32(b[12:], uint32(f.D2))
	return b, nil
}

// IMPURE by contract (mutates target): a corrupt payload is refused, never decoded into an invalid value.
func yearFractionTypeDecodeBinary(v *core.Value, data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("fin.year_fraction: expected 16 bytes, got %d", len(data))
	}
	f := daycount.Fraction{
		N1: int32(binary.LittleEndian.Uint32(data[0:])),
		D1: int32(binary.LittleEndian.Uint32(data[4:])),
		N2: int32(binary.LittleEndian.Uint32(data[8:])),
		D2: int32(binary.LittleEndian.Uint32(data[12:])),
	}
	if err := CheckYearFraction(f); err != nil {
		return fmt.Errorf("fin.year_fraction: %w", err)
	}
	*v = NewYearFractionValue(f)
	return nil
}

// yearFractionCompare orders two fractions exactly: num1·den2 against num2·den1, in big integers (both reduced
// ratios have positive denominators).
func yearFractionCompare(a, b daycount.Fraction) int {
	an, ad := a.Rational()
	bn, bd := b.Rational()
	l := new(big.Int).Mul(big.NewInt(an), big.NewInt(bd))
	r := new(big.Int).Mul(big.NewInt(bn), big.NewInt(ad))
	return l.Cmp(r)
}

// PURE by contract: numeric equality — 31/365 == 62/730.
func yearFractionTypeEqual(v core.Value, other core.Value, final bool) bool {
	if other.Type == value.FinYearFraction {
		an, ad := YearFractionOf(v).Rational()
		bn, bd := YearFractionOf(other).Rational()
		return an == bn && ad == bd
	}
	if final {
		return false
	}
	return core.ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract
func yearFractionTypeBinaryOp(v core.Value, other core.Value, op token.Token, reflected bool) (core.Value, error) {
	f := YearFractionOf(v)
	if reflected {
		// n * f
		if other.Type == value.Int && op == tokens.Mul {
			return yearFractionResult("*", f.Scaled(int64(other.Data)))
		}
		return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	switch other.Type {
	case value.FinYearFraction:
		g := YearFractionOf(other)
		switch op {
		case tokens.Add:
			return yearFractionResult("+", f.Add(g))
		case tokens.Less:
			return core.BoolValue(yearFractionCompare(f, g) < 0), nil
		case tokens.Greater:
			return core.BoolValue(yearFractionCompare(f, g) > 0), nil
		case tokens.LessEq:
			return core.BoolValue(yearFractionCompare(f, g) <= 0), nil
		case tokens.GreaterEq:
			return core.BoolValue(yearFractionCompare(f, g) >= 0), nil
		}
		return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
	case value.Int:
		if op == tokens.Mul {
			return yearFractionResult("*", f.Scaled(int64(other.Data)))
		}
	}
	return core.ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

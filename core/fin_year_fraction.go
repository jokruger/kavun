package core

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/jokruger/fin128/daycount"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.year_fraction is a length of time in YEARS, as an exact N1/D1 + N2/D2: the days a day-count convention
// counts over the year length it takes (fin128's daycount.Fraction). It is owned by the fin module — built only by
// fin.year_fraction(...) and fin.year_fraction_between(...), named after its constructor — and registered here
// because the type table lives in core.
//
// Every value satisfies one invariant: a valid fraction whose numerators fit int32 and whose denominators are in
// 1…65535 (the second term absent, or both its parts set). That is exactly what fin.year_fraction(n1, d1[, n2, d2])
// accepts, so fin.year_fraction(f.terms()...) == f for every value, and Rational() can never overflow.

const finYearFractionTypeName = "fin.year_fraction"

// MaxFinYearFractionDenominator bounds a term's denominator: fin128's own conventions stay at or below 366, and
// ACTFixed's uint16 caps a year length at 65535.
const MaxFinYearFractionDenominator = 65535

// NewFinYearFractionValue wraps a fraction that already satisfies the invariant (see CheckFinYearFraction).
func NewFinYearFractionValue(f daycount.Fraction) Value {
	return Value{Type: value.FinYearFraction, Immutable: true, Ptr: unsafe.Pointer(&f)}
}

// FinYearFractionOf unwraps a fin.year_fraction value.
func FinYearFractionOf(v Value) daycount.Fraction {
	return *(*daycount.Fraction)(v.Ptr)
}

// CheckFinYearFraction reports why f is not a fin.year_fraction value, or nil.
func CheckFinYearFraction(f daycount.Fraction) error {
	if !f.IsValid() {
		return fmt.Errorf("the year fraction is not usable")
	}
	if f.D1 < 1 || f.D1 > MaxFinYearFractionDenominator || f.D2 < 0 || f.D2 > MaxFinYearFractionDenominator {
		return fmt.Errorf("denominator must be between 1 and %d", MaxFinYearFractionDenominator)
	}
	if f.D2 == 0 && f.N2 != 0 {
		return fmt.Errorf("a second numerator needs a second denominator")
	}
	return nil
}

// finYearFractionResult checks an arithmetic result against the invariant: an overflow (fin128's invalid zero) or a
// denominator past the bound raises instead of producing a value that could not be rebuilt.
func finYearFractionResult(op string, f daycount.Fraction) (Value, error) {
	if err := CheckFinYearFraction(f); err != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(fin.year_fraction %s) result out of range: %s", op, err))
	}
	return NewFinYearFractionValue(f), nil
}

// TypeFinYearFraction is the fin.year_fraction type descriptor.
var TypeFinYearFraction = ValueTypeDescr{
	Name:         ConstHook(finYearFractionTypeName),                                          // PURE by contract
	String:       finYearFractionTypeString,                                                   // PURE by contract
	Format:       finYearFractionTypeFormat,                                                   // PURE by contract
	Interface:    func(v Value) any { return FinYearFractionOf(v) },                           // PURE by contract
	EncodeJSON:   finYearFractionTypeEncodeJSON,                                               // PURE by contract
	EncodeBinary: finYearFractionTypeEncodeBinary,                                             // PURE by contract
	DecodeBinary: finYearFractionTypeDecodeBinary,                                             // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return !FinYearFractionOf(v).IsZero(), nil },  // PURE by contract: falsy iff equal to the default 0/1
	Len:          ConstHook(int64(1)),                                                         // PURE by contract
	Equal:        finYearFractionTypeEqual,                                                    // PURE by contract
	BinaryOp:     finYearFractionTypeBinaryOp,                                                 // PURE by contract
	MethodCall:   finYearFractionTypeMethodCall,                                               // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return FinYearFractionOf(v).String(), true }, // PURE by contract
	IsMethodPure: func(string) bool { return true },                                           // All methods are expected to be pure.
}

// finYearFractionTerms answers the constructor's arguments: [n1, d1] or [n1, d1, n2, d2].
func finYearFractionTerms(f daycount.Fraction) []int64 {
	if f.D2 == 0 {
		return []int64{int64(f.N1), int64(f.D1)}
	}
	return []int64{int64(f.N1), int64(f.D1), int64(f.N2), int64(f.D2)}
}

// PURE by contract
func finYearFractionTypeString(v Value) string {
	t := finYearFractionTerms(FinYearFractionOf(v))
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
func finYearFractionTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return finYearFractionTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(finYearFractionTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 || sp.HasUnconsumedTail() || sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(FinYearFractionOf(v).String(), sp, fspec.AlignLeft), nil
}

// PURE by contract: the terms array, which fin.year_fraction(v...) reads back.
func finYearFractionTypeEncodeJSON(v Value) ([]byte, error) {
	t := finYearFractionTerms(FinYearFractionOf(v))
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
func finYearFractionTypeEncodeBinary(v Value) ([]byte, error) {
	f := FinYearFractionOf(v)
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:], uint32(f.N1))
	binary.LittleEndian.PutUint32(b[4:], uint32(f.D1))
	binary.LittleEndian.PutUint32(b[8:], uint32(f.N2))
	binary.LittleEndian.PutUint32(b[12:], uint32(f.D2))
	return b, nil
}

// IMPURE by contract (mutates target): a corrupt payload is refused, never decoded into an invalid value.
func finYearFractionTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("fin.year_fraction: expected 16 bytes, got %d", len(data))
	}
	f := daycount.Fraction{
		N1: int32(binary.LittleEndian.Uint32(data[0:])),
		D1: int32(binary.LittleEndian.Uint32(data[4:])),
		N2: int32(binary.LittleEndian.Uint32(data[8:])),
		D2: int32(binary.LittleEndian.Uint32(data[12:])),
	}
	if err := CheckFinYearFraction(f); err != nil {
		return fmt.Errorf("fin.year_fraction: %w", err)
	}
	*v = NewFinYearFractionValue(f)
	return nil
}

// finYearFractionCompare orders two fractions exactly: num1·den2 against num2·den1, in big integers (both reduced
// ratios have positive denominators).
func finYearFractionCompare(a, b daycount.Fraction) int {
	an, ad := a.Rational()
	bn, bd := b.Rational()
	l := new(big.Int).Mul(big.NewInt(an), big.NewInt(bd))
	r := new(big.Int).Mul(big.NewInt(bn), big.NewInt(ad))
	return l.Cmp(r)
}

// PURE by contract: numeric equality — 31/365 == 62/730.
func finYearFractionTypeEqual(v Value, other Value, final bool) bool {
	if other.Type == value.FinYearFraction {
		an, ad := FinYearFractionOf(v).Rational()
		bn, bd := FinYearFractionOf(other).Rational()
		return an == bn && ad == bd
	}
	if final {
		return false
	}
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract
func finYearFractionTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	f := FinYearFractionOf(v)
	if reflected {
		// n * f
		if other.Type == value.Int && op == token.Mul {
			return finYearFractionResult("*", f.Scaled(int64(other.Data)))
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	switch other.Type {
	case value.FinYearFraction:
		g := FinYearFractionOf(other)
		switch op {
		case token.Add:
			return finYearFractionResult("+", f.Add(g))
		case token.Less:
			return BoolValue(finYearFractionCompare(f, g) < 0), nil
		case token.Greater:
			return BoolValue(finYearFractionCompare(f, g) > 0), nil
		case token.LessEq:
			return BoolValue(finYearFractionCompare(f, g) <= 0), nil
		case token.GreaterEq:
			return BoolValue(finYearFractionCompare(f, g) >= 0), nil
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
	case value.Int:
		if op == token.Mul {
			return finYearFractionResult("*", f.Scaled(int64(other.Data)))
		}
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func finYearFractionTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	noArgs := func() error {
		if len(args) != 0 {
			return errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return nil
	}
	switch name {
	case "copy", "freeze":
		if err := noArgs(); err != nil {
			return Undefined, err
		}
		return v, nil // immutable: copy and freeze are identities
	case "terms":
		if err := noArgs(); err != nil {
			return Undefined, err
		}
		t := finYearFractionTerms(f)
		out := make([]Value, len(t))
		for i, n := range t {
			out[i] = IntValue(n)
		}
		return NewArrayValue(out, false), nil
	case "value":
		// the decimal reading, at a stated rounding — the only bridge to decimal
		if len(args) != 2 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
		}
		scale, mode, err := decimalScaleModeArgs(name, args, 0)
		if err != nil {
			return Undefined, err
		}
		d, err := f.Value(scale, mode)
		if err != nil {
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
		}
		return decimalResult(name, d)
	case "string":
		return convMember(name, finYearFractionTypeName, args, true, NewStringValue(f.String()))
	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		spec := ""
		if len(args) == 1 {
			if args[0].Type != value.String {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
			spec, _ = args[0].AsString()
		}
		sp, err := fspec.Parse(spec)
		if err != nil {
			return Undefined, errs.FromFormatSpecError(name, err)
		}
		s, err := finYearFractionTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil
	case "is_zero":
		if err := noArgs(); err != nil {
			return Undefined, err
		}
		n, _ := f.Rational()
		return BoolValue(n == 0), nil
	case "is_negative":
		if err := noArgs(); err != nil {
			return Undefined, err
		}
		n, _ := f.Rational()
		return BoolValue(n < 0), nil
	case "is_positive":
		if err := noArgs(); err != nil {
			return Undefined, err
		}
		n, _ := f.Rational()
		return BoolValue(n > 0), nil
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

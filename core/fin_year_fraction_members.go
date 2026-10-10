package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_year_fraction's members: one function per entry of TypeFinYearFraction.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func finYearFractionTerms(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	t := finYearFractionTermsOf(f)
	out := make([]Value, len(t))
	for i, n := range t {
		out[i] = IntValue(n)
	}
	return NewArrayValue(out, false), nil
}

// PURE by contract
func finYearFractionValue(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	f := FinYearFractionOf(v)
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
}

// PURE by contract
func finYearFractionString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	return convMember(id.String(), finYearFractionTypeName, args, true, NewStringValue(f.String()))
}

// PURE by contract
func finYearFractionIsZero(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return BoolValue(n == 0), nil
}

// PURE by contract
func finYearFractionIsNegative(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return BoolValue(n < 0), nil
}

// PURE by contract
func finYearFractionIsPositive(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := FinYearFractionOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return BoolValue(n > 0), nil
}

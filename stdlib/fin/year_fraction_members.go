package fin

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_year_fraction's members: one function per entry of TypeYearFraction.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func yearFractionTerms(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	f := YearFractionOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	t := yearFractionTermsOf(f)
	out := make([]core.Value, len(t))
	for i, n := range t {
		out[i] = core.IntValue(n)
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract
func yearFractionValue(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	f := YearFractionOf(v)
	// the decimal reading, at a stated rounding — the only bridge to decimal
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := core.ScaleModeArgs(name, args, 0)
	if err != nil {
		return core.Undefined, err
	}
	d, err := f.Value(scale, mode)
	if err != nil {
		return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return core.DecimalResult(name, d)
}

// PURE by contract
func yearFractionString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	f := YearFractionOf(v)
	return core.ConvMember(id.String(), yearFractionTypeName, args, true, core.NewStringValue(f.String()))
}

// PURE by contract
func yearFractionIsZero(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	f := YearFractionOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return core.BoolValue(n == 0), nil
}

// PURE by contract
func yearFractionIsNegative(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	f := YearFractionOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return core.BoolValue(n < 0), nil
}

// PURE by contract
func yearFractionIsPositive(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	f := YearFractionOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n, _ := f.Rational()
	return core.BoolValue(n > 0), nil
}

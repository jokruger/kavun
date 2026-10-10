package core

import (
	"fmt"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// decimal's members: one function per entry of TypeDecimal.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func decimalDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), decimalTypeName, args, true, v)
}

// PURE by contract
func decimalFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f, ok := v.AsFloat()
	return convMember(id.String(), decimalTypeName, args, ok, FloatValue(f))
}

// PURE by contract
func decimalInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := v.AsInt()
	return convMember(id.String(), decimalTypeName, args, ok, IntValue(i))
}

// PURE by contract
func decimalTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t, ok := v.AsTime()
	return convMember(id.String(), decimalTypeName, args, ok, NewTimeValue(t))
}

// PURE by contract
func decimalBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsBool()
	return convMember(id.String(), decimalTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func decimalString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// scale-preserving, like every other rendering; canonical() is the value-level way to drop the zeros
	return NewStringValue(o.StringFixed()), nil
}

// PURE by contract
func decimalRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return convMember(id.String(), decimalTypeName, args, ok, NewRunesValue(DecodeText(s), false))
}

// PURE by contract
func decimalIsZero(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(o.IsZero()), nil
}

// PURE by contract
func decimalIsNegative(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(o.IsNegative()), nil
}

// PURE by contract
func decimalIsPositive(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(o.IsPositive()), nil
}

// PURE by contract
func decimalIsInf(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// constant false: dec128 has no Inf representation at all
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return False, nil
}

// PURE by contract
func decimalIsNaN(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(o.IsNaN()), nil
}

// PURE by contract
func decimalErrorDetails(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// a valid decimal has no error details; ErrorDetails() is nil then and must not be dereferenced
	if !o.IsNaN() {
		return Undefined, nil
	}
	if details := o.ErrorDetails(); details != nil {
		return NewErrorValue(NewStringValue(details.Error()), KindUser, errs.CategoryUser, false), nil
	}
	return Undefined, nil
}

// PURE by contract
func decimalSign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Sign())), nil
}

// PURE by contract
func decimalScale(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Scale())), nil
}

// PURE by contract
func decimalRescale(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	// LOSSLESS only: widening pads, narrowing is allowed only when the digits it drops are zeros. A value that
	// would change raises — round(n, mode) is the spelling that changes it, and it names how.
	scale, err := decimalScaleArg(name, "scale", args[0])
	if err != nil {
		return Undefined, err
	}
	r, inexact := o.RescaleRoundInexact(scale, dec128.ROUND_TOWARD_ZERO)
	if inexact {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf(
			"(%s) %s has non-zero digits past scale %d; use round(n, mode) to round it", name, o.StringFixed(), scale))
	}
	return decimalResult(name, r)
}

// PURE by contract
func decimalCanonical(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.Canonical())
}

// PURE by contract
func decimalNextUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.NextUp())
}

// PURE by contract
func decimalNextDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.NextDown())
}

// PURE by contract
func decimalAbs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.Abs())
}

// PURE by contract
func decimalNegate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.Neg())
}

// PURE by contract
func decimalSqrt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return decimalResult(name, o.Sqrt())
}

// PURE by contract
func decimalPow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// Integer exponent only. A negative one is the reciprocal (2d.pow(-1) is 0.5d), so a zero base with a
	// negative exponent is a division by zero and is reported as one rather than as a bad value.
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	exp, err := parseIntArg(name, "exponent", args[0])
	if err != nil {
		return Undefined, err
	}
	if exp < 0 && o.IsZero() {
		return Undefined, errs.NewDivisionByZeroError()
	}
	return decimalResult(name, o.PowInt64(exp))
}

// PURE by contract
func decimalRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// EXACTLY n places (1.5 at 2 is 1.50, as Python's round(Decimal, n) answers); a negative n rounds to tens,
	// hundreds, … and answers a whole number at scale 0.
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	places, err := decimalPlacesArg(name, "places", args[0])
	if err != nil {
		return Undefined, err
	}
	mode, err := decimalModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, decimalRoundTo(*o, places, mode))
}

// the fixed-mode twins of round(n, mode): same contract, the mode spelled in the name; each names its mode through
// decimalRoundingModes, so a twin and round(n, "<mode>") cannot drift

// PURE by contract
func decimalRoundCeiling(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "ceiling")
}

// PURE by contract
func decimalRoundFloor(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "floor")
}

// PURE by contract
func decimalRoundDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "down")
}

// PURE by contract
func decimalRoundUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "up")
}

// PURE by contract
func decimalRoundHalfDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "half_down")
}

// PURE by contract
func decimalRoundHalfUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "half_up")
}

// PURE by contract
func decimalRoundHalfEven(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return decimalRoundFixed(v, id, args, "half_even")
}

// decimalRoundFixed is round_<mode>(places): round(places, mode) with the mode fixed by the member.
func decimalRoundFixed(v Value, id member.ID, args []Value, mode string) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	places, err := decimalPlacesArg(name, "places", args[0])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, decimalRoundTo(*o, places, decimalRoundingModes[mode]))
}

// PURE by contract
func decimalRoundToMultiple(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// the nearest multiple of m: cash rounding to 0.05, "the next whole 10". The result carries m's scale.
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	m, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	if !m.IsPositive() {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the multiple must be positive, got %s", name, m.StringFixed()))
	}
	mode, err := decimalModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.RoundToMultiple(m, mode))
}

// PURE by contract
func decimalRoundSignificant(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// at most k significant digits — the grid a rate is quoted on; a shorter value is answered unchanged
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	digits, err := decimalCountArg(name, "digits", args[0])
	if err != nil {
		return Undefined, err
	}
	mode, err := decimalModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.RoundToSignificant(uint8(min(digits, 255)), mode))
}

// PURE by contract
func decimalDivRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// one operation, one rounding decision against the exact result, landing on exactly the given scale
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "3", len(args))
	}
	other, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 1)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.DivRound(other, scale, mode))
}

// PURE by contract
func decimalMulRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "3", len(args))
	}
	other, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 1)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.MulRound(other, scale, mode))
}

// PURE by contract
func decimalMulPercentRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// x * rate / 100: the percentage is a move of the point, not a second division
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "3", len(args))
	}
	other, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 1)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.MulPercentRound(other, scale, mode))
}

// PURE by contract
func decimalMulAddRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// x*b + c, the intermediate held exactly and a single rounding at the end
	if len(args) != 4 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "4", len(args))
	}
	b, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	c, err := decimalOperandArg(name, "second", args[1])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 2)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.MulAddRound(b, c, scale, mode))
}

// PURE by contract
func decimalMulDivRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// x*b / c, the intermediate held exactly and a single rounding at the end
	if len(args) != 4 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "4", len(args))
	}
	b, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	c, err := decimalOperandArg(name, "second", args[1])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 2)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.MulDivRound(b, c, scale, mode))
}

// PURE by contract
func decimalSqrtRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 0)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.SqrtRound(scale, mode))
}

// PURE by contract
func decimalExpRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// exp/ln/log10/log2 are the only operations here that are not exact: faithfully rounded, within one unit
	// in the last place (dec128 measures them correctly rounded in practice). log10/log2 of an exact power of
	// the base are exact.
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 0)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.Exp(scale, mode))
}

// PURE by contract
func decimalLnRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 0)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.Ln(scale, mode))
}

// PURE by contract
func decimalLog10Round(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 0)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.Log10(scale, mode))
}

// PURE by contract
func decimalLog2Round(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 0)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.Log2(scale, mode))
}

// PURE by contract
func decimalPowRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// an integer power computed with guard digits and rounded once — pow(k) truncates at every step, which is
	// what compounding over thousands of periods cannot afford
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "3", len(args))
	}
	exp, err := parseIntArg(name, "exponent", args[0])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 1)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.PowIntRound(exp, scale, mode))
}

// PURE by contract
func decimalNthRootRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// the inverse of pow_round: the monthly factor of an annual rate is factor.nth_root_round(12, n, mode)
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "3", len(args))
	}
	degree, err := decimalCountArg(name, "degree", args[0])
	if err != nil {
		return Undefined, err
	}
	if degree > decimalMaxRootDegree {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) degree must be at most %d, got %d", name, decimalMaxRootDegree, degree))
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 1)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.NthRootRound(int(degree), scale, mode))
}

// PURE by contract
func decimalPowRationalRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// x^(p/q) in one correctly rounded step; the fraction is reduced first
	if len(args) != 4 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "4", len(args))
	}
	p, err := parseIntArg(name, "numerator", args[0])
	if err != nil {
		return Undefined, err
	}
	q, err := decimalCountArg(name, "denominator", args[1])
	if err != nil {
		return Undefined, err
	}
	scale, mode, err := decimalScaleModeArgs(name, args, 2)
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.PowRational(p, q, scale, mode))
}

// PURE by contract
func decimalQuoRem(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// [quotient, remainder]: the quotient is truncated toward zero (as Python's divmod on Decimal) and the
	// remainder carries the receiver's sign; both are exact and q*y + r == x
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	other, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	q, r := o.QuoRem(other)
	qv, err := decimalResult(name, q)
	if err != nil {
		return Undefined, err
	}
	rv, err := decimalResult(name, r)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue([]Value{qv, rv}, false), nil
}

// PURE by contract
func decimalScaleByPow10(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// x * 10^k as a move of the point: exact, or raises — never rounded
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	k, err := parseIntArg(name, "exponent", args[0])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.ScaleByPow10(int(max(min(k, 1000), -1000))))
}

// PURE by contract
func decimalCopySign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	other, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	return decimalResult(name, o.CopySign(other))
}

// PURE by contract
func decimalClamp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// numeric comparison; the answer keeps its own scale (1.5 clamped to [0, 2.00] is 1.5, 3 is 2.00)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	lo, err := decimalOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	hi, err := decimalOperandArg(name, "second", args[1])
	if err != nil {
		return Undefined, err
	}
	if lo.GreaterThan(hi) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) lower bound %s is above upper bound %s", name, lo.StringFixed(), hi.StringFixed()))
	}
	return decimalResult(name, o.Clamp(lo, hi))
}

// PURE by contract
func decimalIsInteger(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(o.IsInteger()), nil
}

// PURE by contract
func decimalSignificantDigits(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	// the p the value AS WRITTEN needs: 1.50 has 3, because a trailing zero is a digit of the representation
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.SignificantDigits())), nil
}

// PURE by contract
func decimalIntegerDigits(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.IntegerDigits())), nil
}

// PURE by contract
func decimalCanFit(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// would a SQL NUMERIC(precision, scale) column hold this exactly? Trailing zeros are not places the column
	// has to hold, so 1.50 fits NUMERIC(3, 1).
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	precision, err := parseIntArg(name, "precision", args[0])
	if err != nil {
		return Undefined, err
	}
	scale, err := parseIntArg(name, "scale", args[1])
	if err != nil {
		return Undefined, err
	}
	if precision < 1 || precision > 255 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) precision must be between 1 and 255", name))
	}
	if scale < 0 || scale > precision {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and the precision %d", name, precision))
	}
	return BoolValue(o.FitsNumeric(uint8(precision), uint8(scale))), nil
}

// PURE by contract
func decimalSplit(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// n shares that sum to EXACTLY the receiver; the leftover quanta go to the largest remainders (ties to the
	// lowest index)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	count, err := decimalSharesCountArg(name, args[0])
	if err != nil {
		return Undefined, err
	}
	scale, err := decimalScaleArg(name, "scale", args[1])
	if err != nil {
		return Undefined, err
	}
	shares, ok := o.Split(count, scale)
	return decimalShares(name, *o, scale, shares, ok)
}

// PURE by contract
func decimalSplitResidual(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// split(count, scale) with every share rounded by mode, and the share at index taking what is left — the
	// last installment of a schedule, the lead bank of a facility
	if len(args) != 4 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "4", len(args))
	}
	count, err := decimalSharesCountArg(name, args[0])
	if err != nil {
		return Undefined, err
	}
	scale, err := decimalScaleArg(name, "scale", args[1])
	if err != nil {
		return Undefined, err
	}
	idx, err := decimalResidualIndexArg(name, args[2], count)
	if err != nil {
		return Undefined, err
	}
	mode, err := decimalModeArg(name, "mode", args[3])
	if err != nil {
		return Undefined, err
	}
	shares, ok := o.SplitResidual(count, scale, idx, mode)
	return decimalShares(name, *o, scale, shares, ok)
}

// PURE by contract
func decimalAllocate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// shares proportional to ratios, summing to EXACTLY the receiver
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	ratios, err := decimalRatiosArg(name, args[0])
	if err != nil {
		return Undefined, err
	}
	scale, err := decimalScaleArg(name, "scale", args[1])
	if err != nil {
		return Undefined, err
	}
	shares, ok := o.Allocate(ratios, scale)
	return decimalShares(name, *o, scale, shares, ok)
}

// PURE by contract
func decimalAllocateResidual(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*dec128.Dec128)(v.Ptr)
	// allocate(ratios, scale) with the residual taken by the share at index, as for split_residual
	if len(args) != 4 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "4", len(args))
	}
	ratios, err := decimalRatiosArg(name, args[0])
	if err != nil {
		return Undefined, err
	}
	scale, err := decimalScaleArg(name, "scale", args[1])
	if err != nil {
		return Undefined, err
	}
	idx, err := decimalResidualIndexArg(name, args[2], len(ratios))
	if err != nil {
		return Undefined, err
	}
	mode, err := decimalModeArg(name, "mode", args[3])
	if err != nil {
		return Undefined, err
	}
	shares, ok := o.AllocateResidual(ratios, scale, idx, mode)
	return decimalShares(name, *o, scale, shares, ok)
}

package core

import (
	"math"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// float's members: one function per entry of TypeFloat.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func floatFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), floatTypeName, args, true, v)
}

// PURE by contract
func floatDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := math.Float64frombits(v.Data)
	// a NaN decimal is an error state, never a produced value: NaN/Inf decline, and so does a finite float
	// too large for dec128's 128-bit coefficient — FromFloat64 signals that overflow as NaN too
	ok := !math.IsInf(f, 0) && !math.IsNaN(f)
	var d Value
	if ok {
		dd := dec128.FromFloat64(f)
		if ok = !dd.IsNaN(); ok {
			d = NewDecimalValue(dd)
		}
	}
	return convMember(id.String(), floatTypeName, args, ok, d)
}

// PURE by contract
func floatInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := v.AsInt()
	return convMember(id.String(), floatTypeName, args, ok, IntValue(i))
}

// PURE by contract
func floatBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsBool()
	return convMember(id.String(), floatTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func floatTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t, ok := v.AsTime()
	return convMember(id.String(), floatTypeName, args, ok, NewTimeValue(t))
}

// PURE by contract
func floatString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	s, _ := v.AsString()
	return NewStringValue(s), nil
}

// PURE by contract
func floatRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return convMember(id.String(), floatTypeName, args, ok, NewRunesValue(DecodeText(s), false))
}

// PURE by contract
func floatIsNaN(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.IsNaN(math.Float64frombits(v.Data))), nil
}

// PURE by contract
func floatIsInf(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// no sign argument: -Inf is x.is_inf() && x.sign() < 0
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.IsInf(math.Float64frombits(v.Data), 0)), nil
}

// PURE by contract
func floatIsZero(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) == 0), nil
}

// PURE by contract
func floatIsNegative(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) < 0), nil
}

// PURE by contract
func floatIsPositive(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) > 0), nil
}

// PURE by contract
func floatAbs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return FloatValue(math.Abs(math.Float64frombits(v.Data))), nil
}

// PURE by contract
func floatSign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) {
		return IntValue(0), nil
	}
	if f > 0 {
		return IntValue(1), nil
	}
	if f < 0 {
		return IntValue(-1), nil
	}
	return IntValue(0), nil
}

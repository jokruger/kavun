package core

import (
	"math"
	"time"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// int's members: one function per entry of TypeInt.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func intInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), intTypeName, args, true, v)
}

// PURE by contract
func intFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f, ok := v.AsFloat()
	return convMember(id.String(), intTypeName, args, ok, FloatValue(f))
}

// PURE by contract
func intDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d, ok := v.AsDecimal()
	return convMember(id.String(), intTypeName, args, ok, NewDecimalValue(d))
}

// PURE by contract
func intBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsBool()
	return convMember(id.String(), intTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func intRune(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	c, ok := v.AsRune()
	return convMember(id.String(), intTypeName, args, ok, RuneValue(c))
}

// PURE by contract
func intByte(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsByte()
	return convMember(id.String(), intTypeName, args, ok, ByteValue(b))
}

// PURE by contract
func intString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// total — the default slot never fires, but every conversion carries it
	s, _ := v.AsString()
	return convMember(id.String(), intTypeName, args, true, NewStringValue(s))
}

// PURE by contract
func intRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return convMember(id.String(), intTypeName, args, ok, NewRunesValue(DecodeText(s), false))

	// The int -> time family. In conversion context an int is a unix timestamp, never a duration
	// (that reading belongs to operator context — `t + n` is nanoseconds; see docs/types/time.md).
	// Each of these names the encoding it reads, and each is the exact inverse of the time accessor
	// with the matching suffix: time_ms <-> unix_ms, time_micro <-> unix_micro, time_nano <->
	// unix_nano, and the unsuffixed time() <-> int()/unix(), which are seconds. All produce UTC, so
	// the result never depends on the host's timezone.
}

// PURE by contract
func intTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t, _ := v.AsTime()
	return convMember(id.String(), intTypeName, args, true, NewTimeValue(t))
}

// PURE by contract
func intDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// in conversion context an int is a day count since 1970-01-01 — the inverse of d.int(); in operator
	// context (d + n) it is a number of days
	dd, ok := intTypeAsDate(v)
	return convMember(id.String(), intTypeName, args, ok, DateValue(dd))
}

// PURE by contract
func intTimeMs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), intTypeName, args, true, NewTimeValue(time.UnixMilli(int64(v.Data)).UTC()))
}

// PURE by contract
func intTimeMicro(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), intTypeName, args, true, NewTimeValue(time.UnixMicro(int64(v.Data)).UTC()))
}

// PURE by contract
func intTimeNano(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), intTypeName, args, true, NewTimeValue(time.Unix(0, int64(v.Data)).UTC()))
}

// PURE by contract
func intIsNaN(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return False, nil
}

// PURE by contract
func intIsInf(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return False, nil
}

// PURE by contract
func intSign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	if v.Data == 0 {
		return IntValue(0), nil
	} else if int64(v.Data) > 0 {
		return IntValue(1), nil
	} else {
		return IntValue(-1), nil
	}
}

// PURE by contract
func intAbs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	i := int64(v.Data)
	if i == math.MinInt64 {
		// |MinInt64| does not fit — int is checked, it never wraps
		return Undefined, errs.NewInvalidValueError("int overflow")
	}
	if i < 0 {
		return IntValue(-i), nil
	}
	return v, nil
}

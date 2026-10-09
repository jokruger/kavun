package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// bool's members: one function per entry of TypeBool.Methods. A bool is always immutable, so copy and freeze answer
// the receiver itself.

// PURE by contract
func boolString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	s, _ := boolTypeAsString(v)
	return NewStringValue(s), nil
}

// PURE by contract
func boolFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	f := ""
	if len(args) == 1 {
		var ok bool
		f, ok = args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
		}
	}
	sp, err := fspec.Parse(f)
	if err != nil {
		return Undefined, errs.FromFormatSpecError(name, err)
	}
	s, err := boolTypeFormat(v, sp)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(s), nil
}

// PURE by contract — the receiver itself: a bool is always immutable, so a copy of any depth is the same value
func boolCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v, nil
}

// PURE by contract — the receiver itself: a bool is always immutable already
func boolFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v, nil
}

// PURE by contract
func boolRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return convMember(id.String(), boolTypeName, args, ok, NewRunesValue([]rune(s), false))
}

// PURE by contract
func boolInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, _ := boolTypeAsInt(v)
	return convMember(id.String(), boolTypeName, args, true, IntValue(i))
}

// PURE by contract
func boolBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), boolTypeName, args, true, v)
}

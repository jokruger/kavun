package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/text"
)

// bool's members: one function per entry of TypeBool.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func boolString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	s, _ := boolTypeAsString(v)
	return NewStringValue(s), nil
}

// PURE by contract
func boolRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return ConvMember(id.String(), boolTypeName, args, ok, NewRunesValue(text.DecodeText(s), false))
}

// PURE by contract
func boolInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, _ := boolTypeAsInt(v)
	return ConvMember(id.String(), boolTypeName, args, true, IntValue(i))
}

// PURE by contract
func boolBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return ConvMember(id.String(), boolTypeName, args, true, v)
}

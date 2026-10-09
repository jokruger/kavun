package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// byte's members: one function per entry of TypeByte.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func byteByte(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), byteTypeName, args, true, v)
}

// PURE by contract
func byteInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := v.AsInt()
	return convMember(id.String(), byteTypeName, args, ok, IntValue(i))
}

// PURE by contract
func byteRune(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	c, ok := v.AsRune()
	return convMember(id.String(), byteTypeName, args, ok, RuneValue(c))
}

// PURE by contract
func byteString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// TOTAL: a byte always has text content — its ASCII symbol below 0x80, and above it the
	// one-octet text that decodes to this octet's escape. The render is unmoved: b'A'.format() -> "65"
	return convMember(id.String(), byteTypeName, args, true, NewStringValue(string([]byte{byte(v.Data)})))
}

// PURE by contract
func byteRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), byteTypeName, args, true, NewRunesValue(DecodeOctets([]byte{byte(v.Data)}), false))
}

// PURE by contract
func byteIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(byte(v.Data) < 0x80), nil
}

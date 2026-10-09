package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// TypeByte's members: one function per entry of TypeByte.Methods.

// PURE by contract
func byteCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// it is always immutable, so we can return the same value regardless of copy depth
	return v, nil
}

// PURE by contract
func byteFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// it is always immutable already, so freeze/freeze_shallow are no-ops
	return v, nil
}

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

// PURE by contract
func byteFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
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
	s, err := byteTypeFormat(v, sp)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(s), nil
}

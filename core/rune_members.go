package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/text"
)

// rune's members: one function per entry of TypeRune.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func runeRune(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return ConvMember(id.String(), runeTypeName, args, true, v)
}

// PURE by contract
func runeInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return ConvMember(id.String(), runeTypeName, args, true, IntValue(int64(v.Data)))
}

// PURE by contract
func runeByte(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsByte()
	return ConvMember(id.String(), runeTypeName, args, ok, ByteValue(b))
}

// PURE by contract
func runeString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// total — the default slot never fires, but every conversion carries it. An escape rune
	// answers the single octet it stands for, so it round-trips back to the data it came from
	return ConvMember(id.String(), runeTypeName, args, true, NewStringValue(text.EncodeRuneText(rune(v.Data))))
}

// PURE by contract
func runeRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the text targets compose through string: 'A'.runes() ≡ 'A'.string().runes()
	return ConvMember(id.String(), runeTypeName, args, true, NewRunesValue([]rune{rune(v.Data)}, false))
}

// PURE by contract
func runeIsValid(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// a real symbol, as opposed to an escape standing for an octet that is not one
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(text.RuneIsValid(rune(v.Data))), nil
}

// PURE by contract
func runeIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(rune(v.Data) < 0x80), nil
}

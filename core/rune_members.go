package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// TypeRune's members: one function per entry of TypeRune.Methods.

// PURE by contract
func runeCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// it is always immutable, so we can return the same value regardless of copy depth
	return v, nil
}

// PURE by contract
func runeFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	// it is always immutable already, so freeze/freeze_shallow are no-ops
	return v, nil
}

// PURE by contract
func runeRune(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), runeTypeName, args, true, v)
}

// PURE by contract
func runeInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), runeTypeName, args, true, IntValue(int64(v.Data)))
}

// PURE by contract
func runeByte(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsByte()
	return convMember(id.String(), runeTypeName, args, ok, ByteValue(b))
}

// PURE by contract
func runeString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// total — the default slot never fires, but every conversion carries it. An escape rune
	// answers the single octet it stands for, so it round-trips back to the data it came from
	return convMember(id.String(), runeTypeName, args, true, NewStringValue(EncodeRuneText(rune(v.Data))))
}

// PURE by contract
func runeRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the text targets compose through string: 'A'.runes() ≡ 'A'.string().runes()
	return convMember(id.String(), runeTypeName, args, true, NewRunesValue([]rune{rune(v.Data)}, false))
}

// PURE by contract
func runeIsValid(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// a real symbol, as opposed to an escape standing for an octet that is not one
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(RuneIsValid(rune(v.Data))), nil
}

// PURE by contract
func runeIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(rune(v.Data) < 0x80), nil
}

// PURE by contract
func runeFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
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
	s, err := runeTypeFormat(v, sp)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(s), nil
}

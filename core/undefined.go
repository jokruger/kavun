package core

import (
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const undefinedTypeName = "undefined"

var TypeUndefined = ValueTypeDescr{
	Name:         ConstHook(undefinedTypeName),                                  // PURE by contract
	String:       func(Value) string { return undefinedTypeName },               // PURE by contract
	Format:       undefinedTypeFormat,                                           // PURE by contract
	EncodeJSON:   func(Value) ([]byte, error) { return []byte("null"), nil },    // PURE by contract
	EncodeBinary: func(Value) ([]byte, error) { return []byte{}, nil },          // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { *v = Undefined; return nil }, // IMPURE by contract (mutates target)
	Interface:    func(Value) any { return nil },                                // PURE by contract
	IsTrue:       Const2Hook[bool, error](false, nil),                           // PURE by contract

	// Undefined propagates on the DATA plane (selectors, indexing, slicing, operators — a chain like a.b.c
	// misses at any level and answers one undefined) and raises on the ACTION plane (call, iteration,
	// membership, members). Getting an iterator is an action, so IsIterable answers false and for-in raises.
	IsIterable: ConstHook(false), // PURE by contract

	Equal:               undefinedTypeEqual,                                                        // PURE by contract
	UnaryOp:             undefinedTypeUnaryOp,                                                      // PURE by contract
	BinaryOp:            undefinedTypeBinaryOp,                                                     // PURE by contract
	AccessIndex:         func(Value, Value) (Value, error) { return Undefined, nil },               // PURE by contract
	AccessNamedProperty: func(VM, Value, string) (Value, error) { return Undefined, nil },          // PURE by contract
	Slice:               func(Value, Value, Value) (Value, error) { return Undefined, nil },        // PURE by contract
	SliceStep:           func(Value, Value, Value, Value) (Value, error) { return Undefined, nil }, // PURE by contract
	AsBool:              func(Value) (bool, bool) { return false, true },                           // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:  {Fn: memberIsTrue, Pure: true},
		members.String:  {Fn: undefinedConvert, Pure: true},
		members.Format:  {Fn: memberFormat, Pure: true},
		members.Runes:   {Fn: undefinedConvert, Pure: true},
		members.Int:     {Fn: undefinedConvert, Pure: true},
		members.Bool:    {Fn: undefinedConvert, Pure: true},
		members.Float:   {Fn: undefinedConvert, Pure: true},
		members.Time:    {Fn: undefinedConvert, Pure: true},
		members.Decimal: {Fn: undefinedConvert, Pure: true},
		members.Date:    {Fn: undefinedConvert, Pure: true},
		members.Array:   {Fn: undefinedConvert, Pure: true},
		members.Bytes:   {Fn: undefinedConvert, Pure: true},
		members.Byte:    {Fn: undefinedConvert, Pure: true},
		members.Rune:    {Fn: undefinedConvert, Pure: true},
		members.Dict:    {Fn: undefinedConvert, Pure: true},
		members.Record:  {Fn: undefinedConvert, Pure: true},
	},
}

// PURE by contract
func undefinedTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return undefinedTypeName, nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
}

func undefinedTypeEqual(v Value, other Value, _ bool) bool {
	// undefined is only equal to undefined
	return v.Type == other.Type
}

func undefinedTypeBinaryOp(Value, Value, token.Token, bool) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}

func undefinedTypeUnaryOp(Value, token.Token) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}

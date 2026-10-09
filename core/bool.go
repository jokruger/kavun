package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const boolTypeName = "bool"

func BoolValue(b bool) Value {
	if b {
		return True
	}
	return False
}

var TypeBool = ValueTypeDescr{
	Name:         ConstHook(boolTypeName),                                 // PURE by contract
	String:       boolTypeString,                                          // PURE by contract
	Format:       boolTypeFormat,                                          // PURE by contract
	Interface:    func(v Value) any { return v.Data != 0 },                // PURE by contract
	EncodeJSON:   boolTypeEncodeJSON,                                      // PURE by contract
	EncodeBinary: boolTypeEncodeBinary,                                    // PURE by contract
	DecodeBinary: boolTypeDecodeBinary,                                    // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil }, // PURE by contract
	Equal:        boolTypeEqual,                                           // PURE by contract
	BinaryOp:     boolTypeBinaryOp,                                        // PURE by contract
	UnaryOp:      boolTypeUnaryOp,                                         // PURE by contract
	Len:          ConstHook(int64(1)),                                     // PURE by contract
	AsString:     boolTypeAsString,                                        // PURE by contract
	AsInt:        boolTypeAsInt,                                           // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true }, // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue: {Fn: memberIsTrue, Pure: true},
		members.String: {Fn: boolString, Pure: true},
		members.Format: {Fn: memberFormat, Pure: true},
		members.Copy:   {Fn: memberSelf, Pure: true},
		members.Freeze: {Fn: memberSelf, Pure: true},
		members.Runes:  {Fn: boolRunes, Pure: true},
		members.Int:    {Fn: boolInt, Pure: true},
		members.Bool:   {Fn: boolBool, Pure: true},
	},
}

func boolTypeEncodeJSON(v Value) ([]byte, error) {
	s := boolTypeString(v)
	return []byte(s), nil
}

func boolTypeEncodeBinary(v Value) ([]byte, error) {
	return []byte{uint8(v.Data)}, nil
}

func boolTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("bool: expected 1 byte, got %d", len(data))
	}
	v.Data = uint64(data[0])
	return nil
}

func boolTypeString(v Value) string {
	if v.Data == 0 {
		return "false"
	}
	return "true"
}

func boolTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	var body string
	switch sp.Verb {
	case 'v':
		return boolTypeString(v), nil

	case 'T':
		body = boolTypeName

	case 0, 't':
		if v.Data == 0 {
			body = "false"
		} else {
			body = "true"
		}

	case 'd':
		if v.Data == 0 {
			body = "0"
		} else {
			body = "1"
		}

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	return fspec.ApplyGenerics(body, sp, fspec.AlignLeft), nil
}

func boolTypeAsString(v Value) (string, bool) {
	if v.Data == 0 {
		return "false", true
	}
	return "true", true
}

func boolTypeAsInt(v Value) (int64, bool) {
	if v.Data == 0 {
		return 0, true
	}
	return 1, true
}

func boolTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Bool:
		return v.Data == other.Data
	}

	// default to false if final
	if final {
		return false
	}

	// delegate to other type's Equal if not final
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func boolTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		// bool always recognizes same-type non-reflected, so it can never be reached reflected — nothing ever needs to
		// delegate into bool.
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Bool:
		l := v.Data != 0
		r := other.Data != 0
		switch op {
		case tokens.Less:
			return BoolValue(!l && r), nil
		case tokens.Greater:
			return BoolValue(l && !r), nil
		case tokens.LessEq:
			return BoolValue(!l || r), nil
		case tokens.GreaterEq:
			return BoolValue(l || !r), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract.
func boolTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case tokens.Xor:
		return BoolValue(v.Data == 0), nil
	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

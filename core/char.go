package core

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

// CharValue creates new char value.
func CharValue(c rune) Value {
	return Value{
		Data: uint64(c),
		Type: VT_CHAR,
	}
}

// ToChar converts boxed char value to rune. It is a caller's responsibility to ensure the type is correct.
func ToChar(v Value) rune {
	return rune(v.Data)
}

/* Char type methods */

func charTypeName(v Value) string {
	return "char"
}

func charTypeEncodeJSON(v Value) ([]byte, error) {
	c := ToChar(v)
	s := strconv.FormatInt(int64(c), 10)
	return []byte(s), nil
}

func charTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v.Data))
	return b, nil
}

func charTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("char: expected 4 bytes, got %d", len(data))
	}
	v.Data = uint64(binary.BigEndian.Uint32(data))
	return nil
}

func charTypeString(v Value) string {
	return fmt.Sprintf("%q", ToChar(v))
}

func charTypeInterface(v Value) any {
	return ToChar(v)
}

func charTypeIsTrue(v Value) bool {
	return ToChar(v) != 0
}

func charTypeAsInt(v Value) (int64, bool) {
	return int64(ToChar(v)), true
}

func charTypeAsString(v Value) (string, bool) {
	return string(ToChar(v)), true
}

func charTypeAsBool(v Value) (bool, bool) {
	return ToChar(v) != 0, true
}

func charTypeAsChar(v Value) (rune, bool) {
	return ToChar(v), true
}

func charTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsChar()
	if !ok {
		return false
	}
	return ToChar(v) == r
}

func charTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_char":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("char.to_char", "0", len(args))
		}
		return v, nil

	case "to_bool":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("char.to_bool", "0", len(args))
		}
		b, _ := charTypeAsBool(v)
		return BoolValue(b), nil

	case "to_int":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("char.to_int", "0", len(args))
		}
		i, _ := int64(ToChar(v)), true
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("char.to_string", "0", len(args))
		}
		s, _ := charTypeAsString(v)
		return vm.Allocator().NewStringValue(s), nil

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, "char")
	}
}

func charTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	switch rhs.Type {
	case VT_INT: // char op int => int
		l := int64(ToChar(v))
		r := ToInt(rhs)
		switch op {
		case token.Add:
			return IntValue(l + r), nil
		case token.Sub:
			return IntValue(l - r), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		default:
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	case VT_STRING: // char op string => string
		l := string(ToChar(v))
		r, _ := stringTypeAsString(rhs)
		switch op {
		case token.Add:
			return a.NewStringValue(l + r), nil
		default:
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// char op any => char
		r, ok := rhs.AsChar()
		if !ok {
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := ToChar(v)
		switch op {
		case token.Add:
			return CharValue(l + r), nil
		case token.Sub:
			return CharValue(l - r), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		default:
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}
	}
}

package core

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

// IntValue creates new boxed int value.
func IntValue(i int64) Value {
	return Value{
		Data: uint64(i),
		Type: VT_INT,
	}
}

// ToInt converts boxed int value to int64. It is a caller's responsibility to ensure the type is correct.
func ToInt(v Value) int64 {
	return int64(v.Data)
}

/* Int type methods */

func intTypeName(v Value) string {
	return "int"
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(ToInt(v), 10)
	return []byte(s), nil
}

func intTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v.Data)
	return b, nil
}

func intTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("int: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.BigEndian.Uint64(data)
	return nil
}

func intTypeString(v Value) string {
	return strconv.FormatInt(ToInt(v), 10)
}

func intTypeInterface(v Value) any {
	return ToInt(v)
}

func intTypeIsTrue(v Value) bool {
	return ToInt(v) != 0
}

func intTypeAsInt(v Value) (int64, bool) {
	return ToInt(v), true
}

func intTypeAsString(v Value) (string, bool) {
	return strconv.FormatInt(ToInt(v), 10), true
}

func intTypeAsFloat(v Value) (float64, bool) {
	return float64(ToInt(v)), true
}

func intTypeAsBool(v Value) (bool, bool) {
	return ToInt(v) != 0, true
}

func intTypeAsChar(v Value) (rune, bool) {
	return rune(ToInt(v)), true
}

func intTypeAsTime(v Value) (time.Time, bool) {
	return time.Unix(ToInt(v), 0), true
}

func intTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsInt()
	if !ok {
		return false
	}
	return ToInt(v) == r
}

func intTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_int", "0", len(args))
		}
		return v, nil

	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_float", "0", len(args))
		}
		f, _ := v.AsFloat()
		return FloatValue(f), nil

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_bool", "0", len(args))
		}
		b, _ := v.AsBool()
		return BoolValue(b), nil

	case "to_char":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_char", "0", len(args))
		}
		c, _ := v.AsChar()
		return CharValue(c), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_string", "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s)

	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("int.to_time", "0", len(args))
		}
		t, _ := v.AsTime()
		return vm.Allocator().NewTimeValue(t)

	default:
		return Undefined, errs.NewInvalidMethodError(name, "int")
	}
}

func intTypeUnaryOp(v Value, a Allocator, op token.Token) (Value, error) {
	i := ToInt(v)
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		return IntValue(-i), nil

	case token.Xor: // see also fast track in VM OpBComplement
		return IntValue(^i), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func intTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	// see also int/int fast track in VM OpBinaryOp

	switch rhs.Type {
	case VT_FLOAT: // int op float => float
		l := float64(ToInt(v))
		r := ToFloat(rhs)
		switch op {
		case token.Add:
			return FloatValue(l + r), nil
		case token.Sub:
			return FloatValue(l - r), nil
		case token.Mul:
			return FloatValue(l * r), nil
		case token.Quo:
			return FloatValue(l / r), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// int op any => int
		r, ok := rhs.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := ToInt(v)
		switch op {
		case token.Add:
			return IntValue(l + r), nil
		case token.Sub:
			return IntValue(l - r), nil
		case token.Mul:
			return IntValue(l * r), nil
		case token.Quo:
			return IntValue(l / r), nil
		case token.Rem:
			return IntValue(l % r), nil
		case token.And:
			return IntValue(l & r), nil
		case token.Or:
			return IntValue(l | r), nil
		case token.Xor:
			return IntValue(l ^ r), nil
		case token.AndNot:
			return IntValue(l &^ r), nil
		case token.Shl:
			return IntValue(l << uint64(r)), nil
		case token.Shr:
			return IntValue(l >> uint64(r)), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}
	}
}

package core

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

// IntValue creates new boxed int value.
func IntValue(i int64) Value {
	return Value{
		Type:  VT_INT,
		Const: true,
		Data:  uint64(i),
	}
}

/* Int type methods */

func intTypeName(v Value) string {
	return "int"
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(int64(v.Data), 10)
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
	return strconv.FormatInt(int64(v.Data), 10)
}

func intTypeInterface(v Value) any {
	return int64(v.Data)
}

func intTypeIsTrue(v Value) bool {
	return v.Data != 0
}

func intTypeAsInt(v Value) (int64, bool) {
	return int64(v.Data), true
}

func intTypeAsString(v Value) (string, bool) {
	return strconv.FormatInt(int64(v.Data), 10), true
}

func intTypeAsFloat(v Value) (float64, bool) {
	return float64(int64(v.Data)), true
}

func intTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	return dec128.FromInt64(int64(v.Data)), true
}

func intTypeAsBool(v Value) (bool, bool) {
	return v.Data != 0, true
}

func intTypeAsRune(v Value) (rune, bool) {
	i := int64(v.Data)
	if i < 0 || i > utf8.MaxRune {
		return rune(i), false
	}
	return rune(i), true
}

func intTypeAsByte(v Value) (byte, bool) {
	i := int64(v.Data)
	if i < 0 || i > math.MaxUint8 {
		return byte(i), false
	}
	return byte(i), true
}

func intTypeAsTime(v Value) (time.Time, bool) {
	return time.Unix(int64(v.Data), 0), true
}

func intTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsInt()
	if !ok {
		return false
	}
	return int64(v.Data) == r
}

func intTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := v.AsFloat()
		return FloatValue(f), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := v.AsDecimal()
		alloc := vm.Allocator()
		r := alloc.NewDecimal()
		*r = d
		return DecimalValue(r), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := v.AsBool()
		return BoolValue(b), nil

	case "rune":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		c, _ := v.AsRune()
		return RuneValue(c), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := v.AsByte()
		return ByteValue(b), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := v.AsTime()
		d := vm.Allocator().NewTime()
		*d = t
		return TimeValue(d), nil

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Data == 0 {
			return IntValue(0), nil
		} else if int64(v.Data) > 0 {
			return IntValue(1), nil
		} else {
			return IntValue(-1), nil
		}

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i := int64(v.Data)
		if i < 0 {
			return IntValue(-i), nil
		}
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "int")
	}
}

func intTypeUnaryOp(v Value, a *Arena, op token.Token) (Value, error) {
	i := int64(v.Data)
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		return IntValue(-i), nil

	case token.Xor: // see also fast track in VM OpBComplement
		return IntValue(^i), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func intTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	// see also int/int fast track in VM OpBinaryOp

	switch rhs.Type {
	case VT_FLOAT: // int op float => float
		l := float64(int64(v.Data))
		r := math.Float64frombits(rhs.Data)
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

	case VT_DECIMAL: // int op decimal => decimal
		l := dec128.FromInt64(int64(v.Data))
		r := (*dec128.Dec128)(rhs.Ptr)
		switch op {
		case token.Add:
			d := a.NewDecimal()
			*d = l.Add(*r)
			return DecimalValue(d), nil
		case token.Sub:
			d := a.NewDecimal()
			*d = l.Sub(*r)
			return DecimalValue(d), nil
		case token.Mul:
			d := a.NewDecimal()
			*d = l.Mul(*r)
			return DecimalValue(d), nil
		case token.Quo:
			d := a.NewDecimal()
			*d = l.Div(*r)
			return DecimalValue(d), nil
		case token.Less:
			return BoolValue(l.LessThan(*r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(*r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(*r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(*r)), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// int op any => int
		r, ok := rhs.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := int64(v.Data)
		switch op {
		case token.Add:
			return IntValue(l + r), nil
		case token.Sub:
			return IntValue(l - r), nil
		case token.Mul:
			return IntValue(l * r), nil
		case token.Quo:
			if r == 0 {
				return Undefined, errs.ErrDivisionByZero
			}
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

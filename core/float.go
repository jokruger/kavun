package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

// FloatValue creates new boxed float value.
func FloatValue(f float64) Value {
	return Value{
		Data: math.Float64bits(f),
		Type: VT_FLOAT,
	}
}

/* Float type methods */

func floatTypeName(v Value) string {
	return "float"
}

func floatTypeEncodeJSON(v Value) ([]byte, error) {
	var y []byte

	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) {
		return nil, errors.New("unsupported Inf value")
	}
	if math.IsNaN(f) {
		return nil, errors.New("unsupported NaN value")
	}

	// Convert as if by ES6 number to string conversion.
	// This matches most other JSON generators.
	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	y = strconv.AppendFloat(y, f, fmt, -1, 64)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(y)
		if n >= 4 && y[n-4] == 'e' && y[n-3] == '-' && y[n-2] == '0' {
			y[n-2] = y[n-1]
			y = y[:n-1]
		}
	}

	return y, nil
}

func floatTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v.Data)
	return b, nil
}

func floatTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("float: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.BigEndian.Uint64(data)
	return nil
}

func floatTypeString(v Value) string {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64)
}

func floatTypeInterface(v Value) any {
	return math.Float64frombits(v.Data)
}

func floatTypeIsTrue(v Value) bool {
	return !math.IsNaN(math.Float64frombits(v.Data))
}

func floatTypeAsInt(v Value) (int64, bool) {
	return int64(math.Float64frombits(v.Data)), true
}

func floatTypeAsString(v Value) (string, bool) {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), true
}

func floatTypeAsFloat(v Value) (float64, bool) {
	return math.Float64frombits(v.Data), true
}

func floatTypeAsDecimal(v Value) (Decimal, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return dec128.NaN(state.NaN), false
	}
	return dec128.FromFloat64(f), true
}

func floatTypeAsBool(v Value) (bool, bool) {
	return !math.IsNaN(math.Float64frombits(v.Data)), true
}

func floatTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsFloat()
	if !ok {
		return false
	}
	return math.Float64frombits(v.Data) == r
}

func floatTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f := math.Float64frombits(v.Data)
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return vm.Allocator().NewDecimalValue(dec128.NaN(state.NaN))
		}
		return vm.Allocator().NewDecimalValue(dec128.FromFloat64(f))

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := v.AsInt()
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s)

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f := math.Float64frombits(v.Data)
		if math.IsNaN(f) {
			return IntValue(0), nil
		}
		if f > 0 {
			return IntValue(1), nil
		}
		if f < 0 {
			return IntValue(-1), nil
		}
		return IntValue(0), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "float")
	}
}

func floatTypeUnaryOp(v Value, a Allocator, op token.Token) (Value, error) {
	f := math.Float64frombits(v.Data)
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		return FloatValue(-f), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func floatTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsFloat()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := math.Float64frombits(v.Data)
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
}

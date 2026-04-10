package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/jokruger/gs/errs"
)

func FloatValue(f float64) Value {
	return Value{
		Data: math.Float64bits(f),
		Type: VT_FLOAT,
	}
}

func toFloat(v Value) float64 {
	return math.Float64frombits(v.Data)
}

func floatTypeName(v Value) string {
	return "float"
}

func floatTypeEncodeJSON(v Value) ([]byte, error) {
	var y []byte

	f := toFloat(v)
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
	return strconv.FormatFloat(toFloat(v), 'f', -1, 64)
}

func floatTypeInterface(v Value) any {
	return toFloat(v)
}

func floatTypeIsTrue(v Value) bool {
	return !math.IsNaN(toFloat(v))
}

func floatTypeAsString(v Value) (string, bool) {
	return strconv.FormatFloat(toFloat(v), 'f', -1, 64), true
}

func floatTypeAsFloat(v Value) (float64, bool) {
	return toFloat(v), true
}

func floatTypeAsBool(v Value) (bool, bool) {
	return !math.IsNaN(toFloat(v)), true
}

func floatTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsFloat()
	if !ok {
		return false
	}
	return toFloat(v) == r
}

func floatTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_float":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("float.to_float", "0", len(args))
		}
		return v, nil

	case "to_int":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("float.to_int", "0", len(args))
		}
		i, _ := v.AsInt()
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("float.to_string", "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s), nil

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, "float")
	}
}

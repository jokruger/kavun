package core

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/jokruger/gs/errs"
)

func IntValue(i int64) Value {
	return Value{
		Data: uint64(i),
		Type: VT_INT,
	}
}

func toInt(v Value) int64 {
	return int64(v.Data)
}

func intTypeName(v Value) string {
	return "int"
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(toInt(v), 10)
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
	return strconv.FormatInt(toInt(v), 10)
}

func intTypeInterface(v Value) any {
	return toInt(v)
}

func intTypeIsTrue(v Value) bool {
	return toInt(v) != 0
}

func intTypeAsString(v Value) (string, bool) {
	return strconv.FormatInt(toInt(v), 10), true
}

func intTypeAsFloat(v Value) (float64, bool) {
	return float64(toInt(v)), true
}

func intTypeAsBool(v Value) (bool, bool) {
	return toInt(v) != 0, true
}

func intTypeAsChar(v Value) (rune, bool) {
	return rune(toInt(v)), true
}

func intTypeAsTime(v Value) (time.Time, bool) {
	return time.Unix(toInt(v), 0), true
}

func intTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsInt()
	if !ok {
		return false
	}
	return toInt(v) == r
}

func intTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_int":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_int", "0", len(args))
		}
		return v, nil

	case "to_float":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_float", "0", len(args))
		}
		f, _ := v.AsFloat()
		return FloatValue(f), nil

	case "to_bool":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_bool", "0", len(args))
		}
		b, _ := v.AsBool()
		return BoolValue(b), nil

	case "to_char":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_char", "0", len(args))
		}
		c, _ := v.AsChar()
		return CharValue(c), nil

	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_string", "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s), nil

	case "to_time":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("int.to_time", "0", len(args))
		}
		t, _ := v.AsTime()
		return vm.Allocator().NewTimeValue(t), nil

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, "int")
	}
}

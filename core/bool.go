package core

import (
	"fmt"

	"github.com/jokruger/gs/errs"
)

func BoolValue(b bool) Value {
	var v Value
	if b {
		v.Data = 1
	}
	v.Type = VT_BOOL
	return v
}

func toBool(v Value) bool {
	return v.Data != 0
}

func boolTypeName(v Value) string {
	return "bool"
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
	if toBool(v) {
		return "true"
	}
	return "false"
}

func boolTypeInterface(v Value) any {
	return toBool(v)
}

func boolTypeIsTrue(v Value) bool {
	return toBool(v)
}

func boolTypeAsString(v Value) (string, bool) {
	if toBool(v) {
		return "true", true
	}
	return "false", true
}

func boolTypeAsInt(v Value) (int64, bool) {
	if toBool(v) {
		return 1, true
	}
	return 0, true
}

func boolTypeAsBool(v Value) (bool, bool) {
	return toBool(v), true
}

func boolTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsBool()
	if !ok {
		return false
	}
	return toBool(v) == r
}

func boolTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_bool":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bool.to_bool", "0", len(args))
		}
		return v, nil

	case "to_int":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bool.to_int", "0", len(args))
		}
		b, _ := boolTypeAsInt(v)
		return IntValue(b), nil

	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bool.to_string", "0", len(args))
		}
		s, _ := boolTypeAsString(v)
		return vm.Allocator().NewStringValue(s), nil

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, "bool")
	}
}

package core

import (
	"fmt"

	"github.com/jokruger/kavun/errs"
)

// BoolValue creates new boxed bool value.
func BoolValue(b bool) Value {
	v := Value{Type: VT_BOOL, Const: true}
	if b {
		v.Data = 1
	}
	return v
}

/* Bool type methods */

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
	if v.Data == 0 {
		return "false"
	}
	return "true"
}

func boolTypeInterface(v Value) any {
	return v.Data != 0
}

func boolTypeIsTrue(v Value) bool {
	return v.Data != 0
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

func boolTypeAsBool(v Value) (bool, bool) {
	return v.Data != 0, true
}

func boolTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsBool()
	if !ok {
		return false
	}
	return (v.Data != 0) == r
}

func boolTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := boolTypeAsInt(v)
		return IntValue(b), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := boolTypeAsString(v)
		return vm.Allocator().NewStringValue(s)

	default:
		return Undefined, errs.NewInvalidMethodError(name, "bool")
	}
}

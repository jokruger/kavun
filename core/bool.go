package core

import (
	"fmt"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
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

func boolTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	var body string
	switch sp.Verb {
	case 'v':
		return boolTypeString(v), nil

	case 'T':
		body = boolTypeName(v)

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

func boolTypeAsByte(v Value) (byte, bool) {
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
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := boolTypeAsInt(v)
		return IntValue(b), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := boolTypeAsByte(v)
		return ByteValue(b), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := boolTypeAsString(v)
		return vm.Allocator().NewStringValue(s), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := boolTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "bool")
	}
}

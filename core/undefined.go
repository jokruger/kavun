package core

import (
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// UndefinedValue creates new boxed undefined value.
func UndefinedValue() Value {
	return Value{}
}

/* Undefined type methods */

func undefinedTypeName(v Value) string {
	return "undefined"
}

func undefinedTypeString(v Value) string {
	return "undefined"
}

func undefinedTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return "undefined", nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics("undefined", sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics("undefined", sp, fspec.AlignLeft), nil
}

func undefinedTypeEncodeJSON(v Value) ([]byte, error) {
	return []byte("null"), nil
}

func undefinedTypeEncodeBinary(v Value) ([]byte, error) {
	return []byte{}, nil
}

func undefinedTypeDecodeBinary(v *Value, data []byte) error {
	return nil
}

func undefinedTypeInterface(v Value) any {
	return nil
}

func undefinedTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
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
		s, err := v.Format(sp)
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewStringValue(s), nil

	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "undefined")
	}
}

func undefinedTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	return UndefinedValue(), nil
}

func undefinedTypeAsBool(v Value) (bool, bool) {
	return false, true
}

package core

import (
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const undefinedTypeName = "undefined"

var Undefined = Value{}

var TypeUndefined = ValueType{
	Name:         ConstHook(undefinedTypeName),
	Interface:    func(Value) any { return nil },
	String:       func(Value) string { return undefinedTypeName },
	Format:       undefinedTypeFormat,
	EncodeJSON:   func(Value) ([]byte, error) { return []byte("null"), nil },
	EncodeBinary: func(Value) ([]byte, error) { return []byte{}, nil },
	DecodeBinary: func(v *Value, _ []byte) error { *v = Undefined; return nil },
	IsTrue:       ConstHook(false), // undefined is always false
	IsIterable:   ConstHook(true),
	Equal:        func(v Value, r Value) bool { return v.Type == r.Type && v.Data == r.Data && v.Ptr == r.Ptr },
	MethodCall:   undefinedTypeMethodCall,
	Access:       func(Value, *Arena, Value, bc.Opcode) (Value, error) { return Undefined, nil },
	AsBool:       func(Value) (bool, bool) { return false, true },
}

func undefinedTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return undefinedTypeName, nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
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

	case "repeat":
		return repeatScalarToArray(v, vm, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, undefinedTypeName)
	}
}

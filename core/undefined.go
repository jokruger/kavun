package core

import (
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/opcode"
)

const undefinedTypeName = "undefined"

var Undefined = Value{}

var TypeUndefined = ValueTypeDescr{
	Name:         ConstHook(undefinedTypeName),
	Interface:    func(*Arena, Value) any { return nil },
	String:       func(*Arena, Value) string { return undefinedTypeName },
	Format:       undefinedTypeFormat,
	EncodeJSON:   func(*Arena, Value) ([]byte, error) { return []byte("null"), nil },
	EncodeBinary: func(*Arena, Value) ([]byte, error) { return []byte{}, nil },
	DecodeBinary: func(_ *Arena, v *Value, _ []byte) error { *v = Undefined; return nil },
	IsTrue:       ConstHook(false), // undefined is always false
	IsIterable:   ConstHook(true),
	Equal:        func(_ *Arena, v Value, r Value) bool { return v.Type == r.Type },
	MethodCall:   undefinedTypeMethodCall,
	Access:       func(*Arena, Value, Value, opcode.Opcode) (Value, error) { return Undefined, nil },
	AsBool:       func(*Arena, Value) (bool, bool) { return false, true },
}

func undefinedTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return undefinedTypeName, nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}
	return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
}

func undefinedTypeMethodCall(a *Arena, _ VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := v.Format(a, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "repeat":
		return repeatScalarToArray(a, v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, undefinedTypeName)
	}
}

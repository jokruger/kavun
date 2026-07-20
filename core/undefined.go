package core

import (
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const undefinedTypeName = "undefined"

var TypeUndefined = ValueTypeDescr{
	Name:         ConstHook(undefinedTypeName),                                           // PURE by contract
	Interface:    func(Value) any { return nil },                                         // PURE by contract
	String:       func(Value) string { return undefinedTypeName },                        // PURE by contract
	Format:       undefinedTypeFormat,                                                    // PURE by contract
	EncodeJSON:   func(Value) ([]byte, error) { return []byte("null"), nil },             // PURE by contract
	EncodeBinary: func(Value) ([]byte, error) { return []byte{}, nil },                   // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { *v = Undefined; return nil },          // IMPURE by contract (mutates target)
	IsTrue:       ConstHook(false),                                                       // PURE by contract
	IsIterable:   ConstHook(true),                                                        // PURE by contract
	Equal:        func(v Value, r Value) bool { return v.Type == r.Type },                // PURE by contract
	MethodCall:   undefinedTypeMethodCall,                                                // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       func(Value, Value, bc.Opcode) (Value, error) { return Undefined, nil }, // PURE by contract
	AsBool:       func(Value) (bool, bool) { return false, true },                        // PURE by contract
	IsMethodPure: func(string) bool { return true },                                      // All methods are expected to be pure.
}

// PURE by contract
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

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func undefinedTypeMethodCall(_ VM, v Value, name string, args []Value) (Value, error) {
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
		return NewStringValue(s), nil

	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "repeat":
		return repeatScalarToArray(v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, undefinedTypeName)
	}
}

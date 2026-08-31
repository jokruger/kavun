package core

import (
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const undefinedTypeName = "undefined"

var TypeUndefined = ValueTypeDescr{
	Name:         ConstHook(undefinedTypeName),                                  // PURE by contract
	Interface:    func(Value) any { return nil },                                // PURE by contract
	String:       func(Value) string { return undefinedTypeName },               // PURE by contract
	Format:       undefinedTypeFormat,                                           // PURE by contract
	EncodeJSON:   func(Value) ([]byte, error) { return []byte("null"), nil },    // PURE by contract
	EncodeBinary: func(Value) ([]byte, error) { return []byte{}, nil },          // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { *v = Undefined; return nil }, // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](false, nil),                           // PURE by contract
	// Undefined propagates on the DATA plane (selectors, indexing, slicing, operators — a chain like a.b.c
	// misses at any level and answers one undefined) and raises on the ACTION plane (call, iteration,
	// membership, members). Getting an iterator is an action, so IsIterable answers false and for-in raises.
	IsIterable:   ConstHook(false),                                                          // PURE by contract
	Equal:        undefinedTypeEqual,                                                        // PURE by contract
	BinaryOp:     undefinedTypeBinaryOp,                                                     // PURE by contract
	UnaryOp:      undefinedTypeUnaryOp,                                                      // PURE by contract
	MethodCall:   undefinedTypeMethodCall,                                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       func(Value, Value, bc.Opcode) (Value, error) { return Undefined, nil },    // PURE by contract
	Slice:        func(Value, Value, Value) (Value, error) { return Undefined, nil },        // PURE by contract
	SliceStep:    func(Value, Value, Value, Value) (Value, error) { return Undefined, nil }, // PURE by contract
	AsBool:       func(Value) (bool, bool) { return false, true },                           // PURE by contract
	IsMethodPure: func(string) bool { return true },                                         // All methods are expected to be pure.
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

func undefinedTypeEqual(v Value, other Value, _ bool) bool {
	// undefined is only equal to undefined
	return v.Type == other.Type
}

func undefinedTypeBinaryOp(Value, Value, token.Token, bool) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}

func undefinedTypeUnaryOp(Value, token.Token) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func undefinedTypeMethodCall(_ VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "bool", "byte", "rune", "int", "float", "decimal", "time",
		"string", "runes", "bytes", "array", "dict", "record":
		// the maybe-missing rescue: the conversion members exist on undefined with a MANDATORY default,
		// so a propagated chain can materialize with a typed fallback — d["missing"].int(0) → 0.
		// Absence converts to nothing, so with no default the conversion raises like every T(undefined).
		switch len(args) {
		case 1:
			return args[0], nil
		case 0:
			return Undefined, errs.NewConversionError(undefinedTypeName, name, "value is missing")
		default:
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}

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

	default:
		return Undefined, errs.NewInvalidMethodError(name, undefinedTypeName)
	}
}

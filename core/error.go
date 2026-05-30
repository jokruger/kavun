package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const errorTypeName = "error"

type Error struct {
	Payload Value
	Kind    string
	Fatal   bool
}

// KindUser is the kind tag automatically assigned to errors constructed from script via the error() builtin.
const KindUser = "user"

// ErrorValue creates new boxed error value.
func ErrorValue(v *Error) Value {
	return Value{
		Type:      VT_ERROR,
		Immutable: true,
		Ptr:       unsafe.Pointer(v),
	}
}

// NewErrorValue creates a heap-allocated user-kind recoverable error value. Script-level errors are recoverable by
// default — the zero value of the Fatal flag (false) keeps user errors visible to deferred recover().
func NewErrorValue(payload Value) Value {
	return ErrorValue(&Error{
		Payload: payload,
		Kind:    KindUser,
	})
}

// NewFatalErrorValue creates a heap-allocated user-kind fatal error value. A fatal error, when raised, bypasses
// recover() and stops the VM, propagating to the host caller.
func NewFatalErrorValue(payload Value) Value {
	return ErrorValue(&Error{
		Payload: payload,
		Kind:    KindUser,
		Fatal:   true,
	})
}

// NewRuntimeErrorValue creates a heap-allocated error value with explicit kind, fatality and a string message wrapped
// as the payload. Used internally by the runtime when boxing an *errs.Error so that script-level recover() can inspect
// it (and so the round-trip back to a Go *errs.Error preserves severity).
func NewRuntimeErrorValue(kind string, fatal bool, message string) Value {
	return ErrorValue(&Error{
		Payload: NewStringValue(message),
		Kind:    kind,
		Fatal:   fatal,
	})
}

var TypeError = ValueType{
	Name:         ConstHook(errorTypeName),
	String:       errorTypeString,
	Format:       errorTypeFormat,
	Interface:    func(a *Arena, v Value) any { return errors.New(v.String(a)) },
	EncodeJSON:   errorTypeEncodeJSON,
	EncodeBinary: errorTypeEncodeBinary,
	DecodeBinary: errorTypeDecodeBinary,
	IsTrue:       ConstHook(false), // error is always false
	Equal:        errorTypeEqual,
	Clone:        errorTypeClone,
	MethodCall:   errorTypeMethodCall,
	AsString:     errorTypeAsString,
	AsBool:       Const2Hook(false, true),
}

func errorTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	s, _ := o.Payload.AsString(a)
	return fmt.Appendf(nil, `{"error":%q}`, s), nil
}

func errorTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Kind); err != nil {
		return nil, fmt.Errorf("error (kind): %w", err)
	}
	if err := enc.Encode(o.Fatal); err != nil {
		return nil, fmt.Errorf("error (fatal): %w", err)
	}
	if err := enc.Encode(o.Payload); err != nil {
		return nil, fmt.Errorf("error (payload): %w", err)
	}
	return buf.Bytes(), nil
}

func errorTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	o := &Error{}
	if err := dec.Decode(&o.Kind); err != nil {
		return fmt.Errorf("error (kind): %w", err)
	}
	if err := dec.Decode(&o.Fatal); err != nil {
		return fmt.Errorf("error (fatal): %w", err)
	}
	if err := dec.Decode(&o.Payload); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func errorTypeString(a *Arena, v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == VT_UNDEFINED {
		return "error()"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String(a))
}

func errorTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}
	switch sp.Verb {
	case 0:
		o := (*Error)(v.Ptr)
		s, _ := o.Payload.AsString(a)
		return fspec.ApplyGenerics(s, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(a, v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName, sp, fspec.AlignLeft), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}
}

func errorTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_ERROR {
		return false
	}
	o := (*Error)(v.Ptr)
	x := (*Error)(r.Ptr)
	return o.Kind == x.Kind && o.Payload.Equal(a, x.Payload)
}

func errorTypeClone(a *Arena, v Value) (Value, error) {
	o := (*Error)(v.Ptr)
	pl, err := o.Payload.Clone(a)
	if err != nil {
		return Undefined, err
	}
	return ErrorValue(&Error{
		Payload: pl,
		Kind:    o.Kind,
		Fatal:   o.Fatal,
	}), nil
}

func errorTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return errorTypeClone(a, v)

	case "value":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return o.Payload, nil

	case "kind":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return a.NewStringValue(o.Kind), nil

	case "is_runtime":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return BoolValue(o.Kind != KindUser), nil

	case "is_fatal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return BoolValue(o.Fatal), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		s, _ := o.Payload.AsString(a)
		return a.NewStringValue(s), nil

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
		s, err := errorTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func errorTypeAsString(a *Arena, v Value) (string, bool) {
	o := (*Error)(v.Ptr)
	if s, ok := o.Payload.AsString(a); ok {
		return s, true
	}
	return o.Payload.String(a), true
}

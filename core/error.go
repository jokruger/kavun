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

/* Error type methods */

func errorTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	s, _ := o.Payload.AsString()
	return fmt.Appendf(nil, `{"error":%q}`, s), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
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

func errorTypeDecodeBinary(v *Value, data []byte) error {
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

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == VT_UNDEFINED {
		return "error()"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String())
}

func errorTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	switch sp.Verb {
	case 0:
		o := (*Error)(v.Ptr)
		s, _ := o.Payload.AsString()
		return fspec.ApplyGenerics(s, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName, sp, fspec.AlignLeft), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
}

func errorTypeInterface(v Value) any {
	return errors.New(v.String())
}

func errorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_ERROR {
		return false
	}
	o := (*Error)(v.Ptr)
	x := (*Error)(r.Ptr)
	return o.Kind == x.Kind && o.Payload.Equal(x.Payload)
}

func errorTypeCopy(v Value, a *Arena) (Value, error) {
	o := (*Error)(v.Ptr)
	pl, err := o.Payload.Copy(a)
	if err != nil {
		return Undefined, err
	}
	return ErrorValue(&Error{
		Payload: pl,
		Kind:    o.Kind,
		Fatal:   o.Fatal,
	}), nil
}

func errorTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return errorTypeCopy(v, vm.Allocator())

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
		return vm.Allocator().NewStringValue(o.Kind), nil

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
		s, _ := o.Payload.AsString()
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
		s, err := errorTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func errorTypeAsString(v Value) (string, bool) {
	o := (*Error)(v.Ptr)
	if s, ok := o.Payload.AsString(); ok {
		return s, true
	}
	return o.Payload.String(), true
}

// error is always false
func errorTypeAsBool(v Value) (bool, bool) {
	return false, true
}

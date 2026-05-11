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

// ErrorOrigin identifies who created an error value.
//   - OriginUser: the value was constructed via the script-level error() builtin (or returned from user code).
//   - OriginVM: the value was constructed by the VM/runtime/builtins as a result of a raised logical error caught by
//     deferred recover().
type ErrorOrigin uint8

const (
	OriginUser ErrorOrigin = 0
	OriginVM   ErrorOrigin = 1
)

func (o ErrorOrigin) String() string {
	switch o {
	case OriginVM:
		return "vm"
	default:
		return "user"
	}
}

type Error struct {
	Payload Value
	Origin  ErrorOrigin // who created the error value, defaults to OriginUser
	Kind    string      // (optional) stable string tag for VM-origin errors, empty for user-origin errors

	cause error // original Go-side error sentinel for VM-origin errors
}

// Cause returns the underlying Go error preserved on this Kavun error value (set on VM-origin errors so that
// errors.Is keeps working when the value re-enters the host as a Go error). Returns nil for user-origin errors.
func (e *Error) Cause() error {
	return e.cause
}

// SetCause records the underlying Go error for this Kavun error value.
// Used internally by the runtime; not exposed to script code.
func (e *Error) SetCause(err error) {
	e.cause = err
}

func (o *Error) Set(payload Value) {
	o.Payload = payload
	o.Origin = 0
	o.Kind = ""
	o.cause = nil
}

// ErrorValue creates new boxed error value.
func ErrorValue(v *Error) Value {
	return Value{
		Type:  VT_ERROR,
		Const: true,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewErrorValue creates new (heap-allocated) error value.
func NewErrorValue(payload Value) Value {
	t := &Error{}
	t.Set(payload)
	return ErrorValue(t)
}

// NewVMErrorValue creates a heap-allocated error value with VM origin and a stable kind tag. Used internally by the
// runtime when wrapping a Go error raised by a builtin/op so that script-level recover() can inspect it.
func NewVMErrorValue(payload Value, kind string) Value {
	return ErrorValue(&Error{
		Payload: payload,
		Origin:  OriginVM,
		Kind:    kind,
	})
}

/* Error type methods */

func errorTypeName(v Value) string {
	return "error"
}

func errorTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	s, _ := o.Payload.AsString()
	return []byte(fmt.Sprintf(`{"error":%q}`, s)), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Payload); err != nil {
		return nil, fmt.Errorf("error (payload): %w", err)
	}
	if err := enc.Encode(uint8(o.Origin)); err != nil {
		return nil, fmt.Errorf("error (origin): %w", err)
	}
	if err := enc.Encode(o.Kind); err != nil {
		return nil, fmt.Errorf("error (kind): %w", err)
	}
	return buf.Bytes(), nil
}

func errorTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var val Value
	if err := dec.Decode(&val); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	o := &Error{Payload: val}
	// origin/kind fields are best-effort: tolerate older binary blobs.
	var originByte uint8
	if err := dec.Decode(&originByte); err == nil {
		o.Origin = ErrorOrigin(originByte)
		var kind string
		if err := dec.Decode(&kind); err == nil {
			o.Kind = kind
		}
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	return fmt.Sprintf("error(%s)", o.Payload.String())
}

func errorTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	switch sp.Verb {
	case 0:
		o := (*Error)(v.Ptr)
		m, _ := o.Payload.AsString()
		return fspec.ApplyGenerics(m, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName(v), sp, fspec.AlignLeft), nil

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
	return o.Origin == x.Origin && o.Kind == x.Kind && o.Payload.Equal(x.Payload)
}

func errorTypeCopy(v Value, a *Arena) (Value, error) {
	o := (*Error)(v.Ptr)
	t, err := o.Payload.Copy(a)
	if err != nil {
		return Undefined, err
	}
	return a.NewErrorValueWithMeta(t, o.Origin, o.Kind), nil
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

	case "origin":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Origin.String()), nil

	case "kind":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Kind), nil

	case "is_user":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return BoolValue(o.Origin == OriginUser), nil

	case "is_vm":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Error)(v.Ptr)
		return BoolValue(o.Origin == OriginVM), nil

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
	s, ok := o.Payload.AsString()
	if ok {
		return s, true
	}
	return "runtime error", true
}

// error is always false
func errorTypeAsBool(v Value) (bool, bool) {
	return false, true
}

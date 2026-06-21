package core

import (
	"errors"
	"fmt"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
)

const errorTypeName = "error"

// KindUser is the kind tag automatically assigned to errors constructed from script via the error() builtin.
const KindUser = "user"

type Error struct {
	Payload Value
	Kind    string
	Fatal   bool
}

func (e *Error) Set(payload Value, kind string, fatal bool) {
	e.Payload = payload
	e.Kind = kind
	e.Fatal = fatal
}

func (a *Arena) MustNewErrorValue(payload Value, kind string, fatal bool) Value {
	v, err := a.NewErrorValue(payload, kind, fatal)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewErrorValue(payload Value, kind string, fatal bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Error); ok {
		a.PinAny(payload) // mark payload as unmanaged because it's now also owned by the error value
		(*Error)(p).Set(payload, kind, fatal)
		return Value{Type: value.Error, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(errorTypeName)
}

func (a *Arena) NewRuntimeErrorValue(kind string, fatal bool, message string) (Value, error) {
	payload, err := a.NewStringValue(message)
	if err != nil {
		return Undefined, err
	}
	return a.NewErrorValue(payload, kind, fatal)
}

var TypeError = ValueTypeDescr{
	Name:         ConstHook(errorTypeName),
	String:       errorTypeString,
	Format:       errorTypeFormat,
	Interface:    func(v Value) any { return errors.New(v.String(a)) },
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

func errorTypeEncodeJSON(v Value) ([]byte, error) {
	o := a.ResolveErrorValue(v)
	s, _ := o.Payload.AsString()
	return fmt.Appendf(nil, `{"error":%q}`, s), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
	o := a.ResolveErrorValue(v)
	pb, err := o.Payload.EncodeBinary(a)
	if err != nil {
		return nil, fmt.Errorf("error (payload): %w", err)
	}

	b := binary.AppendBytes(nil, []byte(o.Kind))
	if o.Fatal {
		b = append(b, byte(1))
	} else {
		b = append(b, byte(0))
	}
	b = binary.AppendBytes(b, pb)
	return b, nil
}

func errorTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	kb, err := binary.ReadBytes(data, &offset, "error (kind)")
	if err != nil {
		return err
	}
	if len(data)-offset < 1 {
		return fmt.Errorf("error (fatal): expected 1 byte, got %d", len(data)-offset)
	}
	fatal := data[offset] != 0
	offset++

	pb, err := binary.ReadBytes(data, &offset, "error (payload)")
	if err != nil {
		return err
	}
	var payload Value
	if err := payload.DecodeBinary(a, pb); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	if offset != len(data) {
		return fmt.Errorf("error: trailing %d bytes", len(data)-offset)
	}

	o, err := a.NewErrorValue(payload, string(kb), fatal)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o
	return nil
}

func errorTypeString(v Value) string {
	o := a.ResolveErrorValue(v)
	if o.Payload.Type == value.Undefined {
		return "error()"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String(a))
}

func errorTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	switch sp.Verb {
	case 0:
		o := a.ResolveErrorValue(v)
		s, _ := o.Payload.AsString()
		return fspec.ApplyGenerics(s, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(a, v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName, sp, fspec.AlignLeft), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
}

func errorTypeEqual(v Value, r Value) bool {
	if r.Type != value.Error {
		return false
	}
	o := a.ResolveErrorValue(v)
	x := a.ResolveErrorValue(r)
	return o.Kind == x.Kind && o.Payload.Equal(a, x.Payload)
}

func errorTypeClone(v Value) (Value, error) {
	o := a.ResolveErrorValue(v)
	pl, err := o.Payload.Clone(a)
	if err != nil {
		return Undefined, err
	}
	return a.NewErrorValue(pl, o.Kind, o.Fatal)
}

func errorTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
		o := a.ResolveErrorValue(v)
		return o.Payload, nil

	case "kind":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveErrorValue(v)
		return a.NewStringValue(o.Kind)

	case "is_runtime":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveErrorValue(v)
		return BoolValue(o.Kind != KindUser), nil

	case "is_fatal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveErrorValue(v)
		return BoolValue(o.Fatal), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveErrorValue(v)
		s, _ := o.Payload.AsString()
		return a.NewStringValue(s)

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
		s, err := errorTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func errorTypeAsString(v Value) (string, bool) {
	o := a.ResolveErrorValue(v)
	if s, ok := o.Payload.AsString(); ok {
		return s, true
	}
	return o.Payload.String(a), true
}

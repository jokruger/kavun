package core

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
)

const errorTypeName = "error"
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

func NewErrorValue(payload Value, kind string, fatal bool) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: payload, Kind: kind, Fatal: fatal}),
	}
}

func NewRuntimeErrorValue(kind string, fatal bool, message string) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: NewStringValue(message), Kind: kind, Fatal: fatal}),
	}
}

var TypeError = ValueTypeDescr{
	Name:         ConstHook(errorTypeName),
	String:       errorTypeString,
	Format:       errorTypeFormat,
	Interface:    func(v Value) any { return errors.New(v.String()) },
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
	o := (*Error)(v.Ptr)
	s, _ := o.Payload.AsString()
	return fmt.Appendf(nil, `{"error":%q}`, s), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	pb, err := o.Payload.EncodeBinary()
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
	if err := payload.DecodeBinary(pb); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	if offset != len(data) {
		return fmt.Errorf("error: trailing %d bytes", len(data)-offset)
	}

	*v = NewErrorValue(payload, string(kb), fatal)
	return nil
}

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == value.Undefined {
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

func errorTypeEqual(v Value, r Value) bool {
	if r.Type != value.Error {
		return false
	}
	o := (*Error)(v.Ptr)
	x := (*Error)(r.Ptr)
	return o.Kind == x.Kind && o.Payload.Equal(x.Payload)
}

func errorTypeClone(v Value) (Value, error) {
	o := (*Error)(v.Ptr)
	pl, err := o.Payload.Clone()
	if err != nil {
		return Undefined, err
	}
	return NewErrorValue(pl, o.Kind, o.Fatal), nil
}

func errorTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return errorTypeClone(v)

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
		return NewStringValue(o.Kind), nil

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
		return NewStringValue(s), nil

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
		return NewStringValue(s), nil

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

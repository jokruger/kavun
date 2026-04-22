package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"unsafe"

	"github.com/jokruger/gs/errs"
)

type Error struct {
	Payload Value
}

func (o *Error) Set(payload Value) {
	o.Payload = payload
}

// ErrorValue creates new boxed error value.
func ErrorValue(v *Error) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_ERROR,
	}
}

// NewErrorValue creates new (heap-allocated) error value.
func NewErrorValue(payload Value) Value {
	t := &Error{}
	t.Set(payload)
	return ErrorValue(t)
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
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == VT_UNDEFINED {
		return "error(undefined)"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String())
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
	return o.Payload.Equal(x.Payload)
}

func errorTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*Error)(v.Ptr)
	t, err := o.Payload.Copy(a)
	if err != nil {
		return Undefined, err
	}
	return a.NewErrorValue(t)
}

func errorTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "value":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
		}
		o := (*Error)(v.Ptr)
		return o.Payload, nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
		}
		o := (*Error)(v.Ptr)
		s, _ := o.Payload.AsString()
		return vm.Allocator().NewStringValue(s)

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

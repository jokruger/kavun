package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"unsafe"

	"github.com/jokruger/gs/errs"
)

type Record struct {
	Elements  map[string]Value
	Immutable bool
}

func (o *Record) Set(elements map[string]Value, immutable bool) {
	o.Elements = elements
	o.Immutable = immutable

	if o.Elements == nil {
		o.Elements = make(map[string]Value)
	}
}

// RecordValue creates new boxed record value.
func RecordValue(v *Record) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_RECORD,
	}
}

// NewRecordValue creates new (heap-allocated) record value.
func NewRecordValue(vals map[string]Value, immutable bool) Value {
	t := &Record{}
	t.Set(vals, immutable)
	return RecordValue(t)
}

// ToRecord converts boxed record value to *Record. It is a caller's responsibility to ensure the type is correct.
func ToRecord(v Value) *Record {
	return (*Record)(v.Ptr)
}

/* Record type methods */

func recordTypeName(v Value) string {
	o := (*Record)(v.Ptr)
	if o.Immutable {
		return "immutable-record"
	}
	return "record"
}

func recordTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
		idx++
	}
	b = append(b, '}')
	return b, nil
}

func recordTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Immutable); err != nil {
		return nil, fmt.Errorf("record (immutable flag): %w", err)
	}
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("record (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func recordTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return fmt.Errorf("record (immutable flag): %w", err)
	}
	var value map[string]Value
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("record (elements): %w", err)
	}
	if value == nil {
		value = make(map[string]Value)
	}
	o := &Record{
		Elements:  value,
		Immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func recordTypeString(v Value) string {
	o := (*Record)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeInterface(v Value) any {
	o := (*Record)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func recordTypeEqual(v Value, r Value) bool {
	switch r.Type {
	case VT_RECORD:
		o := (*Record)(v.Ptr)
		x := (*Record)(r.Ptr)
		if len(o.Elements) != len(x.Elements) {
			return false
		}
		for k, v := range o.Elements {
			if !v.Equal(x.Elements[k]) {
				return false
			}
		}
		return true

	case VT_MAP:
		o := (*Record)(v.Ptr)
		x := (*Map)(r.Ptr)
		if len(o.Elements) != len(x.Elements) {
			return false
		}
		for k, v := range o.Elements {
			if !v.Equal(x.Elements[k]) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func recordTypeCopy(v Value, a Allocator) (Value, error) {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewRecordValue(c, false)
}

func recordTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Record)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("record access", "string", index.TypeName())
	}
	o := (*Record)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

func recordTypeAssign(v Value, index Value, r Value) error {
	o := (*Record)(v.Ptr)
	if o.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("record assignment", "string", index.TypeName())
	}
	o.Elements[k] = r

	return nil
}

func recordTypeIsIterable(v Value) bool {
	return true
}

func recordTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*Record)(v.Ptr)
	return a.NewMapIteratorValue(o.Elements)
}

func recordTypeIsImmutable(v Value) bool {
	o := (*Record)(v.Ptr)
	return o.Immutable
}

func recordTypeIsTrue(v Value) bool {
	o := (*Record)(v.Ptr)
	return len(o.Elements) > 0
}

func recordTypeAsString(v Value) (string, bool) {
	return recordTypeString(v), true
}

func recordTypeAsBool(v Value) (bool, bool) {
	return recordTypeIsTrue(v), true
}

func recordTypeAsMap(v Value, a Allocator) (map[string]Value, bool) {
	o := (*Record)(v.Ptr)
	return o.Elements, true
}

func recordTypeContains(v Value, e Value) bool {
	o := (*Record)(v.Ptr)
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = o.Elements[s]
	return ok
}

func recordTypeLen(v Value) int64 {
	o := (*Record)(v.Ptr)
	return int64(len(o.Elements))
}

func recordTypeDelete(v Value, key Value) (Value, error) {
	o := (*Record)(v.Ptr)
	if o.Immutable {
		return Undefined, errs.NewInvalidDeleteError(v.TypeName())
	}
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("record delete", "string", key.TypeName())
	}
	delete(o.Elements, s)
	return v, nil
}

func recordTypeImmutable(v Value, a Allocator) (Value, error) {
	o := (*Record)(v.Ptr)
	if o.Immutable {
		return v, nil
	}
	return a.NewRecordValue(o.Elements, true)
}

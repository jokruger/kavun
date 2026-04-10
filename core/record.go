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
	value     map[string]Value
	immutable bool
}

func (o *Record) Set(value map[string]Value, immutable bool) {
	o.value = value
	o.immutable = immutable

	if o.value == nil {
		o.value = make(map[string]Value)
	}
}

func (o *Record) Value() map[string]Value {
	return o.value
}

func (o *Record) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Record) Len() int {
	return len(o.value)
}

func (o *Record) Delete(key string) {
	delete(o.value, key)
}

func (o *Record) Has(key string) bool {
	_, ok := o.value[key]
	return ok
}

func (o *Record) Get(key string) (Value, bool) {
	v, ok := o.value[key]
	return v, ok
}

func (o *Record) Keys() []string {
	keys := make([]string, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, k)
	}
	return keys
}

func (o *Record) SetKey(key string, val Value) {
	o.value[key] = val
}

func RecordValue(v *Record) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_RECORD,
	}
}

func NewRecordValue(vals map[string]Value, immutable bool) Value {
	t := &Record{}
	t.Set(vals, immutable)
	return RecordValue(t)
}

func recordTypeName(v Value) string {
	o := (*Record)(v.Ptr)
	if o.immutable {
		return "immutable-record"
	}
	return "record"
}

func recordTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := o.Len() - 1
	idx := 0
	for key, value := range o.Value() {
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
	if err := enc.Encode(o.immutable); err != nil {
		return nil, fmt.Errorf("record (immutable flag): %w", err)
	}
	if err := enc.Encode(o.value); err != nil {
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
		value:     value,
		immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func recordTypeString(v Value) string {
	o := (*Record)(v.Ptr)
	pairs := make([]string, 0, len(o.value))
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeInterface(v Value) any {
	o := (*Record)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.value {
		res[key] = v.Interface()
	}
	return res
}

func recordTypeEqual(v Value, r Value) bool {
	switch {
	case r.IsRecord():
		o := (*Record)(v.Ptr)
		x := (*Record)(r.Ptr)
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equal(x.value[k]) {
				return false
			}
		}
		return true

	case r.IsMap():
		o := (*Record)(v.Ptr)
		x := (*Map)(r.Ptr)
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equal(x.value[k]) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func recordTypeCopy(v Value, a Allocator) Value {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(a)
	}
	return a.NewRecordValue(c, false)
}

func recordTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Record)(v.Ptr)
	e, ok := o.value[name]
	if !ok {
		return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return UndefinedValue(), fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidIndexTypeError("record access", "string", index.TypeName())
	}
	o := (*Record)(v.Ptr)
	r, ok := o.value[k]
	if !ok {
		return UndefinedValue(), nil
	}
	return r, nil
}

func recordTypeAssign(v Value, index Value, r Value) error {
	o := (*Record)(v.Ptr)
	if o.immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("record assignment", "string", index.TypeName())
	}
	o.value[k] = r

	return nil
}

func recordTypeIsIterable(v Value) bool {
	return true
}

func recordTypeIterator(v Value, a Allocator) Value {
	o := (*Record)(v.Ptr)
	return a.NewMapIteratorValue(o.value)
}

func recordTypeIsImmutable(v Value) bool {
	o := (*Record)(v.Ptr)
	return o.immutable
}

func recordTypeIsTrue(v Value) bool {
	o := (*Record)(v.Ptr)
	return len(o.value) > 0
}

func recordTypeAsString(v Value) (string, bool) {
	return recordTypeString(v), true
}

func recordTypeAsBool(v Value) (bool, bool) {
	return recordTypeIsTrue(v), true
}

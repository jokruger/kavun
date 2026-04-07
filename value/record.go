package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Record struct {
	Object
	value     map[string]core.Value
	immutable bool
}

func (o *Record) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals map[string]core.Value
	if err := dec.Decode(&vals); err != nil {
		return err
	}

	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return err
	}

	o.Set(vals, immutable)
	return nil
}

func (o *Record) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(o.value); err != nil {
		return nil, err
	}

	if err := enc.Encode(o.immutable); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o *Record) Set(val map[string]core.Value, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = make(map[string]core.Value)
	}
	o.immutable = immutable
}

func (o *Record) Value() map[string]core.Value {
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

func (o *Record) Get(key string) (core.Value, bool) {
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

func (o *Record) SetKey(key string, val core.Value) {
	o.value[key] = val
}

func (o *Record) TypeName() string {
	if o.immutable {
		return "immutable-record"
	}
	return "record"
}

func (o *Record) String() string {
	pairs := make([]string, 0, len(o.value))
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func (o *Record) Interface() any {
	res := make(map[string]any)
	for key, v := range o.value {
		res[key] = v.Interface()
	}
	return res
}

func (o *Record) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Record) Equals(x core.Value) bool {
	if !x.IsObject() {
		return false
	}

	switch x := x.Object().(type) {
	case *Record:
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equals(x.value[k]) {
				return false
			}
		}
		return true
	case *Map:
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equals(x.value[k]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (o *Record) Copy(alloc core.Allocator) core.Value {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Value, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(alloc)
	}
	return alloc.NewRecordValue(c, false)
}

func (o *Record) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	v, ok := o.value[name]
	if !ok {
		return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
	}
	if !v.IsCallable() {
		return core.UndefinedValue(), fmt.Errorf("%s.%s is not callable, got %s", o.TypeName(), name, v.TypeName())
	}

	return v.Call(vm, args)
}

func (o *Record) Access(vm core.VM, index core.Value, mode core.Opcode) (core.Value, error) {
	k, ok := index.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidIndexTypeError("record access", "string", index.TypeName())
	}
	r, ok := o.value[k]
	if !ok {
		return core.UndefinedValue(), nil
	}
	return r, nil
}

func (o *Record) Assign(index, value core.Value) error {
	if o.immutable {
		return core.NewNotAssignableError(o.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return core.NewInvalidIndexTypeError("record assignment", "string", index.TypeName())
	}
	o.value[k] = value

	return nil
}

func (o *Record) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewMapIterator(o.value)
}

func (o *Record) IsImmutable() bool {
	return o.immutable
}

func (o *Record) IsRecord() bool {
	return true
}

func (o *Record) IsTrue() bool {
	return len(o.value) > 0
}

func (o *Record) IsFalse() bool {
	return len(o.value) == 0
}

func (o *Record) IsIterable() bool {
	return true
}

func (o *Record) AsString() (string, bool) {
	return o.String(), true
}

func (o *Record) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

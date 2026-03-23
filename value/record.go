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
	value     map[string]core.Object
	immutable bool
}

func (o *Record) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals map[string]core.Object
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

func (o *Record) Set(val map[string]core.Object, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = make(map[string]core.Object)
	}

	o.immutable = immutable
}

func (o *Record) Value() map[string]core.Object {
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

func (o *Record) Get(key string) (core.Object, bool) {
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

func (o *Record) SetKey(key string, value core.Object) {
	o.value[key] = value
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

func (o *Record) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Record) Equals(x core.Object) bool {
	switch x := x.(type) {
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

func (o *Record) Copy(alloc core.Allocator) core.Object {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Object, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(alloc)
	}
	return alloc.NewRecord(c, false) // copy always returns a mutable record
}

func (o *Record) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	alloc := vm.Allocator()
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("record access", "string", index)
	}
	r, ok := o.value[k]
	if !ok {
		return alloc.NewUndefined(), nil
	}
	return r, nil
}

func (o *Record) Assign(index, value core.Object) error {
	if o.immutable {
		return core.NewNotAssignableError(o)
	}

	k, ok := index.AsString()
	if !ok {
		return core.NewInvalidIndexTypeError("record assignment", "string", index)
	}
	o.value[k] = value

	return nil
}

func (o *Record) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewMapIterator(o.value)
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

func (o *Record) IsImmutable() bool {
	return o.immutable
}

func (o *Record) AsString() (string, bool) {
	return o.String(), true
}

func (o *Record) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

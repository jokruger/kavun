package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Map struct {
	value     map[string]core.Object
	immutable bool
}

func NewMap(val map[string]core.Object, immutable bool) *Map {
	o := &Map{}
	o.Set(val, immutable)
	return o
}

func (o *Map) GobDecode(b []byte) error {
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

func (o *Map) GobEncode() ([]byte, error) {
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

func (o *Map) Set(val map[string]core.Object, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = make(map[string]core.Object)
	}

	o.immutable = immutable
}

func (o *Map) Value() map[string]core.Object {
	return o.value
}

func (o *Map) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Map) Len() int {
	return len(o.value)
}

func (o *Map) Delete(key string) {
	delete(o.value, key)
}

func (o *Map) Has(key string) bool {
	_, ok := o.value[key]
	return ok
}

func (o *Map) Get(key string) (core.Object, bool) {
	v, ok := o.value[key]
	return v, ok
}

func (o *Map) Keys() []string {
	keys := make([]string, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, k)
	}
	return keys
}

func (o *Map) SetKey(key string, value core.Object) {
	o.value[key] = value
}

func (o *Map) TypeName() string {
	if o.immutable {
		return "immutable-map"
	}
	return "map"
}

func (o *Map) String() string {
	pairs := make([]string, 0, len(o.value))
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("map({%s})", strings.Join(pairs, ", "))
}

func (o *Map) Interface() any {
	res := make(map[string]any)
	for key, v := range o.value {
		res[key] = v.Interface()
	}
	return res
}

func (o *Map) Arity() int {
	return 0
}

func (o *Map) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Map) Equals(x core.Object) bool {
	if o == x {
		return true
	}

	switch x := x.(type) {
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
	default:
		return false
	}
}

func (o *Map) Copy() core.Object {
	// perform a deep copy of the map even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Object, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy()
	}
	return NewMap(c, false) // copy always returns a mutable map
}

func (o *Map) Access(index core.Object, mode core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("map access", "string", index)
	}
	r, ok := o.value[k]
	if !ok {
		return UndefinedValue, nil
	}
	return r, nil
}

func (o *Map) Assign(index, value core.Object) error {
	if o.immutable {
		return core.NewNotAssignableError(o)
	}

	k, ok := index.AsString()
	if !ok {
		return core.NewInvalidIndexTypeError("map assignment", "string", index)
	}
	o.value[k] = value

	return nil
}

func (o *Map) Iterate() core.Iterator {
	return NewMapIterator(o.value)
}

func (o *Map) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Map) IsFalsy() bool {
	return len(o.value) == 0
}

func (o *Map) IsIterable() bool {
	return true
}

func (o *Map) IsCallable() bool {
	return false
}

func (o *Map) IsImmutable() bool {
	return o.immutable
}

func (o *Map) IsVariadic() bool {
	return false
}

func (o *Map) AsString() (string, bool) {
	return o.String(), true
}

func (o *Map) AsInt() (int64, bool) {
	return 0, false
}

func (o *Map) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Map) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Map) AsRune() (rune, bool) {
	return 0, false
}

func (o *Map) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Map) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

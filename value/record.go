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

type Record struct {
	value     map[string]core.Object
	immutable bool
}

func NewRecord(val map[string]core.Object, immutable bool) *Record {
	o := &Record{}
	o.Set(val, immutable)
	return o
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
	var pairs []string
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, v.String()))
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

func (o *Record) Arity() int {
	return 0
}

func (o *Record) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Record) Equals(x core.Object) bool {
	if o == x {
		return true
	}

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

func (o *Record) Copy() core.Object {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Object, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy()
	}
	return NewRecord(c, false) // copy always returns a mutable record
}

func (o *Record) Access(index core.Object, mode core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("record access", "string", index)
	}
	r, ok := o.value[k]
	if !ok {
		return UndefinedValue, nil
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

func (o *Record) Iterate() core.Iterator {
	return NewMapIterator(o.value)
}

func (o *Record) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Record) IsFalsy() bool {
	return len(o.value) == 0
}

func (o *Record) IsIterable() bool {
	return true
}

func (o *Record) IsCallable() bool {
	return false
}

func (o *Record) IsImmutable() bool {
	return o.immutable
}

func (o *Record) IsVariadic() bool {
	return false
}

func (o *Record) AsString() (string, bool) {
	return o.String(), true
}

func (o *Record) AsInt() (int64, bool) {
	return 0, false
}

func (o *Record) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Record) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Record) AsRune() (rune, bool) {
	return 0, false
}

func (o *Record) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Record) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

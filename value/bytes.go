package value

import (
	"bytes"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Bytes struct {
	value []byte
}

func NewBytes(v []byte) *Bytes {
	o := &Bytes{}
	o.Set(v)
	return o
}

func (o *Bytes) Set(v []byte) {
	o.value = v
	if o.value == nil {
		o.value = make([]byte, 0)
	}
}

func (o *Bytes) Value() []byte {
	return o.value
}

func (o *Bytes) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Bytes) Len() int {
	return len(o.value)
}

func (o *Bytes) Append(v []byte) {
	o.value = append(o.value, v...)
}

func (o *Bytes) At(i int) byte {
	return o.value[i]
}

func (o *Bytes) Get(i int) (byte, bool) {
	if i < 0 || i >= len(o.value) {
		return 0, false
	}
	return o.value[i], true
}

func (o *Bytes) Clear() {
	o.value = o.value[:0]
}

func (o *Bytes) Slice(start, end int) []byte {
	return o.value[start:end]
}

func (o *Bytes) TypeName() string {
	return "bytes"
}

func (o *Bytes) String() string {
	return string(o.value)
}

func (o *Bytes) Interface() any {
	return o.value
}

func (o *Bytes) Arity() int {
	return 0
}

func (o *Bytes) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *Bytes:
			if len(o.value)+len(rhs.value) > core.MaxBytesLen {
				return nil, gse.ErrBytesLimit
			}
			return NewBytes(append(o.value, rhs.value...)), nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *Bytes) Equals(x core.Object) bool {
	if o == x {
		return true
	}

	t, ok := x.AsByteSlice()
	if !ok {
		return false
	}
	return bytes.Equal(o.value, t)
}

func (o *Bytes) Copy() core.Object {
	t := make([]byte, len(o.value))
	copy(t, o.value)
	return NewBytes(t)
}

func (o *Bytes) IndexGet(index core.Object) (core.Object, error) {
	i, ok := index.AsInt()
	if !ok {
		return nil, gse.ErrInvalidIndexType
	}
	if i < 0 || i >= int64(len(o.value)) {
		return UndefinedValue, nil
	}
	return NewInt(int64(o.value[i])), nil
}

func (o *Bytes) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *Bytes) Iterate() core.Iterator {
	return NewBytesIterator(o.value)
}

func (o *Bytes) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Bytes) IsFalsy() bool {
	return len(o.value) == 0
}

func (o *Bytes) IsIterable() bool {
	return true
}

func (o *Bytes) IsCallable() bool {
	return false
}

func (o *Bytes) IsImmutable() bool {
	return false
}

func (o *Bytes) IsVariadic() bool {
	return false
}

func (o *Bytes) AsString() (string, bool) {
	return o.String(), true
}

func (o *Bytes) AsInt() (int64, bool) {
	return 0, false
}

func (o *Bytes) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Bytes) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Bytes) AsRune() (rune, bool) {
	return 0, false
}

func (o *Bytes) AsByteSlice() ([]byte, bool) {
	return o.value, true
}

func (o *Bytes) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

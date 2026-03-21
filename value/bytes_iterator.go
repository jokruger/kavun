package value

import "github.com/jokruger/gs/core"

type BytesIterator struct {
	Object
	v []byte
	i int
	l int
}

func NewBytesIterator(v []byte) *BytesIterator {
	o := &BytesIterator{}
	o.Set(v)
	return o
}

func (o *BytesIterator) Set(v []byte) {
	o.v = v
	o.i = 0
	o.l = len(v)
}

func (o *BytesIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *BytesIterator) Key() core.Object {
	return NewInt(int64(o.i - 1))
}

func (o *BytesIterator) Value() core.Object {
	return NewInt(int64(o.v[o.i-1]))
}

func (o *BytesIterator) TypeName() string {
	return "bytes-iterator"
}

func (o *BytesIterator) String() string {
	return "<bytes-iterator>"
}

func (o *BytesIterator) Copy() core.Object {
	t := NewBytesIterator(o.v)
	t.i = o.i
	return t
}

func (o *BytesIterator) IsFalsy() bool {
	return o.v == nil || o.i > o.l
}

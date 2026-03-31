package value

import "github.com/jokruger/gs/core"

type BytesIterator struct {
	Object
	v []byte
	i int
	l int
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

func (o *BytesIterator) Key(alloc core.Allocator) core.Object {
	return alloc.NewInt(int64(o.i - 1))
}

func (o *BytesIterator) Value(alloc core.Allocator) core.Object {
	return alloc.NewInt(int64(o.v[o.i-1]))
}

func (o *BytesIterator) TypeName() string {
	return "bytes-iterator"
}

func (o *BytesIterator) String() string {
	return "<bytes-iterator>"
}

func (o *BytesIterator) Copy(alloc core.Allocator) core.Object {
	t := alloc.NewBytesIterator(o.v).(*BytesIterator)
	t.i = o.i
	return t
}

func (o *BytesIterator) IsTrue() bool {
	return o.v != nil && o.i <= o.l
}

func (o *BytesIterator) IsFalse() bool {
	return !o.IsTrue()
}

func (o *BytesIterator) IsIterator() bool {
	return true
}

func (o *BytesIterator) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

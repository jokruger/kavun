package value

import "github.com/jokruger/gs/core"

type ArrayIterator struct {
	Object
	v []core.Object
	i int
	l int
}

func NewArrayIterator(v []core.Object) *ArrayIterator {
	o := &ArrayIterator{}
	o.Set(v)
	return o
}

func (o *ArrayIterator) Set(v []core.Object) {
	o.v = v
	o.i = 0
	o.l = len(v)
}

func (o *ArrayIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *ArrayIterator) Key() core.Object {
	return NewInt(int64(o.i - 1))
}

func (o *ArrayIterator) Value() core.Object {
	return o.v[o.i-1]
}

func (o *ArrayIterator) TypeName() string {
	return "array-iterator"
}

func (o *ArrayIterator) String() string {
	return "<array-iterator>"
}

func (o *ArrayIterator) Copy() core.Object {
	t := NewArrayIterator(o.v)
	t.i = o.i
	return t
}

func (o *ArrayIterator) IsFalsy() bool {
	return o.v == nil || o.i > o.l
}

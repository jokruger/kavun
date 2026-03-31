package value

import "github.com/jokruger/gs/core"

type ArrayIterator struct {
	Object
	v []core.Object
	i int
	l int
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

func (o *ArrayIterator) Key(alloc core.Allocator) core.Object {
	return alloc.NewInt(int64(o.i - 1))
}

func (o *ArrayIterator) Value(alloc core.Allocator) core.Object {
	return o.v[o.i-1].Copy(alloc)
}

func (o *ArrayIterator) TypeName() string {
	return "array-iterator"
}

func (o *ArrayIterator) String() string {
	return "<array-iterator>"
}

func (o *ArrayIterator) Copy(alloc core.Allocator) core.Object {
	t := alloc.NewArrayIterator(o.v).(*ArrayIterator)
	t.i = o.i
	return t
}

func (o *ArrayIterator) IsTrue() bool {
	return o.v != nil && o.i <= o.l
}

func (o *ArrayIterator) IsFalse() bool {
	return !o.IsTrue()
}

func (o *ArrayIterator) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

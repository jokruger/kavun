package value

import "github.com/jokruger/gs/core"

type MapIterator struct {
	Object
	v map[string]core.Object
	k []string
	i int
	l int
}

func (o *MapIterator) Set(m map[string]core.Object) {
	o.v = m
	o.k = make([]string, 0, len(m))
	for k := range m {
		o.k = append(o.k, k)
	}
	o.i = 0
	o.l = len(o.k)
}

func (o *MapIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *MapIterator) Key(alloc core.Allocator) core.Object {
	k := o.k[o.i-1]
	return alloc.NewString(k)
}

func (o *MapIterator) Value(alloc core.Allocator) core.Object {
	k := o.k[o.i-1]
	return o.v[k].Copy(alloc)
}

func (o *MapIterator) TypeName() string {
	return "map-iterator"
}

func (o *MapIterator) String() string {
	return "<map-iterator>"
}

func (o *MapIterator) Copy(alloc core.Allocator) core.Object {
	t := alloc.NewMapIterator(o.v).(*MapIterator)
	t.i = o.i
	return t
}

func (o *MapIterator) IsTrue() bool {
	return o.v != nil && o.i <= o.l
}

func (o *MapIterator) IsFalse() bool {
	return !o.IsTrue()
}

func (o *MapIterator) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

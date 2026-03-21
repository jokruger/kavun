package value

import "github.com/jokruger/gs/core"

type MapIterator struct {
	Object
	v map[string]core.Object
	k []string
	i int
	l int
}

func NewMapIterator(m map[string]core.Object) *MapIterator {
	o := &MapIterator{}
	o.Set(m)
	return o
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

func (o *MapIterator) Key() core.Object {
	k := o.k[o.i-1]
	return NewString(k)
}

func (o *MapIterator) Value() core.Object {
	k := o.k[o.i-1]
	return o.v[k]
}

func (o *MapIterator) TypeName() string {
	return "map-iterator"
}

func (o *MapIterator) String() string {
	return "<map-iterator>"
}

func (o *MapIterator) Copy() core.Object {
	t := NewMapIterator(o.v)
	t.i = o.i
	return t
}

func (o *MapIterator) IsFalsy() bool {
	return o.v == nil || o.i > o.l
}

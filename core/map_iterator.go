package core

import (
	"fmt"
	"unsafe"
)

type MapIterator struct {
	v map[string]Value
	k []string
	i int
	l int
}

func (o *MapIterator) Set(m map[string]Value) {
	o.v = m
	o.k = make([]string, 0, len(m))
	for k := range m {
		o.k = append(o.k, k)
	}
	o.i = 0
	o.l = len(o.k)
}

func MapIteratorValue(v *MapIterator) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_MAP_ITERATOR,
	}
}

func NewMapIteratorValue(m map[string]Value) Value {
	t := &MapIterator{}
	t.Set(m)
	return MapIteratorValue(t)
}

func mapIteratorTypeName(v Value) string {
	return "map-iterator"
}

func mapIteratorTypeString(v Value) string {
	i := (*MapIterator)(v.Ptr)
	k := "<nil>"
	if i.i > 0 && i.i <= i.l {
		k = i.k[i.i-1]
	}
	return fmt.Sprintf("MapIterator{%s, %d/%d}", k, i.i, i.l)
}

func mapIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_MAP_ITERATOR {
		return false
	}
	a := (*MapIterator)(v.Ptr)
	b := (*MapIterator)(r.Ptr)
	return a == b
}

package core

import (
	"fmt"
	"unsafe"
)

type MapIterator struct {
	v map[string]Value
	k []string
	i int
}

func (o *MapIterator) Set(m map[string]Value) {
	o.v = m
	o.k = make([]string, 0, len(m))
	for k := range m {
		o.k = append(o.k, k)
	}
	o.i = -1
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
	if i.i >= 0 && i.i < len(i.k) {
		k = i.k[i.i]
	}
	return fmt.Sprintf("MapIterator{%s, %d, %d}", k, i.i, len(i.k))
}

func mapIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_MAP_ITERATOR {
		return false
	}
	a := (*MapIterator)(v.Ptr)
	b := (*MapIterator)(r.Ptr)
	return a == b
}

func mapIteratorTypeNext(v Value) bool {
	i := (*MapIterator)(v.Ptr)
	i.i++
	return i.i < len(i.k)
}

func mapIteratorTypeKey(v Value, alloc Allocator) (Value, error) {
	i := (*MapIterator)(v.Ptr)
	return alloc.NewStringValue(i.k[i.i])
}

func mapIteratorTypeValue(v Value, alloc Allocator) (Value, error) {
	i := (*MapIterator)(v.Ptr)
	k := i.k[i.i]
	return i.v[k], nil
}

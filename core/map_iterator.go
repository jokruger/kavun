package core

import (
	"fmt"
	"unsafe"
)

type MapIterator struct {
	Elements map[string]Value
	Keys     []string
	i        int
}

func (o *MapIterator) Set(m map[string]Value) {
	o.Elements = m
	o.Keys = make([]string, 0, len(m))
	for k := range m {
		o.Keys = append(o.Keys, k)
	}
	o.i = -1
}

func MapIteratorValue(v *MapIterator) Value {
	return Value{
		Type: VT_MAP_ITERATOR,
		Ptr:  unsafe.Pointer(v),
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
	if i.i >= 0 && i.i < len(i.Keys) {
		k = i.Keys[i.i]
	}
	return fmt.Sprintf("MapIterator{%s, %d, %d}", k, i.i, len(i.Keys))
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
	return i.i < len(i.Keys)
}

func mapIteratorTypeKey(v Value, alloc Allocator) (Value, error) {
	i := (*MapIterator)(v.Ptr)
	return alloc.NewStringValue(i.Keys[i.i])
}

func mapIteratorTypeValue(v Value, alloc Allocator) (Value, error) {
	i := (*MapIterator)(v.Ptr)
	k := i.Keys[i.i]
	return i.Elements[k], nil
}

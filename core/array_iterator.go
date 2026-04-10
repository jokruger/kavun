package core

import (
	"fmt"
	"unsafe"
)

type ArrayIterator struct {
	v []Value
	i int
	l int
}

func (i *ArrayIterator) Set(v []Value) {
	i.v = v
	i.i = 0
	i.l = len(v)
}

func ArrayIteratorValue(v *ArrayIterator) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_ARRAY_ITERATOR,
	}
}

func NewArrayIteratorValue(v []Value) Value {
	it := &ArrayIterator{}
	it.Set(v)
	return ArrayIteratorValue(it)
}

func arrayIteratorTypeName(v Value) string {
	return "array-iterator"
}

func arrayIteratorTypeString(v Value) string {
	i := (*ArrayIterator)(v.Ptr)
	return fmt.Sprintf("ArrayIterator{%d/%d}", i.i, i.l)
}

func arrayIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_ARRAY_ITERATOR {
		return false
	}
	a := (*ArrayIterator)(v.Ptr)
	b := (*ArrayIterator)(r.Ptr)
	return a == b
}

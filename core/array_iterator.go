package core

import (
	"fmt"
	"unsafe"
)

type ArrayIterator struct {
	Elements []Value
	i        int
}

func (i *ArrayIterator) Set(v []Value) {
	i.Elements = v
	i.i = -1
}

func ArrayIteratorValue(v *ArrayIterator) Value {
	return Value{
		Type: VT_ARRAY_ITERATOR,
		Ptr:  unsafe.Pointer(v),
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
	return fmt.Sprintf("ArrayIterator{%d, %d}", i.i, len(i.Elements))
}

func arrayIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_ARRAY_ITERATOR {
		return false
	}
	a := (*ArrayIterator)(v.Ptr)
	b := (*ArrayIterator)(r.Ptr)
	return a == b
}

func arrayIteratorTypeNext(v Value) bool {
	i := (*ArrayIterator)(v.Ptr)
	i.i++
	return i.i < len(i.Elements)
}

func arrayIteratorTypeKey(v Value, alloc Allocator) (Value, error) {
	i := (*ArrayIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func arrayIteratorTypeValue(v Value, alloc Allocator) (Value, error) {
	i := (*ArrayIterator)(v.Ptr)
	return i.Elements[i.i], nil
}

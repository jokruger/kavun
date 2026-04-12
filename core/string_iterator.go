package core

import (
	"fmt"
	"unsafe"
)

type StringIterator struct {
	v []rune
	i int
}

func (i *StringIterator) Set(v []rune) {
	i.v = v
	i.i = -1
}

func StringIteratorValue(v *StringIterator) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_STRING_ITERATOR,
	}
}

func NewStringIteratorValue(v []rune) Value {
	i := &StringIterator{}
	i.Set(v)
	return StringIteratorValue(i)
}

func stringIteratorTypeName(v Value) string {
	return "string-iterator"
}

func stringIteratorTypeString(v Value) string {
	i := (*StringIterator)(v.Ptr)
	return fmt.Sprintf("StringIterator{%d, %d}", i.i, len(i.v))
}

func stringIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_STRING_ITERATOR {
		return false
	}
	a := (*StringIterator)(v.Ptr)
	b := (*StringIterator)(r.Ptr)
	return a == b
}

func stringIteratorTypeNext(v *Value) bool {
	i := (*StringIterator)(v.Ptr)
	i.i++
	return i.i < len(i.v)
}

func stringIteratorTypeKey(v Value, alloc Allocator) Value {
	i := (*StringIterator)(v.Ptr)
	return IntValue(int64(i.i))
}

func stringIteratorTypeValue(v Value, alloc Allocator) Value {
	i := (*StringIterator)(v.Ptr)
	return CharValue(i.v[i.i])
}

package core

import (
	"fmt"
	"unsafe"
)

type StringIterator struct {
	v []rune
	i int
	l int
}

func (i *StringIterator) Set(v []rune) {
	i.v = v
	i.i = 0
	i.l = len(v)
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
	return fmt.Sprintf("StringIterator{%d/%d}", i.i, i.l)
}

func stringIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_STRING_ITERATOR {
		return false
	}
	a := (*StringIterator)(v.Ptr)
	b := (*StringIterator)(r.Ptr)
	return a == b
}

package core

import (
	"unsafe"
)

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

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

var TypeArrayIterator = ValueType{
	Name:   ConstHook(arrayIteratorTypeName),
	String: SeqIterStringHook[Value](arrayIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[Value],
	Key:    SeqIterKey[Value],
	Value:  SeqIterValueHook(RefValue),
}

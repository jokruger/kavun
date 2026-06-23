package core

import (
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

func NewArrayIteratorValue(arr []Value) Value {
	o := &ArrayIterator{}
	o.Set(arr)
	return Value{Type: value.ArrayIterator, Ptr: unsafe.Pointer(o)}
}

var TypeArrayIterator = ValueTypeDescr{
	Name:   ConstHook(arrayIteratorTypeName),
	String: SeqIterStringHook[Value](arrayIteratorTypeName, arrayIteratorResolve),
	Next:   SeqIterNextHook[Value](arrayIteratorResolve),
	Key:    SeqIterKeyHook[Value](arrayIteratorResolve),
	Value:  SeqIterValueHook(RefValue, arrayIteratorResolve),
}

func arrayIteratorResolve(v Value) *ArrayIterator {
	return (*ArrayIterator)(v.Ptr)
}

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
	Name:   ConstHook(arrayIteratorTypeName),                                      // PURE by contract
	String: SeqIterStringHook[Value](arrayIteratorTypeName, arrayIteratorResolve), // PURE by contract
	Next:   SeqIterNextHook[Value](arrayIteratorResolve),                          // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    SeqIterKeyHook[Value](arrayIteratorResolve),                           // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  SeqIterValueHook(RefValue, arrayIteratorResolve),                      // LOCALISED-STATE by contract (reads iterator cursor)
}

func arrayIteratorResolve(v Value) *ArrayIterator {
	return (*ArrayIterator)(v.Ptr)
}

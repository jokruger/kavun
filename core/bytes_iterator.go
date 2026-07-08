package core

import (
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Name:   ConstHook(bytesIteratorTypeName),                                     // PURE by contract
	String: SeqIterStringHook[byte](bytesIteratorTypeName, bytesIteratorResolve), // PURE by contract
	Next:   SeqIterNextHook[byte](bytesIteratorResolve),                          // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    SeqIterKeyHook[byte](bytesIteratorResolve),                           // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  SeqIterValueHook(ByteValue, bytesIteratorResolve),                    // LOCALISED-STATE by contract (reads iterator cursor)
}

func NewBytesIteratorValue(b []byte) Value {
	o := &BytesIterator{}
	o.Set(b)
	return Value{Type: value.BytesIterator, Ptr: unsafe.Pointer(o)}
}

func bytesIteratorResolve(v Value) *BytesIterator {
	return (*BytesIterator)(v.Ptr)
}

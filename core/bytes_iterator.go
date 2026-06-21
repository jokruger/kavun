package core

import (
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Name:   ConstHook(bytesIteratorTypeName),
	String: SeqIterStringHook[byte](bytesIteratorTypeName, bytesIteratorResolve),
	Next:   SeqIterNextHook[byte](bytesIteratorResolve),
	Key:    SeqIterKeyHook[byte](bytesIteratorResolve),
	Value:  SeqIterValueHook(ByteValue, bytesIteratorResolve),
}

func NewBytesIteratorValue(b []byte) Value {
	o := &BytesIterator{}
	o.Set(b)
	return Value{Type: value.BytesIterator, Ptr: unsafe.Pointer(o)}
}

func bytesIteratorResolve(v Value) *BytesIterator {
	return (*BytesIterator)(v.Ptr)
}

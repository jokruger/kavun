package core

import (
	"unsafe"
)

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

func BytesIteratorValue(v *BytesIterator) Value {
	return Value{
		Type: VT_BYTES_ITERATOR,
		Ptr:  unsafe.Pointer(v),
	}
}

func NewBytesIteratorValue(vals []byte) Value {
	t := &BytesIterator{}
	t.Set(vals)
	return BytesIteratorValue(t)
}

var TypeBytesIterator = ValueType{
	Name:   ConstHook(bytesIteratorTypeName),
	String: SeqIterStringHook[byte](bytesIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[byte],
	Key:    SeqIterKey[byte],
	Value:  SeqIterValueHook(ByteValue),
}

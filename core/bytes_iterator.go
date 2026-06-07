package core

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinBytesIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainBytesIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseBytesIteratorValue(v) },
	Name:    ConstHook(bytesIteratorTypeName),
	String:  SeqIterStringHook[byte](bytesIteratorTypeName),
	Equal:   SeqIterEqual,
	Next:    SeqIterNext[byte],
	Key:     SeqIterKey[byte],
	Value:   SeqIterValueHook(ByteValue),
}

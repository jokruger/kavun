package core

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinBytesIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainBytesIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseBytesIteratorValue(v) },
	Name:    ConstHook(bytesIteratorTypeName),
	String:  SeqIterStringHook[byte](bytesIteratorTypeName, bytesIteratorResolve),
	Next:    SeqIterNextHook[byte](bytesIteratorResolve),
	Key:     SeqIterKeyHook[byte](bytesIteratorResolve),
	Value:   SeqIterValueHook(ByteValue, bytesIteratorResolve),
}

func bytesIteratorResolve(a *Arena, v Value) *BytesIterator {
	return a.ResolveBytesIteratorValue(v)
}

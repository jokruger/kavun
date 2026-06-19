package core

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Name:   ConstHook(bytesIteratorTypeName),
	String: SeqIterStringHook[byte](bytesIteratorTypeName, bytesIteratorResolve),
	Next:   SeqIterNextHook[byte](bytesIteratorResolve),
	Key:    SeqIterKeyHook[byte](bytesIteratorResolve),
	Value:  SeqIterValueHook(ByteValue, bytesIteratorResolve),
}

func bytesIteratorResolve(a *Arena, v Value) *BytesIterator {
	return a.ResolveBytesIteratorValue(v)
}

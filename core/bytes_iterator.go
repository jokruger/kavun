package core

const bytesIteratorTypeName = "bytes-iterator"

type BytesIterator = SeqIter[byte]

var TypeBytesIterator = ValueTypeDescr{
	Name:   ConstHook(bytesIteratorTypeName),
	String: SeqIterStringHook[byte](bytesIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[byte],
	Key:    SeqIterKey[byte],
	Value:  SeqIterValueHook(ByteValue),
}

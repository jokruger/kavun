package core

import (
	"fmt"
	"unsafe"
)

type BytesIterator struct {
	Elements []byte
	i        int
}

func (o *BytesIterator) Set(vals []byte) {
	o.Elements = vals
	o.i = -1
}

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

func bytesIteratorTypeName(v Value) string {
	return "bytes-iterator"
}

func bytesIteratorTypeString(v Value) string {
	i := (*BytesIterator)(v.Ptr)
	return fmt.Sprintf("BytesIterator{%d, %d}", i.i, len(i.Elements))
}

func bytesIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_BYTES_ITERATOR {
		return false
	}
	a := (*BytesIterator)(v.Ptr)
	b := (*BytesIterator)(r.Ptr)
	return a == b
}

func bytesIteratorTypeNext(v Value) bool {
	i := (*BytesIterator)(v.Ptr)
	i.i++
	return i.i < len(i.Elements)
}

func bytesIteratorTypeKey(v Value, alloc Allocator) (Value, error) {
	i := (*BytesIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func bytesIteratorTypeValue(v Value, alloc Allocator) (Value, error) {
	i := (*BytesIterator)(v.Ptr)
	return IntValue(int64(i.Elements[i.i])), nil
}

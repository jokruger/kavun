package core

import (
	"fmt"
	"unsafe"
)

type BytesIterator struct {
	v []byte
	i int
	l int
}

func (o *BytesIterator) Set(vals []byte) {
	o.v = vals
	o.i = 0
	o.l = len(vals)
}

func BytesIteratorValue(v *BytesIterator) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_BYTES_ITERATOR,
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
	return fmt.Sprintf("BytesIterator{%d/%d}", i.i, i.l)
}

func bytesIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_BYTES_ITERATOR {
		return false
	}
	a := (*BytesIterator)(v.Ptr)
	b := (*BytesIterator)(r.Ptr)
	return a == b
}

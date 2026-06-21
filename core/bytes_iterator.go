package core

import (
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
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

func (a *Arena) MustNewBytesIteratorValue(b []byte) Value {
	v, err := a.NewBytesIteratorValue(b)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewBytesIteratorValue(b []byte) (Value, error) {
	if ref, p, ok := a.arena.New(value.BytesIterator); ok {
		(*BytesIterator)(p).Set(b)
		return Value{Type: value.BytesIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(bytesIteratorTypeName)
}

func bytesIteratorResolve(v Value) *BytesIterator {
	return a.ResolveBytesIteratorValue(v)
}

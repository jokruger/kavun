package core

import (
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

func (a *Arena) MustNewArrayIteratorValue(arr []Value) Value {
	v, err := a.NewArrayIteratorValue(arr)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewArrayIteratorValue(arr []Value) (Value, error) {
	if ref, p, ok := a.arena.New(value.ArrayIterator); ok {
		(*ArrayIterator)(p).Set(arr)
		return Value{Type: value.ArrayIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(arrayIteratorTypeName)
}

var TypeArrayIterator = ValueTypeDescr{
	Name:   ConstHook(arrayIteratorTypeName),
	String: SeqIterStringHook[Value](arrayIteratorTypeName, arrayIteratorResolve),
	Next:   SeqIterNextHook[Value](arrayIteratorResolve),
	Key:    SeqIterKeyHook[Value](arrayIteratorResolve),
	Value:  SeqIterValueHook(RefValue, arrayIteratorResolve),
}

func arrayIteratorResolve(v Value) *ArrayIterator {
	return a.ResolveArrayIteratorValue(v)
}

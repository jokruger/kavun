package core

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

var TypeArrayIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinArrayIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainArrayIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseArrayIteratorValue(v) },
	Name:    ConstHook(arrayIteratorTypeName),
	String:  SeqIterStringHook[Value](arrayIteratorTypeName, arrayIteratorResolve),
	Next:    SeqIterNextHook[Value](arrayIteratorResolve),
	Key:     SeqIterKeyHook[Value](arrayIteratorResolve),
	Value:   SeqIterValueHook(RefValue, arrayIteratorResolve),
}

func arrayIteratorResolve(a *Arena, v Value) *ArrayIterator {
	return a.ResolveArrayIteratorValue(v)
}

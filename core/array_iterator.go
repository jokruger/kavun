package core

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

var TypeArrayIterator = ValueTypeDescr{
	Name:   ConstHook(arrayIteratorTypeName),
	String: SeqIterStringHook[Value](arrayIteratorTypeName, arrayIteratorResolve),
	Next:   SeqIterNextHook[Value](arrayIteratorResolve),
	Key:    SeqIterKeyHook[Value](arrayIteratorResolve),
	Value:  SeqIterValueHook(RefValue, arrayIteratorResolve),
}

func arrayIteratorResolve(a *Arena, v Value) *ArrayIterator {
	return a.ResolveArrayIteratorValue(v)
}

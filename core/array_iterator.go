package core

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

var TypeArrayIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinArrayIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainArrayIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseArrayIteratorValue(v) },
	Name:    ConstHook(arrayIteratorTypeName),
	String:  SeqIterStringHook[Value](arrayIteratorTypeName),
	Equal:   SeqIterEqual,
	Next:    SeqIterNext[Value],
	Key:     SeqIterKey[Value],
	Value:   SeqIterValueHook(RefValue),
}

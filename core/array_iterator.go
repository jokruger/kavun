package core

const arrayIteratorTypeName = "array-iterator"

type ArrayIterator = SeqIter[Value]

var TypeArrayIterator = ValueTypeDescr{
	Name:   ConstHook(arrayIteratorTypeName),
	String: SeqIterStringHook[Value](arrayIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[Value],
	Key:    SeqIterKey[Value],
	Value:  SeqIterValueHook(RefValue),
}

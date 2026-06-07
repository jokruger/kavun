package core

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

var TypeRunesIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinRunesIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainRunesIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseRunesIteratorValue(v) },
	Name:    ConstHook(runesIteratorTypeName),
	String:  SeqIterStringHook[rune](runesIteratorTypeName),
	Equal:   SeqIterEqual,
	Next:    SeqIterNext[rune],
	Key:     SeqIterKey[rune],
	Value:   SeqIterValueHook(RuneValue),
}

package core

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

var TypeRunesIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinRunesIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainRunesIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseRunesIteratorValue(v) },
	Name:    ConstHook(runesIteratorTypeName),
	String:  SeqIterStringHook[rune](runesIteratorTypeName, runesIteratorResolve),
	Next:    SeqIterNextHook[rune](runesIteratorResolve),
	Key:     SeqIterKeyHook[rune](runesIteratorResolve),
	Value:   SeqIterValueHook(RuneValue, runesIteratorResolve),
}

func runesIteratorResolve(a *Arena, v Value) *RunesIterator {
	return a.ResolveRunesIteratorValue(v)
}

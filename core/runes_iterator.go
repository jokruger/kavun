package core

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

var TypeRunesIterator = ValueTypeDescr{
	Name:   ConstHook(runesIteratorTypeName),
	String: SeqIterStringHook[rune](runesIteratorTypeName, runesIteratorResolve),
	Next:   SeqIterNextHook[rune](runesIteratorResolve),
	Key:    SeqIterKeyHook[rune](runesIteratorResolve),
	Value:  SeqIterValueHook(RuneValue, runesIteratorResolve),
}

func runesIteratorResolve(a *Arena, v Value) *RunesIterator {
	return a.ResolveRunesIteratorValue(v)
}

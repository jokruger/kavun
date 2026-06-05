package core

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

var TypeRunesIterator = ValueType{
	Name:   ConstHook(runesIteratorTypeName),
	String: SeqIterStringHook[rune](runesIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[rune],
	Key:    SeqIterKey[rune],
	Value:  SeqIterValueHook(RuneValue),
}

package core

import (
	"unsafe"
)

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

func RunesIteratorValue(v *RunesIterator) Value {
	return Value{
		Type: VT_RUNES_ITERATOR,
		Ptr:  unsafe.Pointer(v),
	}
}

func NewRunesIteratorValue(v []rune) Value {
	i := &RunesIterator{}
	i.Set(v)
	return RunesIteratorValue(i)
}

var TypeRunesIterator = ValueType{
	Name:   ConstHook(runesIteratorTypeName),
	String: SeqIterStringHook[rune](runesIteratorTypeName),
	Equal:  SeqIterEqual,
	Next:   SeqIterNext[rune],
	Key:    SeqIterKey[rune],
	Value:  SeqIterValueHook(RuneValue),
}

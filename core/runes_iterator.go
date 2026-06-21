package core

import (
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

func NewRunesIteratorValue(s []rune) Value {
	o := &RunesIterator{}
	o.Set(s)
	return Value{Type: value.RunesIterator, Ptr: unsafe.Pointer(o)}
}

var TypeRunesIterator = ValueTypeDescr{
	Name:   ConstHook(runesIteratorTypeName),
	String: SeqIterStringHook[rune](runesIteratorTypeName, runesIteratorResolve),
	Next:   SeqIterNextHook[rune](runesIteratorResolve),
	Key:    SeqIterKeyHook[rune](runesIteratorResolve),
	Value:  SeqIterValueHook(RuneValue, runesIteratorResolve),
}

func runesIteratorResolve(v Value) *RunesIterator {
	return (*RunesIterator)(v.Ptr)
}

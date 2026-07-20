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
	Name:   ConstHook(runesIteratorTypeName),                                     // PURE by contract
	String: SeqIterStringHook[rune](runesIteratorTypeName, runesIteratorResolve), // PURE by contract
	Next:   SeqIterNextHook[rune](runesIteratorResolve),                          // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    SeqIterKeyHook[rune](runesIteratorResolve),                           // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  SeqIterValueHook(RuneValue, runesIteratorResolve),                    // LOCALISED-STATE by contract (reads iterator cursor)
}

func runesIteratorResolve(v Value) *RunesIterator {
	return (*RunesIterator)(v.Ptr)
}

package core

import (
	"fmt"
	"unsafe"
)

type RunesIterator struct {
	Elements []rune
	i        int
}

func (i *RunesIterator) Set(v []rune) {
	i.Elements = v
	i.i = -1
}

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

func runesIteratorTypeName(v Value) string {
	return "runes-iterator"
}

func runesIteratorTypeString(v Value) string {
	i := (*RunesIterator)(v.Ptr)
	return fmt.Sprintf("RunesIterator{%d, %d}", i.i, len(i.Elements))
}

func runesIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_RUNES_ITERATOR {
		return false
	}
	a := (*RunesIterator)(v.Ptr)
	b := (*RunesIterator)(r.Ptr)
	return a == b
}

func runesIteratorTypeNext(v Value) bool {
	i := (*RunesIterator)(v.Ptr)
	i.i++
	return i.i < len(i.Elements)
}

func runesIteratorTypeKey(v Value, a *Arena) (Value, error) {
	i := (*RunesIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func runesIteratorTypeValue(v Value, a *Arena) (Value, error) {
	i := (*RunesIterator)(v.Ptr)
	return RuneValue(i.Elements[i.i]), nil
}

package core

import (
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

const runesIteratorTypeName = "runes-iterator"

type RunesIterator = SeqIter[rune]

func (a *Arena) MustNewRunesIteratorValue(s []rune) Value {
	v, err := a.NewRunesIteratorValue(s)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewRunesIteratorValue(s []rune) (Value, error) {
	if ref, p, ok := a.arena.New(value.RunesIterator); ok {
		(*RunesIterator)(p).Set(s)
		return Value{Type: value.RunesIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(runesIteratorTypeName)
}

var TypeRunesIterator = ValueTypeDescr{
	Name:   ConstHook(runesIteratorTypeName),
	String: SeqIterStringHook[rune](runesIteratorTypeName, runesIteratorResolve),
	Next:   SeqIterNextHook[rune](runesIteratorResolve),
	Key:    SeqIterKeyHook[rune](runesIteratorResolve),
	Value:  SeqIterValueHook(RuneValue, runesIteratorResolve),
}

func runesIteratorResolve(v Value) *RunesIterator {
	return a.ResolveRunesIteratorValue(v)
}

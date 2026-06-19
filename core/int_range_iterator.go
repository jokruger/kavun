package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/value"
)

const intRangeIteratorTypeName = "range-iterator"

type IntRangeIterator struct {
	i int64 // current index
	v int64 // current value
	l int64 // last value
	s int64 // step
}

func (i *IntRangeIterator) Set(start, stop, step int64) {
	i.i = -1
	i.l = stop
	if start <= stop {
		i.v = start - step
		i.s = step
	} else {
		i.v = start + step
		i.s = -step
	}
}

var TypeIntRangeIterator = ValueTypeDescr{
	Name:   ConstHook(intRangeIteratorTypeName),
	String: intRangeIteratorTypeString,
	Equal:  intRangeIteratorTypeEqual,
	Next:   intRangeIteratorTypeNext,
	Key:    intRangeIteratorTypeKey,
	Value:  intRangeIteratorTypeValue,
}

func intRangeIteratorTypeString(a *Arena, v Value) string {
	i := a.ResolveIntRangeIteratorValue(v)
	return fmt.Sprintf("RangeIterator{%d, %d, %d, %d}", i.i, i.v, i.l, i.s)
}

func intRangeIteratorTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != value.IntRangeIterator {
		return false
	}
	x := a.ResolveIntRangeIteratorValue(v)
	y := a.ResolveIntRangeIteratorValue(r)
	return *x == *y
}

func intRangeIteratorTypeNext(a *Arena, v Value) bool {
	i := a.ResolveIntRangeIteratorValue(v)
	i.i++
	i.v += i.s
	if i.s > 0 {
		return i.v < i.l
	}
	return i.v > i.l
}

func intRangeIteratorTypeKey(a *Arena, v Value) (Value, error) {
	i := a.ResolveIntRangeIteratorValue(v)
	return IntValue(int64(i.i)), nil
}

func intRangeIteratorTypeValue(a *Arena, v Value) (Value, error) {
	i := a.ResolveIntRangeIteratorValue(v)
	return IntValue(i.v), nil
}

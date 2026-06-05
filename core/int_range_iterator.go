package core

import (
	"fmt"
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

var TypeIntRangeIterator = ValueType{
	Name:   ConstHook(intRangeIteratorTypeName),
	String: intRangeIteratorTypeString,
	Equal:  intRangeIteratorTypeEqual,
	Next:   intRangeIteratorTypeNext,
	Key:    intRangeIteratorTypeKey,
	Value:  intRangeIteratorTypeValue,
}

func intRangeIteratorTypeString(a *Arena, v Value) string {
	i := (*IntRangeIterator)(v.Ptr)
	return fmt.Sprintf("RangeIterator{%d, %d, %d, %d}", i.i, i.v, i.l, i.s)
}

func intRangeIteratorTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_INT_RANGE_ITERATOR {
		return false
	}
	x := (*IntRangeIterator)(v.Ptr)
	y := (*IntRangeIterator)(r.Ptr)
	return *x == *y
}

func intRangeIteratorTypeNext(a *Arena, v Value) bool {
	i := (*IntRangeIterator)(v.Ptr)
	i.i++
	i.v += i.s
	if i.s > 0 {
		return i.v < i.l
	}
	return i.v > i.l
}

func intRangeIteratorTypeKey(a *Arena, v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func intRangeIteratorTypeValue(a *Arena, v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(i.v), nil
}

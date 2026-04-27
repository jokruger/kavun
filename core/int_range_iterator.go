package core

import (
	"fmt"
	"unsafe"
)

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

func IntRangeIteratorValue(v *IntRangeIterator) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_INT_RANGE_ITERATOR,
	}
}

func NewIntRangeIteratorValue(start, stop, step int64) Value {
	it := &IntRangeIterator{}
	it.Set(start, stop, step)
	return IntRangeIteratorValue(it)
}

func intRangeIteratorTypeName(v Value) string {
	return "range-iterator"
}

func intRangeIteratorTypeString(v Value) string {
	i := (*IntRangeIterator)(v.Ptr)
	return fmt.Sprintf("RangeIterator{%d, %d, %d, %d}", i.i, i.v, i.l, i.s)
}

func intRangeIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != VT_INT_RANGE_ITERATOR {
		return false
	}
	a := (*IntRangeIterator)(v.Ptr)
	b := (*IntRangeIterator)(r.Ptr)
	return *a == *b
}

func intRangeIteratorTypeNext(v Value) bool {
	i := (*IntRangeIterator)(v.Ptr)
	i.i++
	i.v += i.s
	if i.s > 0 {
		return i.v < i.l
	}
	return i.v > i.l
}

func intRangeIteratorTypeKey(v Value, a *Arena) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func intRangeIteratorTypeValue(v Value, a *Arena) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(i.v), nil
}

package core

import (
	"fmt"
	"unsafe"

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

func NewIntRangeIteratorValue(start, stop, step int64) Value {
	o := &IntRangeIterator{}
	o.Set(start, stop, step)
	return Value{Type: value.IntRangeIterator, Ptr: unsafe.Pointer(o)}
}

var TypeIntRangeIterator = ValueTypeDescr{
	Name:   ConstHook(intRangeIteratorTypeName),
	String: intRangeIteratorTypeString,
	Equal:  intRangeIteratorTypeEqual,
	Next:   intRangeIteratorTypeNext,
	Key:    intRangeIteratorTypeKey,
	Value:  intRangeIteratorTypeValue,
}

func intRangeIteratorTypeString(v Value) string {
	i := (*IntRangeIterator)(v.Ptr)
	return fmt.Sprintf("RangeIterator{%d, %d, %d, %d}", i.i, i.v, i.l, i.s)
}

func intRangeIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != value.IntRangeIterator {
		return false
	}
	x := (*IntRangeIterator)(v.Ptr)
	y := (*IntRangeIterator)(r.Ptr)
	return *x == *y
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

func intRangeIteratorTypeKey(v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func intRangeIteratorTypeValue(v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(i.v), nil
}

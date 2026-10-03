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
	n int64 // element count
	s int64 // signed step
}

func (i *IntRangeIterator) Set(start, stop, step int64) {
	// it stops by count, not by comparing the value against stop: one step past the last element may lie
	// past int64 and wrap. The value itself may wrap freely — every element it is read at is exact
	i.i = -1
	i.n = int64(intRangeCount(start, stop, step))
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
	Name:   ConstHook(intRangeIteratorTypeName), // PURE by contract
	String: intRangeIteratorTypeString,          // PURE by contract
	Next:   intRangeIteratorTypeNext,            // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    intRangeIteratorTypeKey,             // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  intRangeIteratorTypeValue,           // LOCALISED-STATE by contract (reads iterator cursor)
}

func intRangeIteratorTypeString(v Value) string {
	i := (*IntRangeIterator)(v.Ptr)
	return fmt.Sprintf("RangeIterator{%d, %d, %d, %d}", i.i, i.v, i.n, i.s)
}

func intRangeIteratorTypeNext(v Value) bool {
	i := (*IntRangeIterator)(v.Ptr)
	i.i++
	i.v += i.s
	return i.i < i.n
}

func intRangeIteratorTypeKey(v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func intRangeIteratorTypeValue(v Value) (Value, error) {
	i := (*IntRangeIterator)(v.Ptr)
	return IntValue(i.v), nil
}

package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const arrayIteratorTypeName = "array-iterator"

// ArrayIterator walks an array's elements by position; the cursor starts before the first element.
type ArrayIterator struct {
	Elements []Value
	i        int
}

func NewArrayIteratorValue(elems []Value) Value {
	return Value{Type: value.ArrayIterator, Ptr: unsafe.Pointer(&ArrayIterator{Elements: elems, i: -1})}
}

var TypeArrayIterator = ValueTypeDescr{
	Name:   ConstHook(arrayIteratorTypeName), // PURE by contract
	String: arrayIteratorString,              // PURE by contract
	Next:   arrayIteratorNext,                // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    arrayIteratorKey,                 // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  arrayIteratorValue,               // LOCALISED-STATE by contract (reads iterator cursor)
}

// PURE by contract
func arrayIteratorString(v Value) string {
	o := (*ArrayIterator)(v.Ptr)
	return fmt.Sprintf("%s<%d, %d>", arrayIteratorTypeName, o.i, len(o.Elements))
}

// LOCALISED-STATE: advances the iterator's internal cursor. Iterators are held by a single consumer for the
// duration of iteration; the optimizer never speculatively evaluates iterator advancement. See docs/purity.md.
func arrayIteratorNext(v Value) bool {
	o := (*ArrayIterator)(v.Ptr)
	o.i++
	return o.i < len(o.Elements)
}

// PURE: reads the iterator's current cursor without advancing it. See docs/purity.md.
func arrayIteratorKey(v Value) (Value, error) {
	return IntValue(int64((*ArrayIterator)(v.Ptr).i)), nil
}

// PURE: reads the iterator's current element without advancing it. See docs/purity.md.
func arrayIteratorValue(v Value) (Value, error) {
	o := (*ArrayIterator)(v.Ptr)
	return (o.Elements[o.i]), nil
}

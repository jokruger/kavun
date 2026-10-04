package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const bytesIteratorTypeName = "bytes-iterator"

// BytesIterator walks bytes' octets by position; the cursor starts before the first element.
type BytesIterator struct {
	Elements []byte
	i        int
}

func NewBytesIteratorValue(elems []byte) Value {
	return Value{Type: value.BytesIterator, Ptr: unsafe.Pointer(&BytesIterator{Elements: elems, i: -1})}
}

var TypeBytesIterator = ValueTypeDescr{
	Name:   ConstHook(bytesIteratorTypeName), // PURE by contract
	String: bytesIteratorString,              // PURE by contract
	Next:   bytesIteratorNext,                // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    bytesIteratorKey,                 // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  bytesIteratorValue,               // LOCALISED-STATE by contract (reads iterator cursor)
}

// PURE by contract
func bytesIteratorString(v Value) string {
	o := (*BytesIterator)(v.Ptr)
	return fmt.Sprintf("%s<%d, %d>", bytesIteratorTypeName, o.i, len(o.Elements))
}

// LOCALISED-STATE: advances the iterator's internal cursor. Iterators are held by a single consumer for the
// duration of iteration; the optimizer never speculatively evaluates iterator advancement. See docs/purity.md.
func bytesIteratorNext(v Value) bool {
	o := (*BytesIterator)(v.Ptr)
	o.i++
	return o.i < len(o.Elements)
}

// PURE: reads the iterator's current cursor without advancing it. See docs/purity.md.
func bytesIteratorKey(v Value) (Value, error) {
	return IntValue(int64((*BytesIterator)(v.Ptr).i)), nil
}

// PURE: reads the iterator's current element without advancing it. See docs/purity.md.
func bytesIteratorValue(v Value) (Value, error) {
	o := (*BytesIterator)(v.Ptr)
	return ByteValue(o.Elements[o.i]), nil
}

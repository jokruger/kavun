package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const runesIteratorTypeName = "runes-iterator"

// RunesIterator walks a text's symbols by position; the cursor starts before the first element.
type RunesIterator struct {
	Elements []rune
	i        int
}

func NewRunesIteratorValue(elems []rune) Value {
	return Value{Type: value.RunesIterator, Ptr: unsafe.Pointer(&RunesIterator{Elements: elems, i: -1})}
}

var TypeRunesIterator = ValueTypeDescr{
	Name:   ConstHook(runesIteratorTypeName), // PURE by contract
	String: runesIteratorString,              // PURE by contract
	Next:   runesIteratorNext,                // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    runesIteratorKey,                 // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  runesIteratorValue,               // LOCALISED-STATE by contract (reads iterator cursor)
}

// PURE by contract
func runesIteratorString(v Value) string {
	o := (*RunesIterator)(v.Ptr)
	return fmt.Sprintf("%s<%d, %d>", runesIteratorTypeName, o.i, len(o.Elements))
}

// LOCALISED-STATE: advances the iterator's internal cursor. Iterators are held by a single consumer for the
// duration of iteration; the optimizer never speculatively evaluates iterator advancement. See docs/purity.md.
func runesIteratorNext(v Value) bool {
	o := (*RunesIterator)(v.Ptr)
	o.i++
	return o.i < len(o.Elements)
}

// PURE: reads the iterator's current cursor without advancing it. See docs/purity.md.
func runesIteratorKey(v Value) (Value, error) {
	return IntValue(int64((*RunesIterator)(v.Ptr).i)), nil
}

// PURE: reads the iterator's current element without advancing it. See docs/purity.md.
func runesIteratorValue(v Value) (Value, error) {
	o := (*RunesIterator)(v.Ptr)
	return RuneValue(o.Elements[o.i]), nil
}

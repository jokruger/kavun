package core

import (
	"fmt"
	"slices"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const dictIteratorTypeName = "dict-iterator"

type DictIterator struct {
	Elements map[string]Value
	Keys     []string
	i        int
}

// Set snapshots the map's keys in sorted order. Iteration over a map is ORDERED: `for k in d` and
// `for k, v in d` visit keys lexically, on a `record` as much as a `dict` (both types iterate through this
// one iterator). The order is part of the contract, not an accident of the Go map — a script that folds,
// prints or accumulates over a map is reproducible run to run, and agrees with `keys()`/`values()`/`array()`
// and every other member, all of which sort too.
func (o *DictIterator) Set(m map[string]Value) {
	o.Elements = m
	o.Keys = make([]string, 0, len(m))
	for k := range m {
		o.Keys = append(o.Keys, k)
	}
	slices.Sort(o.Keys)
	o.i = -1
}

func NewDictIteratorValue(m map[string]Value) Value {
	o := &DictIterator{}
	o.Set(m)
	return Value{Type: value.DictIterator, Ptr: unsafe.Pointer(o)}
}

var TypeDictIterator = ValueTypeDescr{
	Name:   ConstHook(dictIteratorTypeName), // PURE by contract
	String: dictIteratorTypeString,          // PURE by contract
	Next:   dictIteratorTypeNext,            // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    dictIteratorTypeKey,             // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  dictIteratorTypeValue,           // LOCALISED-STATE by contract (reads iterator cursor)
	Elem:   dictIteratorTypeKey,             // a map's element is its KEY: `for k in d` yields keys (the attachment is the two-variable form's second binding)
}

func dictIteratorTypeString(v Value) string {
	i := (*DictIterator)(v.Ptr)
	k := "<nil>"
	if i.i >= 0 && i.i < len(i.Keys) {
		k = i.Keys[i.i]
	}
	return fmt.Sprintf("DictIterator{%s, %d, %d}", k, i.i, len(i.Keys))
}

func dictIteratorTypeNext(v Value) bool {
	i := (*DictIterator)(v.Ptr)
	i.i++
	return i.i < len(i.Keys)
}

func dictIteratorTypeKey(v Value) (Value, error) {
	i := (*DictIterator)(v.Ptr)
	return NewStringValue(i.Keys[i.i]), nil
}

func dictIteratorTypeValue(v Value) (Value, error) {
	i := (*DictIterator)(v.Ptr)
	k := i.Keys[i.i]
	return i.Elements[k], nil
}

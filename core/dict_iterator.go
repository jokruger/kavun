package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const dictIteratorTypeName = "dict-iterator"

type DictIterator struct {
	Elements map[string]Value
	Keys     []string
	i        int
}

func (o *DictIterator) Set(m map[string]Value) {
	o.Elements = m
	o.Keys = make([]string, 0, len(m))
	for k := range m {
		o.Keys = append(o.Keys, k)
	}
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
	Equal:  dictIteratorTypeEqual,           // PURE by contract
	Next:   dictIteratorTypeNext,            // LOCALISED-STATE by contract (advances iterator cursor)
	Key:    dictIteratorTypeKey,             // LOCALISED-STATE by contract (reads iterator cursor)
	Value:  dictIteratorTypeValue,           // LOCALISED-STATE by contract (reads iterator cursor)
}

func dictIteratorTypeString(v Value) string {
	i := (*DictIterator)(v.Ptr)
	k := "<nil>"
	if i.i >= 0 && i.i < len(i.Keys) {
		k = i.Keys[i.i]
	}
	return fmt.Sprintf("DictIterator{%s, %d, %d}", k, i.i, len(i.Keys))
}

func dictIteratorTypeEqual(v Value, r Value) bool {
	if r.Type != value.DictIterator {
		return false
	}
	x := (*DictIterator)(v.Ptr)
	y := (*DictIterator)(r.Ptr)
	return x == y
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

package core

import (
	"fmt"
	"unsafe"
)

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

func DictIteratorValue(v *DictIterator) Value {
	return Value{
		Type: VT_DICT_ITERATOR,
		Ptr:  unsafe.Pointer(v),
	}
}

func NewDictIteratorValue(m map[string]Value) Value {
	t := &DictIterator{}
	t.Set(m)
	return DictIteratorValue(t)
}

func dictIteratorTypeName(v Value) string {
	return "dict-iterator"
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
	if r.Type != VT_DICT_ITERATOR {
		return false
	}
	a := (*DictIterator)(v.Ptr)
	b := (*DictIterator)(r.Ptr)
	return a == b
}

func dictIteratorTypeNext(v Value) bool {
	i := (*DictIterator)(v.Ptr)
	i.i++
	return i.i < len(i.Keys)
}

func dictIteratorTypeKey(v Value, a *Arena) (Value, error) {
	i := (*DictIterator)(v.Ptr)
	return a.NewStringValue(i.Keys[i.i]), nil
}

func dictIteratorTypeValue(v Value, a *Arena) (Value, error) {
	i := (*DictIterator)(v.Ptr)
	k := i.Keys[i.i]
	return i.Elements[k], nil
}

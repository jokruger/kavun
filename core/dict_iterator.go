package core

import (
	"fmt"
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

var TypeDictIterator = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinDictIteratorValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainDictIteratorValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseDictIteratorValue(v) },
	Name:    ConstHook(dictIteratorTypeName),
	String:  dictIteratorTypeString,
	Equal:   dictIteratorTypeEqual,
	Next:    dictIteratorTypeNext,
	Key:     dictIteratorTypeKey,
	Value:   dictIteratorTypeValue,
}

func dictIteratorTypeString(a *Arena, v Value) string {
	i := a.ResolveDictIteratorValue(v)
	k := "<nil>"
	if i.i >= 0 && i.i < len(i.Keys) {
		k = i.Keys[i.i]
	}
	return fmt.Sprintf("DictIterator{%s, %d, %d}", k, i.i, len(i.Keys))
}

func dictIteratorTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_DICT_ITERATOR {
		return false
	}
	x := a.ResolveDictIteratorValue(v)
	y := a.ResolveDictIteratorValue(r)
	return x == y
}

func dictIteratorTypeNext(a *Arena, v Value) bool {
	i := a.ResolveDictIteratorValue(v)
	i.i++
	return i.i < len(i.Keys)
}

func dictIteratorTypeKey(a *Arena, v Value) (Value, error) {
	i := a.ResolveDictIteratorValue(v)
	return a.NewStringValue(i.Keys[i.i])
}

func dictIteratorTypeValue(a *Arena, v Value) (Value, error) {
	i := a.ResolveDictIteratorValue(v)
	k := i.Keys[i.i]
	return i.Elements[k], nil
}

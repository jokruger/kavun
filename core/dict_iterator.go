package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
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

func (a *Arena) MustNewDictIteratorValue(m map[string]Value) Value {
	v, err := a.NewDictIteratorValue(m)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewDictIteratorValue(m map[string]Value) (Value, error) {
	if ref, p, ok := a.arena.New(value.DictIterator); ok {
		(*DictIterator)(p).Set(m)
		return Value{Type: value.DictIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(dictIteratorTypeName)
}

var TypeDictIterator = ValueTypeDescr{
	Name:   ConstHook(dictIteratorTypeName),
	String: dictIteratorTypeString,
	Equal:  dictIteratorTypeEqual,
	Next:   dictIteratorTypeNext,
	Key:    dictIteratorTypeKey,
	Value:  dictIteratorTypeValue,
}

func dictIteratorTypeString(v Value) string {
	i := a.ResolveDictIteratorValue(v)
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
	x := a.ResolveDictIteratorValue(v)
	y := a.ResolveDictIteratorValue(r)
	return x == y
}

func dictIteratorTypeNext(v Value) bool {
	i := a.ResolveDictIteratorValue(v)
	i.i++
	return i.i < len(i.Keys)
}

func dictIteratorTypeKey(v Value) (Value, error) {
	i := a.ResolveDictIteratorValue(v)
	return a.NewStringValue(i.Keys[i.i])
}

func dictIteratorTypeValue(v Value) (Value, error) {
	i := a.ResolveDictIteratorValue(v)
	k := i.Keys[i.i]
	return i.Elements[k], nil
}

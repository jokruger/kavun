package alloc

import (
	"time"

	"github.com/jokruger/gs/core"
)

type Allocator struct {
}

func New() core.Allocator {
	return &Allocator{}
}

func (a *Allocator) NewBuiltinFunctionValue(name string, fn core.NativeFunc, arity int, variadic bool) core.Value {
	o := &core.BuiltinFunction{}
	o.Set(fn, name, arity, variadic)
	return core.BuiltinFunctionValue(o)
}

func (a *Allocator) NewErrorValue(e core.Value) core.Value {
	o := &core.Error{}
	o.Set(e)
	return core.ErrorValue(o)
}

func (a *Allocator) NewTimeValue(t time.Time) core.Value {
	o := &t
	return core.TimeValue(o)
}

func (a *Allocator) NewStringValue(s string) core.Value {
	o := &core.String{}
	o.Set(s)
	return core.StringValue(o)
}

func (a *Allocator) NewStringIteratorValue(s []rune) core.Value {
	o := &core.StringIterator{}
	o.Set(s)
	return core.StringIteratorValue(o)
}

func (a *Allocator) NewBytesValue(b []byte) core.Value {
	o := &core.Bytes{}
	o.Set(b)
	return core.BytesValue(o)
}

func (a *Allocator) NewBytesIteratorValue(b []byte) core.Value {
	o := &core.BytesIterator{}
	o.Set(b)
	return core.BytesIteratorValue(o)
}

func (a *Allocator) NewArrayValue(arr []core.Value, immutable bool) core.Value {
	o := &core.Array{}
	o.Set(arr, immutable)
	return core.ArrayValue(o)
}

func (a *Allocator) NewArrayIteratorValue(arr []core.Value) core.Value {
	o := &core.ArrayIterator{}
	o.Set(arr)
	return core.ArrayIteratorValue(o)
}

func (a *Allocator) NewMapValue(m map[string]core.Value, immutable bool) core.Value {
	o := &core.Map{}
	o.Set(m, immutable)
	return core.MapValue(o)
}

func (a *Allocator) NewMapIteratorValue(m map[string]core.Value) core.Value {
	o := &core.MapIterator{}
	o.Set(m)
	return core.MapIteratorValue(o)
}

func (a *Allocator) NewRecordValue(m map[string]core.Value, immutable bool) core.Value {
	o := &core.Record{}
	o.Set(m, immutable)
	return core.RecordValue(o)
}

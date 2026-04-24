package alloc

import (
	"time"

	"github.com/jokruger/kavun/core"
)

type Allocator struct {
}

func New() core.Allocator {
	return &Allocator{}
}

func (a *Allocator) Reset() {
}

/* Low-level resources */

func (a *Allocator) NewDecimal() (*core.Decimal, error) {
	o := &core.Decimal{}
	return o, nil
}

func (a *Allocator) NewTime() (*time.Time, error) {
	o := &time.Time{}
	return o, nil
}

func (a *Allocator) NewRunes(capacity int, resize bool) ([]rune, error) {
	o := make([]rune, 0, capacity)
	if resize {
		o = o[:capacity]
	}
	return o, nil
}

func (a *Allocator) NewBytes(capacity int, resize bool) ([]byte, error) {
	o := make([]byte, 0, capacity)
	if resize {
		o = o[:capacity]
	}
	return o, nil
}

func (a *Allocator) NewArray(capacity int, resize bool) ([]core.Value, error) {
	o := make([]core.Value, 0, capacity)
	if resize {
		o = o[:capacity]
	}
	return o, nil
}

func (a *Allocator) NewMap(capacity int) (map[string]core.Value, error) {
	o := make(map[string]core.Value, capacity)
	return o, nil
}

/* Value envelopes */

func (a *Allocator) NewBuiltinFunctionValue(name string, fn core.NativeFunc, arity int8, variadic bool) (core.Value, error) {
	o := &core.BuiltinFunction{}
	o.Set(fn, name, arity, variadic)
	return core.BuiltinFunctionValue(o), nil
}

func (a *Allocator) NewCompiledFunctionValue(instructions []byte, free []*core.Value, sourceMap map[int]core.Pos, numLocals int, numParameters int8, varArgs bool) (core.Value, error) {
	o := &core.CompiledFunction{}
	o.Set(instructions, free, sourceMap, numLocals, numParameters, varArgs)
	return core.CompiledFunctionValue(o), nil
}

func (a *Allocator) NewErrorValue(e core.Value) (core.Value, error) {
	o := &core.Error{}
	o.Set(e)
	return core.ErrorValue(o), nil
}

func (a *Allocator) NewStringValue(s string) (core.Value, error) {
	o := &core.String{}
	o.Set(s)
	return core.StringValue(o), nil
}

func (a *Allocator) NewRunesValue(r []rune) (core.Value, error) {
	o := &core.Runes{}
	o.Set(r)
	return core.RunesValue(o), nil
}

func (a *Allocator) NewBytesValue(b []byte) (core.Value, error) {
	o := &core.Bytes{}
	o.Set(b)
	return core.BytesValue(o), nil
}

func (a *Allocator) NewArrayValue(arr []core.Value, immutable bool) (core.Value, error) {
	o := &core.Array{}
	o.Set(arr)
	return core.ArrayValue(o, immutable), nil
}

func (a *Allocator) NewMapValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	o := &core.Map{}
	o.Set(m)
	return core.MapValue(o, immutable), nil
}

func (a *Allocator) NewRecordValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	o := &core.Map{}
	o.Set(m)
	return core.RecordValue(o, immutable), nil
}

func (a *Allocator) NewIntRangeValue(start, stop, step int64) (core.Value, error) {
	o := &core.IntRange{}
	o.Set(start, stop, step)
	return core.IntRangeValue(o), nil
}

func (a *Allocator) NewRunesIteratorValue(s []rune) (core.Value, error) {
	o := &core.RunesIterator{}
	o.Set(s)
	return core.RunesIteratorValue(o), nil
}

func (a *Allocator) NewBytesIteratorValue(b []byte) (core.Value, error) {
	o := &core.BytesIterator{}
	o.Set(b)
	return core.BytesIteratorValue(o), nil
}

func (a *Allocator) NewArrayIteratorValue(arr []core.Value) (core.Value, error) {
	o := &core.ArrayIterator{}
	o.Set(arr)
	return core.ArrayIteratorValue(o), nil
}

func (a *Allocator) NewMapIteratorValue(m map[string]core.Value) (core.Value, error) {
	o := &core.MapIterator{}
	o.Set(m)
	return core.MapIteratorValue(o), nil
}

func (a *Allocator) NewIntRangeIteratorValue(start, stop, step int64) (core.Value, error) {
	o := &core.IntRangeIterator{}
	o.Set(start, stop, step)
	return core.IntRangeIteratorValue(o), nil
}

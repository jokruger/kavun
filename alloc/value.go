package alloc

import (
	"github.com/jokruger/kavun/core"
)

func (a *Allocator) ReleaseValue(v core.Value) {
}

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

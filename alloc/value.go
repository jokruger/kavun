package alloc

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func (a *Allocator) ReleaseValue(v core.Value) {
	switch v.Type {
	case core.VT_BUILTIN_FUNCTION:
		a.allocs++
	case core.VT_COMPILED_FUNCTION:
		a.allocs++
	case core.VT_ERROR:
		a.allocs++
	case core.VT_INT_RANGE:
		a.allocs++
	case core.VT_RUNES_ITERATOR:
		a.allocs++
	case core.VT_BYTES_ITERATOR:
		a.allocs++
	case core.VT_ARRAY_ITERATOR:
		a.allocs++
	case core.VT_MAP_ITERATOR:
		a.allocs++
	case core.VT_INT_RANGE_ITERATOR:
		a.allocs++
	}
}

func (a *Allocator) NewBuiltinFunctionValue(name string, fn core.NativeFunc, arity int8, variadic bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.BuiltinFunction{}
	o.Set(fn, name, arity, variadic)
	return core.BuiltinFunctionValue(o), nil
}

func (a *Allocator) NewCompiledFunctionValue(instructions []byte, free []*core.Value, sourceMap map[int]core.Pos, numLocals int, numParameters int8, varArgs bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.CompiledFunction{}
	o.Set(instructions, free, sourceMap, numLocals, numParameters, varArgs)
	return core.CompiledFunctionValue(o), nil
}

func (a *Allocator) NewErrorValue(e core.Value) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Error{}
	o.Set(e)
	return core.ErrorValue(o), nil
}

func (a *Allocator) NewStringValue(s string) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.String{}
	o.Set(s)
	return core.StringValue(o), nil
}

func (a *Allocator) NewIntRangeValue(start, stop, step int64) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.IntRange{}
	o.Set(start, stop, step)
	return core.IntRangeValue(o), nil
}

func (a *Allocator) NewRunesIteratorValue(s []rune) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.RunesIterator{}
	o.Set(s)
	return core.RunesIteratorValue(o), nil
}

func (a *Allocator) NewBytesIteratorValue(b []byte) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.BytesIterator{}
	o.Set(b)
	return core.BytesIteratorValue(o), nil
}

func (a *Allocator) NewArrayIteratorValue(arr []core.Value) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.ArrayIterator{}
	o.Set(arr)
	return core.ArrayIteratorValue(o), nil
}

func (a *Allocator) NewMapIteratorValue(m map[string]core.Value) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.MapIterator{}
	o.Set(m)
	return core.MapIteratorValue(o), nil
}

func (a *Allocator) NewIntRangeIteratorValue(start, stop, step int64) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.IntRangeIterator{}
	o.Set(start, stop, step)
	return core.IntRangeIteratorValue(o), nil
}

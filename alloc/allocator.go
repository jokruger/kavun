package alloc

import (
	"math"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

type Allocator struct {
	allocs uint64
}

// New returns a new Allocator with the given maximum number of allocations. If maxAllocs is 0, then the maximum number of allocations is 2^64 - 1.
// Allocator must be used in a single-threaded context only.
func New(maxAllocs uint64) core.Allocator {
	if maxAllocs == 0 {
		maxAllocs = math.MaxUint64
	}
	return &Allocator{
		allocs: maxAllocs,
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

func (a *Allocator) NewErrorValue(e core.Value) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Error{}
	o.Set(e)
	return core.ErrorValue(o), nil
}

func (a *Allocator) NewTimeValue(t time.Time) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &t
	return core.TimeValue(o), nil
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

func (a *Allocator) NewStringIteratorValue(s []rune) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.StringIterator{}
	o.Set(s)
	return core.StringIteratorValue(o), nil
}

func (a *Allocator) NewBytesValue(b []byte) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Bytes{}
	o.Set(b)
	return core.BytesValue(o), nil
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

func (a *Allocator) NewArrayValue(arr []core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Array{}
	o.Set(arr, immutable)
	return core.ArrayValue(o), nil
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

func (a *Allocator) NewMapValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Map{}
	o.Set(m, immutable)
	return core.MapValue(o), nil
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

func (a *Allocator) NewRecordValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Record{}
	o.Set(m, immutable)
	return core.RecordValue(o), nil
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

func (a *Allocator) NewIntRangeIteratorValue(start, stop, step int64) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.IntRangeIterator{}
	o.Set(start, stop, step)
	return core.IntRangeIteratorValue(o), nil
}

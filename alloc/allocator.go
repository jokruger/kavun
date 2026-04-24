package alloc

import (
	"math"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

type Allocator struct {
	allocs uint64 // remaining number of allocations
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

/* ===== */

func (a *Allocator) NewRunesValue(r []rune) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Runes{}
	o.Set(r)
	return core.RunesValue(o), nil
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

func (a *Allocator) NewArrayValue(arr []core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Array{}
	o.Set(arr)
	return core.ArrayValue(o, immutable), nil
}

func (a *Allocator) NewMapValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Map{}
	o.Set(m)
	return core.MapValue(o, immutable), nil
}

func (a *Allocator) NewRecordValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	a.allocs--
	if a.allocs == 0 {
		return core.Undefined, errs.ErrObjectAllocLimit
	}
	o := &core.Map{}
	o.Set(m)
	return core.RecordValue(o, immutable), nil
}

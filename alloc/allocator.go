package alloc

import (
	"github.com/jokruger/kavun/core"
)

type Allocator struct {
}

func New() core.Allocator {
	return &Allocator{}
}

/* ===== */

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

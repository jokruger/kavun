package alloc

import "github.com/jokruger/kavun/core"

func (a *Allocator) NewArray(capacity int) ([]core.Value, error) {
	o := make([]core.Value, 0, capacity)
	return o, nil
}

func (a *Allocator) ReleaseArray(v []core.Value) {
}

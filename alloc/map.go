package alloc

import "github.com/jokruger/kavun/core"

func (a *Allocator) NewMap(capacity int) (map[string]core.Value, error) {
	o := make(map[string]core.Value, capacity)
	return o, nil
}

func (a *Allocator) ReleaseMap(v map[string]core.Value) {
}

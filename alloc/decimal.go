package alloc

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func (a *Allocator) NewDecimal() (*core.Decimal, error) {
	a.allocs--
	if a.allocs == 0 {
		return nil, errs.ErrObjectAllocLimit
	}
	o := &core.Decimal{}
	return o, nil
}

func (a *Allocator) ReleaseDecimal(d *core.Decimal) {
	a.allocs++
}

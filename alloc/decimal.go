package alloc

import (
	"github.com/jokruger/kavun/core"
)

func (a *Allocator) NewDecimal() (*core.Decimal, error) {
	o := &core.Decimal{}
	return o, nil
}

func (a *Allocator) ReleaseDecimal(d *core.Decimal) {
}

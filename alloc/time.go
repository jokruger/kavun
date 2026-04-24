package alloc

import (
	"time"

	"github.com/jokruger/kavun/errs"
)

func (a *Allocator) NewTime() (*time.Time, error) {
	a.allocs--
	if a.allocs == 0 {
		return nil, errs.ErrObjectAllocLimit
	}
	o := &time.Time{}
	return o, nil
}

func (a *Allocator) ReleaseTime(t *time.Time) {
	a.allocs++
}

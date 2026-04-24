package alloc

import (
	"time"
)

func (a *Allocator) NewTime() (*time.Time, error) {
	o := &time.Time{}
	return o, nil
}

func (a *Allocator) ReleaseTime(t *time.Time) {
}

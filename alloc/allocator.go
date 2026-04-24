package alloc

import (
	"github.com/jokruger/kavun/core"
)

type Allocator struct {
}

func New() core.Allocator {
	return &Allocator{}
}

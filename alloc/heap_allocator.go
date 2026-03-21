package alloc

import "github.com/jokruger/gs/core"

type HeapAllocator struct {
}

func NewHeapAllocator() core.Allocator {
	return &HeapAllocator{}
}

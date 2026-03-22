package stdlib

import (
	"github.com/jokruger/gs/core"
)

func wrapError(vm core.VM, err error) core.Object {
	alloc := vm.Allocator()
	if err == nil {
		return alloc.NewBool(true)
	}
	return alloc.NewError(alloc.NewString(err.Error()))
}

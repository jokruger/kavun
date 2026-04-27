package stdlib

import (
	"github.com/jokruger/kavun/core"
)

// wrapError converts a Go error into a Kavun error value.
// If the error is nil, it returns a boolean true value (many stdlib functions expected to return True if no errors occurred).
func wrapError(vm core.VM, err error) (core.Value, error) {
	if err == nil {
		return core.True, nil
	}
	alloc := vm.Allocator()
	payload := alloc.NewStringValue(err.Error())
	return alloc.NewErrorValue(payload), nil
}

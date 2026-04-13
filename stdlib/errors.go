package stdlib

import (
	"github.com/jokruger/gs/core"
)

// wrapError converts a Go error into a GS error value.
// If the error is nil, it returns a boolean true value (many stdlib functions expected to return True if no errors occurred).
func wrapError(vm core.VM, err error) core.Value {
	if err == nil {
		return core.True
	}
	alloc := vm.Allocator()
	payload := alloc.NewStringValue(err.Error())
	return alloc.NewErrorValue(payload)
}

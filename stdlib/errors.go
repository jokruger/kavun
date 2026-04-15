package stdlib

import (
	"github.com/jokruger/gs/core"
)

// wrapError converts a Go error into a GS error value.
// If the error is nil, it returns a boolean true value (many stdlib functions expected to return True if no errors occurred).
func wrapError(vm core.VM, err error) (core.Value, error) {
	if err == nil {
		return core.True, nil
	}
	alloc := vm.Allocator()
	payload, err := alloc.NewStringValue(err.Error())
	if err != nil {
		return core.Undefined, err
	}
	return alloc.NewErrorValue(payload)
}

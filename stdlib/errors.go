package stdlib

import (
	"github.com/jokruger/kavun/core"
)

// wrapError converts a Go error into a Kavun error value.
// If error is nil, it returns a boolean true (many stdlib functions expected to return True if no errors occurred).
func wrapError(a *core.Arena, err error) (core.Value, error) {
	if err == nil {
		return core.True, nil
	}
	return core.NewErrorValue(a.NewStringValue(err.Error())), nil
}

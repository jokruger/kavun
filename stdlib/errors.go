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
	nv, err := a.NewStringValue(err.Error())
	if err != nil {
		return core.Undefined, err
	}
	return a.NewErrorValue(nv, core.KindUser, false)
}

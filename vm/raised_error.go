package vm

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// raisedError is the Go-level error used to propagate a raise() call through the VM. It carries a Kavun error value so
// the unwinder can hand it directly to recover() without re-wrapping.
type raisedError struct {
	val core.Value
	str string
	err error
}

func newRaisedError(a *core.Arena, v core.Value) error {
	var kind, str string
	var fatal bool
	if v.Type == value.Error {
		o := a.ResolveErrorValue(v)
		kind = o.Kind
		fatal = o.Fatal
		str, _ = o.Payload.AsString()
		if str == "" {
			str = o.Payload.String(a)
		}
	} else {
		str = fmt.Sprintf("error: %s", v.String(a))
	}

	return &raisedError{
		val: v,
		str: str,
		err: &errs.Error{
			Message:     str,
			Kind:        kind,
			Recoverable: !fatal,
		},
	}
}

func (r *raisedError) Error() string {
	return r.str
}

// Value exposes the underlying Kavun error value to the runtime.
func (r *raisedError) KavunValue() core.Value {
	return r.val
}

// Unwrap exposes an *errs.Error so errs.IsCritical can see the severity of a raise()d error. The fatality is taken
// from the boxed value.Error so a script-level error(payload, true) raised by the user is treated as Fatal and bypasses
// recover().
func (r *raisedError) Unwrap() error {
	return r.err
}

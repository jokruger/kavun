package vm

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// raisedError is the Go-level error used to propagate a raise() call through the VM. It carries a Kavun error value so
// the unwinder can hand it directly to recover() without re-wrapping.
type raisedError struct {
	value core.Value
}

func (r *raisedError) Error() string {
	if r.value.Type != core.VT_ERROR {
		return "error"
	}
	o := (*core.Error)(r.value.Ptr)
	if s, ok := o.Payload.AsString(); ok && s != "" {
		return s
	}
	return o.Payload.String()
}

// Value exposes the underlying Kavun error value to the runtime.
func (r *raisedError) KavunValue() core.Value {
	return r.value
}

// Unwrap exposes an *errs.Error so errs.IsCritical can see the severity of a raise()d error. The fatality is taken
// from the boxed core.Error so a script-level error(payload, true) raised by the user is treated as Fatal and bypasses
// recover().
func (r *raisedError) Unwrap() error {
	kind := ""
	fatal := false
	msg := r.Error()
	if r.value.Type == core.VT_ERROR {
		o := (*core.Error)(r.value.Ptr)
		kind = o.Kind
		fatal = o.Fatal
	}
	return &errs.Error{
		Message:     msg,
		Kind:        kind,
		Recoverable: !fatal,
	}
}

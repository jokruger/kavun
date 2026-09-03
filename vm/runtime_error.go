package vm

import (
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// RuntimeError is what Run, RunContext and Eval answer when a script fails at run time. It carries the classification
// the host needs — kind, severity, payload, stack trace — so a caller never has to parse the message.
//
// Error() renders the stable text "Runtime Error: <kind>: <message>" followed by one "\n\tat file:line:col" line per
// frame, innermost first. Unwrap() exposes the underlying cause, so errors.Is(err, errs.ErrStackOverflow),
// errs.AsError(err) and errs.IsCritical(err) all keep working on the returned error.
//
// Compile errors are NOT RuntimeErrors — they come from Script.Compile and have their own type.
type RuntimeError struct {
	// Kind is the stable tag: an errs.Kind* constant for a runtime failure, core.KindUser for a script's own
	// error(...)/raise(...), or "" if the failure carried no kind at all.
	Kind string

	// Category is where the error came from: a builtin/member/module (Runtime), the script's own raise (User) or
	// require (Requirement), or the VM itself (System). One kind, one category.
	Category errs.Category

	// Fatal reports that the error bypassed every recover() and stopped the VM. A fatal error means the script was
	// stopped; a recoverable one means the script simply did not handle it.
	Fatal bool

	// Message is the human-readable detail, without the kind prefix. For a raise(x) it is the rendering of x.
	Message string

	// Payload is the Kavun error value's payload: the message string for a runtime error, and whatever the script
	// passed for raise(x) / require(cond, x) — a dict, a record, any value.
	Payload core.Value

	// Trace lists the source positions of the active frames, innermost first.
	Trace []ast.SourceFilePos

	// cause is the error as the VM produced it, kept so the Unwrap chain reaches the *errs.Error underneath.
	cause error
}

func (e *RuntimeError) Error() string {
	var b strings.Builder
	b.WriteString("Runtime Error: ")
	b.WriteString(e.cause.Error())
	for _, p := range e.Trace {
		b.WriteString("\n\tat ")
		b.WriteString(p.String())
	}
	return b.String()
}

// Unwrap answers the cause as the VM produced it. Its own chain ends in an *errs.Error, which is what makes
// errors.Is / errs.AsError / errs.IsCritical work through a RuntimeError.
func (e *RuntimeError) Unwrap() error {
	return e.cause
}

// newRuntimeError classifies err and collects the stack trace from the current frame chain. Like the frame walk it
// replaces, it consumes the chain (framesIndex is wound back to the main frame) — Run resets it before the next run.
func (v *VM) newRuntimeError(err error) *RuntimeError {
	e := &RuntimeError{cause: err}

	// makeVMErrorValue already normalises every error shape the VM can produce — a wrapped Kavun value, a raise()d
	// value, an *errs.Error, or a bare Go error — into one error value; read the classification off that.
	if ev := v.makeVMErrorValue(err); ev.Type == value.Error {
		o := (*core.Error)(ev.Ptr)
		e.Kind = o.Kind
		e.Category = o.Category
		e.Fatal = o.Fatal
		e.Payload = o.Payload
		if s, ok := o.Payload.AsString(); ok {
			e.Message = s
		} else {
			e.Message = o.Payload.String()
		}
	} else {
		e.Category = errs.CategorySystem
		e.Payload = core.Undefined
		e.Message = err.Error()
	}

	e.Trace = append(e.Trace, v.fileSet.Position(v.curFrame.fn.SourcePos(v.ip-1)))
	for v.framesIndex > 1 {
		v.framesIndex--
		v.curFrame = &v.frames[v.framesIndex-1]
		e.Trace = append(e.Trace, v.fileSet.Position(v.curFrame.fn.SourcePos(v.curFrame.ip-1)))
	}
	return e
}

package vm

import (
	"errors"
	"slices"
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

	// Trace lists the source positions of the frames the error passed through, innermost first — a callback's own
	// frames included, ahead of the call that ran it.
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

// newRuntimeError classifies err and collects the stack trace: the frames of the callbacks it was raised in, then the
// current frame chain. It consumes the chain (framesIndex is wound back to the main frame) — Run resets it before
// the next run.
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

	// frames a callback had when the error left VM.Call come first: they were innermost, and are gone from the chain
	if ct := calleeTraceOf(err); ct != nil {
		e.Trace = append(e.Trace, ct.trace...)
	}
	e.Trace = v.appendFrameTrace(e.Trace, 0)
	v.framesIndex = 1
	v.curFrame = &v.frames[0]
	return e
}

// appendFrameTrace appends the source positions of the frames from the current one down to frames[bottom], innermost
// first, skipping the synthetic trampoline frames. It leaves the frame chain as it is.
func (v *VM) appendFrameTrace(trace []ast.SourceFilePos, bottom int) []ast.SourceFilePos {
	trace = append(trace, v.fileSet.Position(v.curFrame.fn.SourcePos(v.ip)))
	for i := v.framesIndex - 2; i >= bottom; i-- {
		if f := &v.frames[i]; f.fn != callbackTrampolineFn {
			trace = append(trace, v.fileSet.Position(f.fn.SourcePos(f.ip)))
		}
	}
	return trace
}

// calleeTraceError is an error that left VM.Call, carrying the source positions of the callee's frames, innermost
// first: VM.Call restores the caller's frame chain before it returns, so the frames the error was raised in are gone
// by the time it escapes the script. A nested callback's frames accumulate (the outermost wrapper holds them all).
// It renders and unwraps as the error itself, so classification and errors.Is/As see through it.
type calleeTraceError struct {
	err   error
	trace []ast.SourceFilePos
}

func (e *calleeTraceError) Error() string { return e.err.Error() }

func (e *calleeTraceError) Unwrap() error { return e.err }

// withCalleeTrace wraps err, which is leaving VM.Call, with the positions of the frames above frames[bottom] (the
// callee's), after those of any callback deeper down that err already carries.
func (v *VM) withCalleeTrace(err error, bottom int) error {
	var trace []ast.SourceFilePos
	if ct := calleeTraceOf(err); ct != nil {
		trace = slices.Clone(ct.trace)
	}
	return &calleeTraceError{err: err, trace: v.appendFrameTrace(trace, bottom)}
}

// calleeTraceOf answers the callee trace err carries, or nil.
func calleeTraceOf(err error) *calleeTraceError {
	ct, _ := errors.AsType[*calleeTraceError](err)
	return ct
}

// keepCalleeTrace answers err — a Kavun error value turned back into a Go error by the unwinder — with the callee
// trace of the error it was made from, as long as it is still that error (from is the in-flight value it started
// as, ct the trace it carried); an error a defer raised in its place starts without one.
func keepCalleeTrace(err error, errVal, from core.Value, ct *calleeTraceError) error {
	if ct == nil || errVal.Ptr != from.Ptr {
		return err
	}
	return &calleeTraceError{err: err, trace: ct.trace}
}

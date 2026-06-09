package vm

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// runUntilSuspend repeatedly drives v.run() and the cooperative unwinder. It returns when either v.err is nil
// (clean OpSuspend) or v.err is set with an error that escapes past the trampoline frame at index stopAt.
//
// stopAt is the index of the frame above which the runner is allowed to unwind. Frames at or below stopAt are not
// touched; an error reaching stopAt is reported back to the caller (via v.err remaining set).
func (v *VM) runUntilSuspend(stopAt int) {
	for {
		v.run()
		if v.err == nil {
			return
		}
		if errs.IsCritical(v.err) {
			return
		}
		if !v.tryRecover(stopAt) {
			return
		}
	}
}

// tryRecover walks frames above stopAt (top-down), running their deferred calls in LIFO order. If some defer recovers
// (clears the in-flight error), the surrounding function exits with its named-result value (or undefined), the VM state
// is set up to resume in the caller, and tryRecover returns true with v.err cleared. If no defer recovers, the frame
// chain is left INTACT (frames are not popped) so the surrounding error reporter can still produce a useful stack
// trace, and v.err is set to the propagated error.
func (v *VM) tryRecover(stopAt int) bool {
	errVal := v.makeVMErrorValue(v.err)
	v.err = nil

	// Walk frames from innermost to outermost without popping.
	for idx := v.framesIndex - 1; idx >= stopAt; idx-- {
		f := &v.frames[idx]
		if f.fn == callbackTrampolineFn {
			// Trampoline boundary — propagate to the host caller.
			v.err = unwrapKavunError(v.alloc, errVal)
			return false
		}
		if len(f.defers) == 0 {
			continue
		}
		f.inFlightErr = errVal
		// Run f's defers on its behalf. We don't move v.curFrame/v.curInsts/v.ip here: deferred bodies don't access
		// f's locals directly — they access them through closure free vars (boxed *core.Value), and invokeDeferred
		// sets up its own frame for the deferred body and restores dispatch state on exit.
		for len(f.defers) > 0 {
			d := f.defers[len(f.defers)-1]
			f.defers = f.defers[:len(f.defers)-1]
			v.invokeDeferred(f, d)
			if v.err != nil {
				if errs.IsCritical(v.err) {
					return false
				}
				f.inFlightErr = v.makeVMErrorValue(v.err)
				v.err = nil
			}
		}
		if f.inFlightErr.Type == core.VT_UNDEFINED {
			// Recovered. Simulate normal return from f, popping all frames between f and the (now-current) top.
			res := core.Undefined
			if f.fn.HasNamedResult() {
				res = v.readNamedResult(f)
			}
			v.unwindToFrameAndReturn(idx, res)
			return true
		}
		// Update propagating error and continue outward.
		errVal = f.inFlightErr
		f.inFlightErr = core.Undefined
	}

	v.err = unwrapKavunError(v.alloc, errVal)
	return false
}

// unwindToFrameAndReturn pops all frames above and including frameIdx, resetting their per-call state, and resumes
// execution in the caller of frameIdx with `res` placed in the callee result slot. Mirrors the OpReturn handler's tail
// logic.
func (v *VM) unwindToFrameAndReturn(frameIdx int, res core.Value) {
	target := &v.frames[frameIdx]
	bp := target.basePointer
	// Clear state on all popped frames.
	for i := frameIdx; i < v.framesIndex; i++ {
		v.frames[i].defers = nil
		v.frames[i].inFlightErr = core.Undefined
		v.frames[i].deferredFor = nil
	}
	v.framesIndex = frameIdx
	v.curFrame = &v.frames[v.framesIndex-1]
	v.curInsts = v.curFrame.fn.Instructions
	v.ip = v.curFrame.ip
	v.sp = bp
	v.stack[v.sp-1] = res
}

// invokeDeferred runs a single deferred call belonging to owner.
// It pushes a synthetic trampoline frame (so the deferred's OpReturn suspends back into Go) plus the deferred call's
// frame, then runs the VM with cooperative unwinding bounded by the trampoline. On exit, the trampoline + any remaining
// sub-frames are unwound and v.err reflects any error that escaped the deferred subtree.
func (v *VM) invokeDeferred(owner *frame, d deferred) {
	// Method-call form: dispatch directly. The receiver's type method table is not required to produce a Kavun-level
	// frame, so any recover() inside is meaningless; this matches Go's "recover only in a deferred function" rule
	// (here: only in deferred function values, not deferred method calls).
	if d.method != "" {
		_, err := d.fn.MethodCall(v.alloc, v, d.method, d.args)
		if err != nil {
			v.err = err
		}
		return
	}

	callee := d.fn
	args := d.args
	numArgs := len(args)

	// Snapshot dispatch state so we can restore on every exit path.
	savedIp := v.ip
	savedSp := v.sp
	savedCurInsts := v.curInsts
	savedCurFrame := v.curFrame
	savedFramesIndex := v.framesIndex

	// Builtin and other non-compiled callables run directly without a new frame; their result is discarded.
	// recover() inside such a builtin would not work (it requires deferredFor on a real frame), but builtins typically
	// can't reach script-level recover anyway.
	switch callee.Type {
	case core.VT_COMPILED_FUNCTION:
		// fall through to the framed path
	case core.VT_BUILTIN_FUNCTION:
		_, err := core.BuiltinFunctions[callee.Data].Func(v.alloc, v, args)
		if err != nil {
			v.err = err
		}
		return
	case core.VT_BUILTIN_CLOSURE:
		_, err := v.alloc.ResolveBuiltinClosureValue(callee).Func(v.alloc, v, args)
		if err != nil {
			v.err = err
		}
		return
	default:
		_, err := callee.Call(v.alloc, v, args)
		if err != nil {
			v.err = err
		}
		return
	}

	cfn := v.alloc.ResolveCompiledFunctionValue(callee)

	// Capacity checks.
	if v.framesIndex+2 > len(v.frames) {
		v.err = errs.ErrStackOverflow
		return
	}
	if v.sp+1+numArgs > len(v.stack) {
		v.err = errs.ErrStackOverflow
		return
	}

	// Push trampoline frame so deferred's OpReturn cleanly exits run().
	tf := &v.frames[v.framesIndex]
	tf.ip = -1
	tf.basePointer = v.sp
	tf.fn = callbackTrampolineFn
	tf.freeVars = nil
	tf.defers = nil
	tf.inFlightErr = core.Undefined
	tf.deferredFor = nil
	trampolineIdx := v.framesIndex
	v.framesIndex++

	// Push callee + args (matches OpCall layout: callee slot, then args).
	v.stack[v.sp] = callee
	v.sp++
	for _, a := range args {
		v.stack[v.sp] = a
		v.sp++
	}

	// Roll up variadic params if needed.
	if cfn.VarArgs {
		realArgs := int(cfn.NumParameters) - 1
		varArgsLen := numArgs - realArgs
		if varArgsLen >= 0 {
			arr := v.alloc.NewArray(varArgsLen, true)
			spStart := v.sp - varArgsLen
			for i := spStart; i < v.sp; i++ {
				arr[i-spStart] = v.stack[i]
			}
			v.stack[spStart] = v.alloc.NewArrayValue(arr, true)
			v.sp = spStart + 1
			numArgs = realArgs + 1
		}
	}
	if numArgs != int(cfn.NumParameters) {
		v.err = errs.NewWrongNumArgumentsError("defer", fmt.Sprintf("%d", cfn.NumParameters), numArgs)
		// roll back trampoline + stack
		v.framesIndex = savedFramesIndex
		v.sp = savedSp
		v.ip = savedIp
		v.curInsts = savedCurInsts
		v.curFrame = savedCurFrame
		return
	}

	// Push deferred function's frame, marking it as running on behalf of `owner` so OpRecover can find the in-flight
	// error.
	df := &v.frames[v.framesIndex]
	df.ip = -1
	df.basePointer = v.sp - numArgs
	df.fn = cfn
	df.freeVars = cfn.Free
	df.defers = nil
	df.inFlightErr = core.Undefined
	df.deferredFor = owner
	v.curFrame = df
	v.curInsts = cfn.Instructions
	v.ip = -1
	v.framesIndex++
	v.sp = v.sp - numArgs + cfn.NumLocals

	// Run, allowing cooperative unwinding inside the deferred subtree (everything above the trampoline).
	// Errors that escape past the trampoline land in v.err and we report them to the outer unwinder.
	v.runUntilSuspend(trampolineIdx)

	// Restore the outer dispatch state regardless of how the inner run exited.
	// Trampoline + any remaining frames above are dropped.
	v.framesIndex = savedFramesIndex
	v.sp = savedSp
	v.ip = savedIp
	v.curInsts = savedCurInsts
	v.curFrame = savedCurFrame

	// Reset the trampoline slot for cleanliness.
	tf.fn = nil
	tf.freeVars = nil
	tf.defers = nil
	tf.inFlightErr = core.Undefined
	tf.deferredFor = nil
}

// runFrameDefers executes f's defers in LIFO order at normal return time. Mirrors the unwind path but starts with
// f.inFlightErr cleared. If a deferred raises an error (and isn't recovered by a later defer), the new error is
// returned via v.err so the OpReturn handler can route the frame into the unwind path instead of normal return.
func (v *VM) runFrameDefers(f *frame) {
	for len(f.defers) > 0 {
		d := f.defers[len(f.defers)-1]
		f.defers = f.defers[:len(f.defers)-1]
		v.invokeDeferred(f, d)
		if v.err != nil {
			if errs.IsCritical(v.err) {
				return
			}
			f.inFlightErr = v.makeVMErrorValue(v.err)
			v.err = nil
		}
	}
}

// readNamedResult returns the named-result value of frame f, dereferencing the value-pointer indirection that
// GetLocalPtr installs when the slot is captured by a closure (e.g. when a deferred function assigns to res).
func (v *VM) readNamedResult(f *frame) core.Value {
	if !f.fn.HasNamedResult() {
		return core.Undefined
	}
	val := v.stack[f.basePointer+f.fn.NamedResultSlot()]
	if val.Type == core.VT_VALUE_PTR {
		return *(*core.Value)(val.Ptr)
	}
	return val
}

// writeNamedResult writes val into frame f's named-result slot, going through the value-pointer indirection if the
// slot has been captured by a closure (so deferred functions observing the slot by name see the update).
func (v *VM) writeNamedResult(f *frame, val core.Value) {
	if !f.fn.HasNamedResult() {
		return
	}
	sp := f.basePointer + f.fn.NamedResultSlot()
	if v.stack[sp].Type == core.VT_VALUE_PTR {
		(*core.Value)(v.stack[sp].Ptr).Set(val)
		return
	}
	v.stack[sp] = val
}

// makeVMErrorValue converts a Go error into a Kavun error value.
// Reads Kind and Message from the source *errs.Error; if the error doesn't implement *errs.Error (shouldn't happen for
// recoverable errors but possible for legacy inline errors) we fall back to an empty kind and use err.Error() as the
// message body.
func (v *VM) makeVMErrorValue(err error) core.Value {
	// If the error is already a Kavun-wrapped error, just unwrap it.
	if w, ok := err.(*kavunErrorWrap); ok {
		return w.val
	}
	// raise() bubbles a user-origin Kavun error directly without re-wrapping its payload.
	type raisedErrorIface interface{ KavunValue() core.Value }
	if r, ok := err.(raisedErrorIface); ok {
		return r.KavunValue()
	}
	kind := ""
	fatal := false
	msg := err.Error()
	if e := errs.AsError(err); e != nil {
		kind = e.Kind
		fatal = !e.Recoverable
		msg = e.Message
	}
	return v.alloc.NewRuntimeErrorValue(kind, fatal, msg)
}

// kavunErrorWrap carries a Kavun error value through the Go-error channel so that propagation across frames preserves
// user-visible payload and metadata.
type kavunErrorWrap struct {
	val core.Value
	str string
	err error
}

// unwrapKavunError converts a Kavun error value back into a Go error.
func unwrapKavunError(a *core.Arena, v core.Value) error {
	if v.Type != core.VT_ERROR {
		return fmt.Errorf("error: %s", v.String(a))
	}

	// Reproduce the *errs.Error "kind: message" formatting so that runtime errors flowing back to the host
	// (via formatRuntimeError) keep the stable display form scripts and tests expect.
	var str string
	o := (*core.Error)(v.Ptr)
	if s, ok := o.Payload.AsString(a); ok {
		str = s
	} else if o.Payload.Type != core.VT_UNDEFINED {
		str = o.Payload.String(a)
	}
	msg := str
	if o.Kind != "" && o.Kind != core.KindUser {
		if str == "" {
			str = o.Kind
		}
		str = o.Kind + ": " + str
	}

	return &kavunErrorWrap{
		val: v,
		str: str,
		err: &errs.Error{
			Kind:        o.Kind,
			Recoverable: !o.Fatal,
			Message:     msg,
		},
	}
}

func (w *kavunErrorWrap) Error() string {
	return w.str
}

// Unwrap re-creates an *errs.Error from the wrapped Kavun error value so that errors.Is(hostErr, errs.ErrXxx) keeps
// working at the host boundary. Recoverability is derived directly from the boxed core.Error's Fatal flag so a fatal
// error round-tripping through this path is still reported as fatal.
func (w *kavunErrorWrap) Unwrap() error {
	return w.err
}

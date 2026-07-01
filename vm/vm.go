package vm

import (
	"fmt"
	"math"
	"sync/atomic"

	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
)

var (
	callbackTrampolineInstructions = bc.Instructions{bc.Instruction{Op: bc.Suspend}}
	callbackTrampolineFn           = &core.CompiledFunction{Instructions: callbackTrampolineInstructions[:]}
)

// deferred is a queued deferred call captured by OpDefer/OpDeferMethod. Arguments are evaluated at defer time and
// stored here; the call itself runs when the surrounding function exits (LIFO order).
// When method is empty, fn is called as a regular function with args.
// When method is non-empty, fn is the receiver value and method names the member function to invoke with args.
type deferred struct {
	fn     core.Value
	args   []core.Value
	method string
}

// frame represents a function call frame.
type frame struct {
	// Hot scalar fields first: read and written on every instruction fetch and return.
	ip          int // instruction pointer within fn.Instructions; -1 means "before first instruction"
	basePointer int // index into VM stack where this frame's locals start

	// Pointer fields: accessed on closure captures and function entry/exit.
	fn       *core.CompiledFunction // the function being executed
	freeVars []*core.Value          // captured free variables for closures

	// Deferred-call queue (LIFO). Nil unless OpDefer was executed.
	defers []deferred

	// inFlightErr is the error currently propagating through this frame. Set by the unwinder when this frame is on the
	// unwind path; read and cleared by recover() called from a deferred function.
	// Undefined when no error is in flight.
	inFlightErr core.Value

	// deferredFor, if non-nil, points to the frame that owns this defer. Set when this frame is invoked as a deferred
	// call. Used by OpRecover to identify the "owner" frame whose in-flight error it should clear.
	// Per Go semantics, recover() works only when called directly from a deferred function — other functions called
	// from the deferred do not inherit deferredFor.
	deferredFor *frame
}

// VM is a virtual machine that executes the bytecode compiled by Compiler.
// VM must be used in a single-threaded context only.
type VM struct {
	// Dispatch state
	ip       int             // instruction pointer into curInsts
	sp       int             // stack pointer (index of next free slot)
	curInsts bc.Instructions // instructions of the current frame
	curFrame *frame          // frame currently being executed

	// Runtime state
	static      *core.Static // static data from bytecode
	globals     []core.Value // global variable storage used by global load/store/select opcodes
	frames      []frame      // call frame stack
	stack       []core.Value // operand stack
	framesIndex int          // number of active frames; updated on calls, returns, and synthetic callback frames

	// Cold diagnostic state: only used when execution aborts or a stack trace is formatted.
	fileSet *parser.SourceFileSet // source positions for runtime stack traces
	err     error                 // last runtime error captured by run()

	// concurrent state
	abort int64 // flag for aborting execution
}

// NewVM creates a VM.
func NewVM(maxFrames int, maxStack int) *VM {
	if maxFrames <= 0 {
		maxFrames = DefaultMaxFrames
	}

	if maxStack <= 0 {
		maxStack = DefaultStackSize
	}

	return &VM{
		frames: make([]frame, maxFrames),
		stack:  make([]core.Value, maxStack),
	}
}

// Reset resets the VM state to run new main function.
func (v *VM) Reset(bytecode *Bytecode, globals []core.Value) {
	if globals == nil {
		globals = make([]core.Value, GlobalsSize)
	}

	v.ip = -1
	v.sp = 0
	atomic.StoreInt64(&v.abort, 0)
	v.static = &bytecode.Static
	v.globals = globals

	v.frames[0].fn = bytecode.MainFunction
	v.frames[0].ip = -1
	v.frames[0].basePointer = 0
	v.frames[0].defers = nil
	v.frames[0].inFlightErr = core.Undefined
	v.frames[0].deferredFor = nil
	v.frames[0].freeVars = nil
	v.curFrame = &v.frames[0]
	v.curInsts = v.curFrame.fn.Instructions
	v.framesIndex = 1

	v.fileSet = bytecode.FileSet
	v.err = nil
}

// Clear drops the VM's Go references to bytecode, globals, and per-frame state so the Go garbage collector can reclaim
// them. Clear is optional and only useful when the same VM is reused for multiple runs and you want to break Go
// references between runs to reduce live-heap pressure.
func (v *VM) Clear() {
	v.static = nil
	v.globals = nil
	v.curInsts = nil
	v.fileSet = nil
	v.err = nil
	for i := range v.frames {
		v.frames[i].fn = nil
		v.frames[i].freeVars = nil
		v.frames[i].defers = nil
		v.frames[i].inFlightErr = core.Undefined
		v.frames[i].deferredFor = nil
	}
}

// Abort aborts the execution.
func (v *VM) Abort() {
	atomic.StoreInt64(&v.abort, 1)
}

// IsStackEmpty tests if the stack is empty or not.
func (v *VM) IsStackEmpty() bool {
	return v.sp == 0
}

// Recover returns the in-flight error of the surrounding "deferred-for" frame (and clears it) so the surrounding
// function returns normally; otherwise Undefined.
//
// Effective only when called directly from a deferred script function. Concretely we require:
//   - v.curFrame is a real compiled Kavun function (not the trampoline / not nil), and
//   - v.curFrame.deferredFor is non-nil (i.e. the current frame was entered as a deferred call).
//
// Calling recover() one level deeper (through another Kavun function invocation) does not work because OpCall resets
// deferredFor on the new frame. Host builtins invoked as defers also do not flip deferredFor, so recover() from such
// a builtin returns Undefined as well. Matches Go's "recover only in a deferred function" rule.
func (v *VM) Recover() core.Value {
	if v.curFrame == nil || v.curFrame.deferredFor == nil {
		return core.Undefined
	}
	// Defensive: a real deferred-for frame is always running compiled bytecode (not the synthetic trampoline).
	if v.curFrame.fn == nil || v.curFrame.fn == callbackTrampolineFn {
		return core.Undefined
	}
	target := v.curFrame.deferredFor
	if target.inFlightErr.Type == value.Undefined {
		return core.Undefined
	}
	err := target.inFlightErr
	target.inFlightErr = core.Undefined
	return err
}

// initFrameLocals clears non-argument local slots when entering a frame.
// Stack slots are reused across calls; without this, DefineLocal may release stale values from prior frames.
func (v *VM) initFrameLocals(f *frame, numArgs int) {
	start := f.basePointer + numArgs
	end := f.basePointer + f.fn.NumLocals
	for i := start; i < end; i++ {
		v.stack[i] = core.Undefined
	}
}

// releaseFrameLocals releases every local slot of f and clears it to Undefined.
func (v *VM) releaseFrameLocals(f *frame) {
	start := f.basePointer
	end := f.basePointer + f.fn.NumLocals
	for i := start; i < end; i++ {
		v.stack[i] = core.Undefined
	}
}

// Call calls a compiled function with the given arguments and returns the result.
func (v *VM) Call(cfv core.Value, args []core.Value) (core.Value, error) {
	if cfv.Type != value.CompiledFunction {
		return core.Undefined, errs.NewInvalidArgumentTypeError("call", "function", "compiled function", cfv.TypeName())
	}
	fn := (*core.CompiledFunction)(cfv.Ptr)

	// Check argument count and roll up variadic args if needed
	numArgs := len(args)
	if fn.VarArgs {
		if numArgs < int(fn.NumParameters)-1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("call", fmt.Sprintf("at least %d", fn.NumParameters-1), numArgs)
		}
		realArgs := int(fn.NumParameters) - 1
		varArgs := numArgs - realArgs
		if varArgs >= 0 {
			newArgs := make([]core.Value, realArgs+1)
			copy(newArgs, args[:realArgs])
			newArgs[realArgs] = core.NewArrayValue(args[realArgs:], true)
			args = newArgs
			numArgs = realArgs + 1
		}
	} else if numArgs != int(fn.NumParameters) {
		return core.Undefined, errs.NewWrongNumArgumentsError("call", fmt.Sprintf("%d", fn.NumParameters), numArgs)
	}

	// Save current VM state
	savedIp := v.ip
	savedSp := v.sp
	savedCurInsts := v.curInsts
	savedCurFrame := v.curFrame
	savedFramesIndex := v.framesIndex
	savedErr := v.err

	// Clear error for fresh call
	v.err = nil

	// This helper consumes two frame slots: a synthetic trampoline frame and the callee frame.
	if v.framesIndex+1 >= len(v.frames) {
		v.err = errs.NewStackOverflowError("native callback frames")
		return core.Undefined, v.err
	}

	// fn.NumLocals always includes parameters (variadic tail collapsed into one array slot above), and numArgs has been
	// reshaped to equal fn.NumParameters by this point, so numArgs <= fn.NumLocals. The check below therefore also
	// covers the push phase (callee slot + args) and is safe for stack overflow check.
	if v.sp+1+fn.NumLocals+fn.MaxStack > len(v.stack) {
		v.err = errs.ErrStackOverflow
		return core.Undefined, v.err
	}

	// Create a synthetic trampoline frame that returns into OpSuspend.
	// The function object is immutable and shared; the per-call state lives in the frame.
	trampolineFrame := &v.frames[v.framesIndex]
	trampolineFrame.ip = -1
	trampolineFrame.basePointer = v.sp
	trampolineFrame.fn = callbackTrampolineFn
	trampolineFrame.freeVars = nil
	v.framesIndex++

	// Push callee slot (matches normal OpCall stack layout). This is where OpReturn will write the return value.
	v.stack[v.sp] = cfv
	v.sp++

	// Push arguments onto stack.
	for _, arg := range args {
		v.stack[v.sp] = arg
		v.sp++
	}

	// Set up callback frame (similar to OpCall for CompiledFunction)
	v.curFrame = &v.frames[v.framesIndex]
	v.curFrame.ip = -1
	v.curFrame.basePointer = v.sp - numArgs // Points to first arg (after callee slot)
	v.curFrame.fn = fn
	v.curFrame.freeVars = fn.Free
	v.curFrame.defers = nil
	v.curFrame.inFlightErr = core.Undefined
	v.curFrame.deferredFor = nil
	v.curInsts = fn.Instructions
	v.ip = -1
	v.framesIndex++
	v.initFrameLocals(v.curFrame, numArgs)
	v.sp = v.sp - numArgs + fn.NumLocals
	if fn.HasNamedResult() {
		v.stack[v.curFrame.basePointer+fn.NamedResultSlot()] = core.Undefined
	}

	// Execute the callback by calling run()
	// When callback returns (OpReturn), it will return to trampoline frame.
	// Trampoline executes OpSuspend, which exits run().
	// Cooperative unwinding: errors raised by the callback may be caught by deferred recover() inside the callee chain,
	// bounded by the trampoline frame at savedFramesIndex.
	trampolineIdx := savedFramesIndex // the trampoline frame's index
	v.runUntilSuspend(trampolineIdx)

	// Extract result before restoring state
	// OpReturn places the result at sp-1, which is the callee slot we reserved
	var result core.Value
	if v.err == nil {
		// The return value is at savedSp (the callee slot position)
		result = v.stack[savedSp]
	}

	// Restore VM state
	v.ip = savedIp
	v.sp = savedSp
	v.curInsts = savedCurInsts
	v.curFrame = savedCurFrame
	v.framesIndex = savedFramesIndex

	// Preserve error from callback, but restore if no error
	err := v.err
	v.err = savedErr

	return result, err
}

// Run starts the execution.
func (v *VM) Run() (err error) {
	// reset VM states
	v.sp = 0
	v.curFrame = &(v.frames[0])
	v.curInsts = v.curFrame.fn.Instructions
	v.framesIndex = 1
	v.ip = -1
	atomic.StoreInt64(&v.abort, 0)

	v.runUntilSuspend(1)
	if v.err == nil {
		return nil
	}
	return v.formatRuntimeError(v.err)
}

// formatRuntimeError annotates a runtime error with a stack trace built from the current frame chain.
// Used when an error escapes all defers.
func (v *VM) formatRuntimeError(err error) error {
	filePos := v.fileSet.Position(v.curFrame.fn.SourcePos(v.ip - 1))
	out := fmt.Errorf("Runtime Error: %w\n\tat %s", err, filePos)
	for v.framesIndex > 1 {
		v.framesIndex--
		v.curFrame = &v.frames[v.framesIndex-1]
		filePos = v.fileSet.Position(v.curFrame.fn.SourcePos(v.curFrame.ip - 1))
		out = fmt.Errorf("%w\n\tat %s", out, filePos)
	}
	return out
}

func (v *VM) run() {
	for {
		v.ip++
		switch v.curInsts[v.ip].Op {
		case bc.AbortCheck:
			if atomic.LoadInt64(&v.abort) != 0 {
				return
			}

		case bc.Suspend:
			return

		case bc.Return:
			hasResult := v.curInsts[v.ip].Op1 == 1
			var res core.Value // default is core.Undefined
			if hasResult {
				res = v.stack[v.sp-1]
				// Go parity: `return EXPR` in a function with a named result is sugar for
				// `<name> = EXPR; return`. Assigning to the named-result slot lets any deferred
				// function observe (and mutate) the returned value through the named result.
				if v.curFrame.fn.HasNamedResult() && len(v.curFrame.defers) > 0 {
					v.writeNamedResult(v.curFrame, res)
				}
			} else if v.curFrame.fn.HasNamedResult() {
				// Bare return: use the named-result slot if defined.
				res = v.readNamedResult(v.curFrame)
			}
			// Run any deferred calls before popping the frame.
			if len(v.curFrame.defers) > 0 {
				v.runFrameDefers(v.curFrame)
				if v.err != nil {
					// A critical error escaped the deferred subtree.
					return
				}
				if v.curFrame.inFlightErr.Type != value.Undefined {
					// A non-recovered error was raised by a defer. Hand off to the unwinder; clear the local in-flight
					// state since we're about to escape this frame.
					errVal := v.curFrame.inFlightErr
					v.curFrame.inFlightErr = core.Undefined
					v.err = unwrapKavunError(errVal)
					return
				}
				// Defers may have updated the named-result slot; re-read it (covers both bare return and `return EXPR`
				// since the latter wrote EXPR into the slot above).
				if v.curFrame.fn.HasNamedResult() {
					res = v.readNamedResult(v.curFrame)
				}
			}
			v.releaseFrameLocals(v.curFrame)
			v.framesIndex--
			v.frames[v.framesIndex].defers = nil
			v.frames[v.framesIndex].inFlightErr = core.Undefined
			v.frames[v.framesIndex].deferredFor = nil
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip
			v.sp = v.frames[v.framesIndex].basePointer
			v.stack[v.sp-1] = res

		case bc.Pop:
			v.sp--

		case bc.UnaryNeg:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Int: // fast track for integers
				v.stack[v.sp] = core.IntValue(-int64(l.Data))
				v.sp++
			case value.Float: // fast track for floats
				v.stack[v.sp] = core.FloatValue(-math.Float64frombits(l.Data))
				v.sp++
			default:
				res, err := l.UnaryOp(token.Sub)
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case bc.UnaryNot:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Bool: // fast track for booleans
				v.stack[v.sp] = core.BoolValue(l.Data == 0)
				v.sp++
			default:
				v.stack[v.sp] = core.BoolValue(!l.IsTrue())
				v.sp++
			}

		case bc.UnaryBitNot:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Int: // fast track for integer
				v.stack[v.sp] = core.IntValue(^int64(l.Data))
				v.sp++
			default:
				res, err := l.UnaryOp(token.Xor)
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case bc.Equal:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(l == r || l.Equal(r))
			v.sp++

		case bc.NotEqual:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(!(l == r || l.Equal(r)))
			v.sp++

		case bc.Contains:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.stack[v.sp-2] = core.BoolValue(r.Contains(l))
			v.sp--

		case bc.Immutable:
			val := v.stack[v.sp-1]
			t, err := val.ToImmutable()
			if err != nil {
				v.err = err
				return
			}
			// ToImmutable only flips the immutable flag; the slot keeps ownership of the same underlying ref.
			v.stack[v.sp-1] = t

		case bc.AccessIndex:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res, err := l.Access(n, bc.AccessIndex)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case bc.AccessSelector:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			val, err := l.Access(n, bc.AccessSelector)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case bc.Slice:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			l := v.stack[v.sp-3]
			v.sp -= 3
			res, err := l.Slice(low, high)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case bc.SliceStep:
			step := v.stack[v.sp-1]
			high := v.stack[v.sp-2]
			low := v.stack[v.sp-3]
			l := v.stack[v.sp-4]
			v.sp -= 4
			res, err := l.SliceStep(low, high, step)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case bc.IterInit:
			l := v.stack[v.sp-1]
			v.sp--
			if !l.IsIterable() {
				v.err = errs.NewNotIterableError(l.TypeName())
				return
			}
			it, err := l.Iterator()
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = it
			v.sp++

		case bc.IterNext:
			it := v.stack[v.sp-1]
			res := core.BoolValue(it.Next())
			v.stack[v.sp-1] = res

		case bc.IterKey:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Key()
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case bc.IterValue:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Value()
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case bc.FormatRuntimeSpec:
			specVal := v.stack[v.sp-1]
			val := v.stack[v.sp-2]
			v.sp -= 2
			if specVal.Type != value.String {
				v.err = errs.NewInvalidArgumentTypeError("f-string", "spec", "string", specVal.TypeName())
				return
			}
			specText := *(*string)(specVal.Ptr)
			parsed, err := fspec.Parse(specText)
			if err != nil {
				v.err = errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, fmt.Sprintf("f-string format spec %q: %v", specText, err))
				return
			}
			s, err := val.Format(parsed)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = core.NewStringValue(s)
			v.sp++

		case bc.FormatStaticSpec:
			fs := v.static.FormatSpecs[v.curInsts[v.ip].Op3]
			val := v.stack[v.sp-1]
			s, err := val.Format(fs.Spec)
			if err != nil {
				v.sp--
				v.err = err
				return
			}
			v.stack[v.sp-1] = core.NewStringValue(s)

		case bc.BinaryOp:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip].Op1)
			if l.Type == value.Int && r.Type == value.Int {
				li := int64(l.Data)
				ri := int64(r.Data)
				switch tok {
				case token.Add:
					v.stack[v.sp-2] = core.IntValue(li + ri)
					v.sp--
					continue
				case token.Sub:
					v.stack[v.sp-2] = core.IntValue(li - ri)
					v.sp--
					continue
				case token.Mul:
					v.stack[v.sp-2] = core.IntValue(li * ri)
					v.sp--
					continue
				case token.Quo:
					if ri == 0 {
						v.sp -= 2
						v.err = errs.ErrDivisionByZero
						return
					}
					v.stack[v.sp-2] = core.IntValue(li / ri)
					v.sp--
					continue
				case token.Rem:
					if ri == 0 {
						v.sp -= 2
						v.err = errs.ErrDivisionByZero
						return
					}
					v.stack[v.sp-2] = core.IntValue(li % ri)
					v.sp--
					continue
				case token.Less:
					v.stack[v.sp-2] = core.BoolValue(li < ri)
					v.sp--
					continue
				case token.Greater:
					v.stack[v.sp-2] = core.BoolValue(li > ri)
					v.sp--
					continue
				case token.LessEq:
					v.stack[v.sp-2] = core.BoolValue(li <= ri)
					v.sp--
					continue
				case token.GreaterEq:
					v.stack[v.sp-2] = core.BoolValue(li >= ri)
					v.sp--
					continue
				}
			}
			res, err := l.BinaryOp(tok, r)
			if err != nil {
				v.sp -= 2
				v.err = err
				return
			}
			v.stack[v.sp-2] = res
			v.sp--

		case bc.ImportBuiltinModule:
			m, err := stdlib.GetModule(int(v.curInsts[v.ip].Op3))
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = m
			v.sp++

		case bc.DefineLocal:
			v.sp--
			sp := v.curFrame.basePointer + int(v.curInsts[v.ip].Op3)
			v.stack[sp] = v.stack[v.sp] // move value from stack (sp is decremented)

		case bc.LoadLocal:
			e := v.stack[v.curFrame.basePointer+int(v.curInsts[v.ip].Op3)]
			if e.Type == value.ValuePtr {
				e = *(*core.Value)(e.Ptr)
			}
			v.stack[v.sp] = e // copy local value to stack
			v.sp++

		case bc.StoreLocal:
			sp := v.curFrame.basePointer + int(v.curInsts[v.ip].Op3)
			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			v.sp--
			val := v.stack[v.sp] // move value from stack to local slot (sp is decremented)
			if v.stack[sp].Type == value.ValuePtr {
				// if target slot is a free variable, update the pointee value so all referencing free variables can observe the change
				*(*core.Value)(v.stack[sp].Ptr) = val
			} else {
				v.stack[sp] = val // move val from local slot to stack
			}

		case bc.StoreIndexedLocal:
			localIndex := int(v.curInsts[v.ip].Op3)
			numSelectors := int(v.curInsts[v.ip].Op2)
			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			dst := v.stack[v.curFrame.basePointer+localIndex]
			if dst.Type == value.ValuePtr {
				dst = *(*core.Value)(dst.Ptr)
			}
			if e := v.indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case bc.LoadFree:
			v.stack[v.sp] = *v.curFrame.freeVars[v.curInsts[v.ip].Op3]
			v.sp++

		case bc.StoreFree:
			*v.curFrame.freeVars[v.curInsts[v.ip].Op3] = v.stack[v.sp-1] // move value from stack to free variable (sp is decremented)
			v.sp--

		case bc.StoreIndexedFree:
			freeIndex := int(v.curInsts[v.ip].Op3)
			numSelectors := int(v.curInsts[v.ip].Op2)
			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			if e := v.indexAssign(*v.curFrame.freeVars[freeIndex], val, selectors); e != nil {
				v.err = e
				return
			}

		case bc.LoadLocalPtr:
			sp := v.curFrame.basePointer + int(v.curInsts[v.ip].Op3)
			var freeVar *core.Value
			if v.stack[sp].Type == value.ValuePtr {
				freeVar = (*core.Value)(v.stack[sp].Ptr)
			} else {
				val := v.stack[sp]
				freeVar = &val
				v.stack[sp] = core.NewValuePtrValue(freeVar)
			}
			v.stack[v.sp] = core.NewValuePtrValue(freeVar)
			v.sp++

		case bc.LoadFreePtr:
			v.stack[v.sp] = core.NewValuePtrValue(v.curFrame.freeVars[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadBuiltinFunction:
			v.stack[v.sp] = core.BuiltinFunctionValue(uint64(v.curInsts[v.ip].Op3))
			v.sp++

		case bc.MakeClosure:
			numFree := int(v.curInsts[v.ip].Op2)
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				// Compiler guarantees each free-var operand is a ValuePtr pushed by GetLocalPtr/GetFreePtr.
				ptr := v.stack[v.sp-numFree+i]
				free[i] = (*core.Value)(ptr.Ptr)
			}
			v.sp -= numFree
			fn := v.static.CompiledFunctions[v.curInsts[v.ip].Op3]
			v.stack[v.sp] = core.NewCompiledFunctionValue(fn.Instructions, free, fn.SourceMap, fn.NumLocals, fn.MaxStack, fn.NumParameters, fn.NamedResult, fn.VarArgs)
			v.sp++

		case bc.LoadGlobal:
			v.stack[v.sp] = v.globals[v.curInsts[v.ip].Op3]
			v.sp++

		case bc.StoreGlobal:
			v.sp--
			v.globals[v.curInsts[v.ip].Op3] = v.stack[v.sp] // move value from stack to global (sp is decremented)

		case bc.StoreIndexedGlobal:
			numSelectors := int(v.curInsts[v.ip].Op2)
			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
			for i := range numSelectors {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			if e := v.indexAssign(v.globals[v.curInsts[v.ip].Op3], val, selectors); e != nil {
				v.err = e
				return
			}

		case bc.MakeArray:
			n := int(v.curInsts[v.ip].Op3)
			elements := make([]core.Value, 0, n)
			for i := v.sp - n; i < v.sp; i++ {
				elements = append(elements, v.stack[i])
			}
			v.sp -= n
			v.stack[v.sp] = core.NewArrayValue(elements, false)
			v.sp++

		case bc.MakeRecord:
			n := int(v.curInsts[v.ip].Op3)
			kv := make(map[string]core.Value, n)
			for i := v.sp - n; i < v.sp; i += 2 {
				l := v.stack[i]
				e := v.stack[i+1]
				switch l.Type {
				case value.String: // fast track for strings
					kv[*(*string)(l.Ptr)] = e
				default:
					key, ok := l.AsString()
					if !ok {
						v.err = errs.NewInvalidArgumentTypeError("record", "key", "string", v.stack[i].TypeName())
						return
					}
					kv[key] = e
				}
			}
			v.sp -= n
			v.stack[v.sp] = core.NewRecordValue(kv, false)
			v.sp++

		case bc.CallFunction:
			numArgs := int(v.curInsts[v.ip].Op2)
			val := v.stack[v.sp-1-numArgs]
			if val.Type != value.CompiledFunction && val.Type != value.BuiltinFunction && val.Type != value.BuiltinClosure && !val.IsCallable() {
				v.err = errs.NewNotCallableError(val.TypeName())
				return
			}

			if v.curInsts[v.ip].Op1 == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case value.Array:
					o := (*core.Array)(arg.Ptr)
					// Bounds-check before expansion: spread is the one OpCall case whose stack growth is data-driven
					// and cannot be modeled by the compile-time MaxStack analyzer.
					if v.sp+len(o.Elements) > len(v.stack) {
						v.err = errs.ErrStackOverflow
						return
					}
					for _, item := range o.Elements {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(o.Elements) - 1
				default:
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName())
					return
				}
			}

			switch val.Type {
			case value.CompiledFunction: // special case for compiled functions
				if atomic.LoadInt64(&v.abort) != 0 {
					return
				}
				callee := (*core.CompiledFunction)(val.Ptr)
				if callee.VarArgs {
					// if the closure is variadic, roll up all variadic parameters into an array
					realArgs := int(callee.NumParameters) - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := make([]core.Value, varArgs)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						v.stack[spStart] = core.NewArrayValue(args, true)
						v.sp = spStart + 1
					}
				}
				if numArgs != int(callee.NumParameters) {
					if callee.VarArgs {
						v.err = errs.NewWrongNumArgumentsError("call", fmt.Sprintf(">=%d", callee.NumParameters-1), numArgs)
					} else {
						v.err = errs.NewWrongNumArgumentsError("call", fmt.Sprintf("%d", callee.NumParameters), numArgs)
					}
					return
				}

				// test if it's tail-call
				// Note: tail-call optimization is unsafe when the current frame has registered defers, because reusing
				// the frame would skip running them. Skip the optimization in that case so OpReturn can run the defers
				// as usual.
				if callee == v.curFrame.fn && len(v.curFrame.defers) == 0 { // recursion
					nextOp := v.curInsts[v.ip+1].Op
					if nextOp == bc.Return || (nextOp == bc.Pop && v.curInsts[v.ip+2].Op == bc.Return) {
						// Move new args into the first numArgs local slots (ownership transfer).
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] = v.stack[v.sp-numArgs+p]
						}
						v.sp -= numArgs + 1
						v.ip = -1 // reset IP to beginning of the frame
						continue
					}
				}
				if v.framesIndex >= len(v.frames) {
					v.err = errs.ErrStackOverflow
					return
				}
				if v.sp-numArgs+callee.NumLocals+callee.MaxStack > len(v.stack) {
					v.err = errs.ErrStackOverflow
					return
				}

				// update call frame
				v.curFrame.ip = v.ip // store current ip before call
				v.curFrame = &v.frames[v.framesIndex]
				v.curFrame.fn = callee
				v.curFrame.freeVars = callee.Free
				v.curFrame.basePointer = v.sp - numArgs
				v.curFrame.defers = nil
				v.curFrame.inFlightErr = core.Undefined
				v.curFrame.deferredFor = nil
				v.curInsts = callee.Instructions
				v.ip = -1
				v.framesIndex++
				v.initFrameLocals(v.curFrame, numArgs)
				v.sp = v.sp - numArgs + callee.NumLocals
				if callee.HasNamedResult() {
					v.stack[v.curFrame.basePointer+callee.NamedResultSlot()] = core.Undefined
				}

			case value.BuiltinFunction: // fast track for built-in functions
				res, err := core.BuiltinFunctions[val.Data].Func(v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			case value.BuiltinClosure: // fast track for built-in closure
				res, err := (*core.BuiltinClosure)(val.Ptr).Func(v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			default:
				res, err := val.Call(v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case bc.CallMethod:
			numArgs := int(v.curInsts[v.ip].Op2)
			receiver := v.stack[v.sp-1-numArgs]
			if v.curInsts[v.ip].Op1 == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case value.Array:
					o := (*core.Array)(arg.Ptr)
					// Bounds-check before expansion (see OpCall for rationale).
					if v.sp+len(o.Elements) > len(v.stack) {
						v.err = errs.ErrStackOverflow
						return
					}
					for _, item := range o.Elements {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(o.Elements) - 1
				default:
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName())
					return
				}
				receiver = v.stack[v.sp-1-numArgs]
			}

			name := v.static.Strings[v.curInsts[v.ip].Op3]
			res, err := receiver.MethodCall(v, name, v.stack[v.sp-numArgs:v.sp])
			v.sp -= numArgs + 1
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case bc.Defer:
			numArgs := int(v.curInsts[v.ip].Op2)
			// Stack layout: [..., callee, arg1, ..., argN]
			argsStart := v.sp - numArgs
			calleeIdx := argsStart - 1
			callee := v.stack[calleeIdx]
			// Copy args out of the operand stack into a fresh slice so later stack operations cannot mutate them.
			var capturedArgs []core.Value
			if numArgs > 0 {
				capturedArgs = make([]core.Value, numArgs)
				copy(capturedArgs, v.stack[argsStart:v.sp])
			}
			v.curFrame.defers = append(v.curFrame.defers, deferred{fn: callee, args: capturedArgs})
			v.sp = calleeIdx

		case bc.DeferMethod:
			numArgs := int(v.curInsts[v.ip].Op2)
			methodName := v.static.Strings[v.curInsts[v.ip].Op3]
			argsStart := v.sp - numArgs
			recvIdx := argsStart - 1
			recv := v.stack[recvIdx]
			var capturedArgs []core.Value
			if numArgs > 0 {
				capturedArgs = make([]core.Value, numArgs)
				copy(capturedArgs, v.stack[argsStart:v.sp])
			}
			v.curFrame.defers = append(v.curFrame.defers, deferred{fn: recv, args: capturedArgs, method: methodName})
			v.sp = recvIdx

		case bc.Jump:
			v.ip = int(v.curInsts[v.ip].Op3) - 1

		case bc.JumpFalsy:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				}
			default:
				if !l.IsTrue() {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				}
			}

		case bc.AndJump:
			l := v.stack[v.sp-1]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				} else {
					v.sp--
				}
			default:
				if !l.IsTrue() {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				} else {
					v.sp--
				}
			}

		case bc.OrJump:
			l := v.stack[v.sp-1]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					v.sp--
				} else {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				}
			default:
				if !l.IsTrue() {
					v.sp--
				} else {
					v.ip = int(v.curInsts[v.ip].Op3) - 1
				}
			}

		case bc.PushUndefined:
			v.stack[v.sp] = core.Undefined
			v.sp++

		case bc.PushBool:
			v.stack[v.sp] = core.Value{Type: value.Bool, Immutable: true, Data: uint64(v.curInsts[v.ip].Op1)}
			v.sp++

		case bc.PushByte:
			v.stack[v.sp] = core.Value{Type: value.Int, Immutable: true, Data: uint64(v.curInsts[v.ip].Op1)}
			v.sp++

		case bc.PushRune:
			v.stack[v.sp] = core.Value{Type: value.Int, Immutable: true, Data: uint64(v.curInsts[v.ip].Op3)}
			v.sp++

		case bc.PushInt:
			v.stack[v.sp] = core.Value{Type: value.Int, Immutable: true, Data: uint64(int64(int32(v.curInsts[v.ip].Op3)))}
			v.sp++

		case bc.LoadStaticDecimal:
			v.stack[v.sp] = core.NewStaticDecimalValue(&v.static.Decimals[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticString:
			v.stack[v.sp] = core.NewStaticStringValue(&v.static.Strings[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticRunes:
			v.stack[v.sp] = core.NewStaticRunesValue(&v.static.Runes[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticBytes:
			v.stack[v.sp] = core.NewStaticBytesValue(&v.static.Bytes[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticTime:
			v.stack[v.sp] = core.NewStaticTimeValue(&v.static.Times[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticFormatSpec:
			v.stack[v.sp] = core.NewStaticFormatSpecValue(&v.static.FormatSpecs[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticCompiledFunction:
			v.stack[v.sp] = core.NewStaticCompiledFunctionValue(&v.static.CompiledFunctions[v.curInsts[v.ip].Op3])
			v.sp++

		case bc.LoadStaticPrimitive:
			v.stack[v.sp] = v.static.Primitives[v.curInsts[v.ip].Op3].Value()
			v.sp++

		default:
			v.err = errs.NewInternalError(fmt.Sprintf("unknown opcode: %d", v.curInsts[v.ip]))
			return
		}
	}
}

func (v *VM) indexAssign(dst, src core.Value, selectors []core.Value) error {
	numSel := len(selectors)
	for si := numSel - 1; si > 0; si-- {
		next, err := dst.Access(selectors[si], bc.AccessIndex)
		if err != nil {
			return err
		}
		dst = next
	}
	return dst.Assign(selectors[0], src)
}

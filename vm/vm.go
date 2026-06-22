package vm

import (
	"fmt"
	"math"
	"sync/atomic"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
)

var (
	callbackTrampolineInstructions = [...]byte{opcode.Suspend.Byte()}
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
	ip       int    // instruction pointer into curInsts
	sp       int    // stack pointer (index of next free slot)
	curInsts []byte // instructions of the current frame
	curFrame *frame // frame currently being executed
	abort    int64  // flag for aborting execution

	// Runtime state
	static      *core.Static // static data from bytecode
	globals     []core.Value // global variable storage used by global load/store/select opcodes
	frames      []frame      // call frame stack
	stack       []core.Value // operand stack
	framesIndex int          // number of active frames; updated on calls, returns, and synthetic callback frames

	// Cold diagnostic state: only used when execution aborts or a stack trace is formatted.
	fileSet *parser.SourceFileSet // source positions for runtime stack traces
	err     error                 // last runtime error captured by run()
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

// Reset resets the VM state to run new main function. It also binds the bytecode to the arena.
func (v *VM) Reset(alloc *core.Arena, bytecode *Bytecode, globals []core.Value) {
	if globals == nil {
		globals = make([]core.Value, GlobalsSize)
	}

	bytecode.Bind(alloc) // bind bytecode to arena before running to resolve static values

	v.ip = -1
	v.sp = 0
	atomic.StoreInt64(&v.abort, 0)
	v.static = &bytecode.Static
	v.globals = globals
	v.alloc = alloc

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

// Clear drops the VM's Go references to bytecode, globals, arena, and per-frame state so the Go garbage collector can
// reclaim them. It does NOT Release any pooled refpool references — releasing pool refs is the arena's job (call
// arena.Reset for that). Clear is optional and only useful when the same VM is reused for multiple runs and you want
// to break Go references between runs to reduce live-heap pressure.
func (v *VM) Clear() {
	v.static = nil
	v.globals = nil
	v.alloc = nil
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

// releaseFrameLocals releases every local slot of f and clears it to Undefined. Called from OpReturn (and from the
// unwinder for skipped frames) to drop the +1 references the popped frame's locals own so their pool slots can be
// reused immediately rather than waiting for arena reset. Callers must Retain any value they intend to keep beyond
// this call (e.g. the function's return value when it points to a local) before invoking releaseFrameLocals.
func (v *VM) releaseFrameLocals(f *frame) {
	start := f.basePointer
	end := f.basePointer + f.fn.NumLocals
	for i := start; i < end; i++ {
		t := v.stack[i]
		if t.Type >= value.FirstArenaType && !t.Static {
			v.alloc.ReleaseAllocated(t)
		}
		v.stack[i] = core.Undefined
	}
}

// Call calls a compiled function with the given arguments and returns the result.
func (v *VM) Call(cfv core.Value, args []core.Value) (core.Value, error) {
	if cfv.Type != value.CompiledFunction {
		return core.Undefined, errs.NewInvalidArgumentTypeError("call", "function", "compiled function", cfv.TypeName(v.alloc))
	}
	fn := v.alloc.ResolveCompiledFunctionValue(cfv)

	// Check argument count and roll up variadic args if needed
	numArgs := len(args)
	if fn.VarArgs {
		if numArgs < int(fn.NumParameters)-1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("call", fmt.Sprintf("at least %d", fn.NumParameters-1), numArgs)
		}
		realArgs := int(fn.NumParameters) - 1
		varArgs := numArgs - realArgs
		if varArgs >= 0 {
			newArgs := v.alloc.NewArray(realArgs+1, true)
			copy(newArgs, args[:realArgs])
			nv, err := v.alloc.NewArrayValue(args[realArgs:], true)
			if err != nil {
				return core.Undefined, err
			}
			newArgs[realArgs] = nv
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

	// Push callee slot (matches normal OpCall stack layout)
	// This is where OpReturn will write the return value.
	// Retain so the stack slot becomes +1 owner; OpReturn will Release it before overwriting with the result.
	if cfv.Type >= value.FirstArenaType && !cfv.Static {
		v.alloc.RetainAllocated(cfv)
	}
	v.stack[v.sp] = cfv
	v.sp++

	// Push arguments onto stack. Args are borrowed from the host caller; the stack slots become +1 owners, so Retain
	// each so the eventual Release inside the callee (via OpReturn for compiled functions) is balanced.
	for _, arg := range args {
		if arg.Type >= value.FirstArenaType && !arg.Static {
			v.alloc.RetainAllocated(arg)
		}
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
	for atomic.LoadInt64(&v.abort) == 0 {
		v.ip++
		switch opcode.Opcode(v.curInsts[v.ip]) {
		case opcode.Nop:
			// do nothing

		case opcode.StaticPrimitiveValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = v.static.Primitives[n]
			v.sp++

		case opcode.StaticDecimalValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(value.Decimal, true, uint64(n))
			v.sp++

		case opcode.StaticStringValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(value.String, true, uint64(n))
			v.sp++

		case opcode.StaticRunesValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(value.Runes, true, uint64(n))
			v.sp++

		case opcode.StaticFormatSpecValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(value.FormatSpec, true, uint64(n))
			v.sp++

		case opcode.StaticCompiledFunctionValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(value.CompiledFunction, true, uint64(n))
			v.sp++

		case opcode.BComplement:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Int: // fast track for integer
				v.stack[v.sp] = core.IntValue(^int64(l.Data))
				v.sp++
			default:
				res, err := l.UnaryOp(v.alloc, token.Xor)
				if err != nil {
					v.err = err
					return
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.Pop:
			v.sp--
			t := v.stack[v.sp]
			if t.Type >= value.FirstArenaType && !t.Static {
				v.alloc.ReleaseAllocated(t)
			}

		case opcode.True:
			v.stack[v.sp] = core.True
			v.sp++

		case opcode.False:
			v.stack[v.sp] = core.False
			v.sp++

		case opcode.Equal:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res := core.BoolValue(l == r || l.Equal(v.alloc, r))
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			if r.Type >= value.FirstArenaType && !r.Static {
				v.alloc.ReleaseAllocated(r)
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.NotEqual:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res := core.BoolValue(!(l == r || l.Equal(v.alloc, r)))
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			if r.Type >= value.FirstArenaType && !r.Static {
				v.alloc.ReleaseAllocated(r)
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.Minus:
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
				res, err := l.UnaryOp(v.alloc, token.Sub)
				if err != nil {
					v.err = err
					return
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.LNot:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Bool: // fast track for booleans
				v.stack[v.sp] = core.BoolValue(l.Data == 0)
				v.sp++
			default:
				res := core.BoolValue(!l.IsTrue(v.alloc))
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.JumpFalsy:
			v.ip += 2
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			default:
				if !l.IsTrue(v.alloc) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
			}

		case opcode.AndJump:
			v.ip += 2
			l := v.stack[v.sp-1]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				} else {
					v.sp--
					if l.Type >= value.FirstArenaType && !l.Static {
						v.alloc.ReleaseAllocated(l)
					}
				}
			default:
				if !l.IsTrue(v.alloc) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				} else {
					v.sp--
					if l.Type >= value.FirstArenaType && !l.Static {
						v.alloc.ReleaseAllocated(l)
					}
				}
			}

		case opcode.OrJump:
			v.ip += 2
			l := v.stack[v.sp-1]
			switch l.Type {
			case value.Bool: // fast track for booleans
				if l.Data == 0 {
					v.sp--
					if l.Type >= value.FirstArenaType && !l.Static {
						v.alloc.ReleaseAllocated(l)
					}
				} else {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			default:
				if !l.IsTrue(v.alloc) {
					v.sp--
					if l.Type >= value.FirstArenaType && !l.Static {
						v.alloc.ReleaseAllocated(l)
					}
				} else {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			}

		case opcode.Jump:
			pos := int(v.curInsts[v.ip+2]) | int(v.curInsts[v.ip+1])<<8
			v.ip = pos - 1

		case opcode.Null:
			v.stack[v.sp] = core.Undefined
			v.sp++

		case opcode.Array:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			elements := v.alloc.NewArray(n, false)
			for i := v.sp - n; i < v.sp; i++ {
				e := v.stack[i]
				if e.Type >= value.FirstArenaType && !e.Static {
					v.alloc.PinAllocated(e) // mark it as unmanaged because now array is also owns it
				}
				elements = append(elements, e)
			}
			v.sp -= n
			nv, err := v.alloc.NewArrayValue(elements, false)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.Record:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]core.Value, n)
			for i := v.sp - n; i < v.sp; i += 2 {
				l := v.stack[i]
				e := v.stack[i+1]
				if e.Type >= value.FirstArenaType && !e.Static {
					v.alloc.PinAllocated(e) // mark it as unmanaged because now record is also owns it
				}
				switch l.Type {
				case value.String: // fast track for strings
					kv[*v.alloc.ResolveStringValue(l)] = e
				default:
					key, ok := l.AsString(v.alloc)
					if !ok {
						v.err = errs.NewInvalidArgumentTypeError("record", "key", "string", v.stack[i].TypeName(v.alloc))
						return
					}
					kv[key] = e
				}
			}
			v.sp -= n
			nv, err := v.alloc.NewRecordValue(kv, false)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.Contains:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			res := core.BoolValue(r.Contains(v.alloc, l))
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			if r.Type >= value.FirstArenaType && !r.Static {
				v.alloc.ReleaseAllocated(r)
			}
			v.stack[v.sp-2] = res
			v.sp--

		case opcode.Immutable:
			val := v.stack[v.sp-1]
			t, err := val.ToImmutable(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			// ToImmutable only flips the immutable flag; the slot keeps ownership of the same underlying ref.
			v.stack[v.sp-1] = t

		case opcode.Index:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res, err := l.Access(v.alloc, n, opcode.Index)
			if err != nil {
				if n.Type >= value.FirstArenaType && !n.Static {
					v.alloc.ReleaseAllocated(n)
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = err
				return
			}
			if n.Type >= value.FirstArenaType && !n.Static {
				v.alloc.ReleaseAllocated(n)
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.SliceIndex:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			l := v.stack[v.sp-3]
			v.sp -= 3
			res, err := l.Slice(v.alloc, low, high)
			if err != nil {
				if low.Type >= value.FirstArenaType && !low.Static {
					v.alloc.ReleaseAllocated(low)
				}
				if high.Type >= value.FirstArenaType && !high.Static {
					v.alloc.ReleaseAllocated(high)
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = err
				return
			}
			if low.Type >= value.FirstArenaType && !low.Static {
				v.alloc.ReleaseAllocated(low)
			}
			if high.Type >= value.FirstArenaType && !high.Static {
				v.alloc.ReleaseAllocated(high)
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.SliceIndexStep:
			step := v.stack[v.sp-1]
			high := v.stack[v.sp-2]
			low := v.stack[v.sp-3]
			l := v.stack[v.sp-4]
			v.sp -= 4
			res, err := l.SliceStep(v.alloc, low, high, step)
			if err != nil {
				if low.Type >= value.FirstArenaType && !low.Static {
					v.alloc.ReleaseAllocated(low)
				}
				if high.Type >= value.FirstArenaType && !high.Static {
					v.alloc.ReleaseAllocated(high)
				}
				if step.Type >= value.FirstArenaType && !step.Static {
					v.alloc.ReleaseAllocated(step)
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = err
				return
			}
			if low.Type >= value.FirstArenaType && !low.Static {
				v.alloc.ReleaseAllocated(low)
			}
			if high.Type >= value.FirstArenaType && !high.Static {
				v.alloc.ReleaseAllocated(high)
			}
			if step.Type >= value.FirstArenaType && !step.Static {
				v.alloc.ReleaseAllocated(step)
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.Call:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			val := v.stack[v.sp-1-numArgs]
			if val.Type != value.CompiledFunction && val.Type != value.BuiltinFunction && val.Type != value.BuiltinClosure && !val.IsCallable(v.alloc) {
				v.err = errs.NewNotCallableError(val.TypeName(v.alloc))
				return
			}

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case value.Array:
					o := v.alloc.ResolveArrayValue(arg)
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
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName(v.alloc))
					return
				}
			}

			switch val.Type {
			case value.CompiledFunction: // special case for compiled functions
				callee := v.alloc.ResolveCompiledFunctionValue(val)
				if callee.VarArgs {
					// if the closure is variadic, roll up all variadic parameters into an array
					realArgs := int(callee.NumParameters) - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := v.alloc.NewArray(varArgs, true)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						nv, err := v.alloc.NewArrayValue(args, true)
						if err != nil {
							v.err = err
							return
						}
						v.stack[spStart] = nv
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
					nextOp := opcode.Opcode(v.curInsts[v.ip+1])
					if nextOp == opcode.Return || (nextOp == opcode.Pop && opcode.Return == opcode.Opcode(v.curInsts[v.ip+2])) {
						// Move new args into the first numArgs local slots (ownership transfer).
						// Release the old local values they overwrite. Remaining locals retain their old
						// values; the compiler assigns to them before reading.
						for p := 0; p < numArgs; p++ {
							old := v.stack[v.curFrame.basePointer+p]
							v.stack[v.curFrame.basePointer+p] = v.stack[v.sp-numArgs+p]
							if old.Type >= value.FirstArenaType && !old.Static {
								v.alloc.ReleaseAllocated(old)
							}
						}
						// Release the callee slot before discarding it via sp decrement.
						t := v.stack[v.sp-numArgs-1]
						if t.Type >= value.FirstArenaType && !t.Static {
							v.alloc.ReleaseAllocated(t)
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
				res, err := core.BuiltinFunctions[val.Data].Func(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
				// Pin res so it survives the arg/callee releases below (res may alias an arg or be sourced
				// from a container element). The stack-slot write that follows becomes the new +1 owner;
				// pinning leaks at most one slot until Arena.Reset, which is acceptable per §5a.
				if res.Type >= value.FirstArenaType && !res.Static {
					v.alloc.PinAllocated(res)
				}
				for i := v.sp - numArgs; i < v.sp; i++ {
					t := v.stack[i]
					if t.Type >= value.FirstArenaType && !t.Static {
						v.alloc.ReleaseAllocated(t)
					}
				}
				t := v.stack[v.sp-numArgs-1]
				if t.Type >= value.FirstArenaType && !t.Static {
					v.alloc.ReleaseAllocated(t)
				}
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			case value.BuiltinClosure: // fast track for built-in closure
				res, err := v.alloc.ResolveBuiltinClosureValue(val).Func(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
				if res.Type >= value.FirstArenaType && !res.Static {
					v.alloc.PinAllocated(res)
				}
				for i := v.sp - numArgs; i < v.sp; i++ {
					t := v.stack[i]
					if t.Type >= value.FirstArenaType && !t.Static {
						v.alloc.ReleaseAllocated(t)
					}
				}
				t := v.stack[v.sp-numArgs-1]
				if t.Type >= value.FirstArenaType && !t.Static {
					v.alloc.ReleaseAllocated(t)
				}
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			default:
				res, err := val.Call(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
				if res.Type >= value.FirstArenaType && !res.Static {
					v.alloc.PinAllocated(res)
				}
				for i := v.sp - numArgs; i < v.sp; i++ {
					t := v.stack[i]
					if t.Type >= value.FirstArenaType && !t.Static {
						v.alloc.ReleaseAllocated(t)
					}
				}
				t := v.stack[v.sp-numArgs-1]
				if t.Type >= value.FirstArenaType && !t.Static {
					v.alloc.ReleaseAllocated(t)
				}
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.Return:
			v.ip++
			hasResult := v.curInsts[v.ip] == 1
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
					v.err = unwrapKavunError(v.alloc, errVal)
					return
				}
				// Defers may have updated the named-result slot; re-read it (covers both bare return and `return EXPR`
				// since the latter wrote EXPR into the slot above).
				if v.curFrame.fn.HasNamedResult() {
					res = v.readNamedResult(v.curFrame)
				}
			}
			// Pin res to keep it alive across releaseFrameLocals: res may alias a local (named result, returned local
			// variable, or a value pulled from a local container). Pinning is safe and cheap; the caller becomes the
			// +1 owner via the stack-slot write below.
			if res.Type >= value.FirstArenaType && !res.Static {
				v.alloc.PinAllocated(res)
			}
			// Release every local slot of the popped frame so refpool entries can be reused immediately.
			v.releaseFrameLocals(v.curFrame)
			v.framesIndex--
			v.frames[v.framesIndex].defers = nil
			v.frames[v.framesIndex].inFlightErr = core.Undefined
			v.frames[v.framesIndex].deferredFor = nil
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip
			v.sp = v.frames[v.framesIndex].basePointer
			// The callee value occupies stack[sp-1]; release it before overwriting with the result.
			t := v.stack[v.sp-1]
			if t.Type >= value.FirstArenaType && !t.Static {
				v.alloc.ReleaseAllocated(t)
			}
			v.stack[v.sp-1] = res

		case opcode.Defer:
			v.ip++
			numArgs := int(v.curInsts[v.ip])
			// Stack layout: [..., callee, arg1, ..., argN]
			argsStart := v.sp - numArgs
			calleeIdx := argsStart - 1
			callee := v.stack[calleeIdx]
			// Copy args out of the operand stack into a fresh slice (arena-allocated to avoid per-defer GC pressure
			// in hot loops) so later stack operations cannot mutate them.
			var capturedArgs []core.Value
			if numArgs > 0 {
				capturedArgs = v.alloc.NewArray(numArgs, true)
				copy(capturedArgs, v.stack[argsStart:v.sp])
			}
			v.curFrame.defers = append(v.curFrame.defers, deferred{fn: callee, args: capturedArgs})
			v.sp = calleeIdx

		case opcode.DeferMethod:
			// Operands: [methodIdx (2 bytes), numArgs (1 byte)]
			methodIdx := (int(v.curInsts[v.ip+1]) << 8) | int(v.curInsts[v.ip+2])
			numArgs := int(v.curInsts[v.ip+3])
			v.ip += 3
			methodName := v.static.Strings[methodIdx]
			argsStart := v.sp - numArgs
			recvIdx := argsStart - 1
			recv := v.stack[recvIdx]
			var capturedArgs []core.Value
			if numArgs > 0 {
				capturedArgs = v.alloc.NewArray(numArgs, true)
				copy(capturedArgs, v.stack[argsStart:v.sp])
			}
			v.curFrame.defers = append(v.curFrame.defers, deferred{fn: recv, args: capturedArgs, method: methodName})
			v.sp = recvIdx

		case opcode.GetGlobal:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			e := v.globals[n]
			if e.Type >= value.FirstArenaType && !e.Static {
				v.alloc.RetainAllocated(e) // increase ref count because we copy value to stack
			}
			v.stack[v.sp] = e // copy global value to stack
			v.sp++

		case opcode.SetGlobal:
			v.ip += 2
			v.sp--
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			t := v.globals[n]
			if t.Type >= value.FirstArenaType && !t.Static {
				v.alloc.ReleaseAllocated(t) // release old global value before overwriting it
			}
			v.globals[n] = v.stack[v.sp] // move value from stack to global (sp is decremented)

		case opcode.SetSelGlobal:
			v.ip += 3
			globalIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numSelectors := int(v.curInsts[v.ip])
			// selectors and RHS value
			selectors := v.alloc.NewArray(numSelectors, true)
			for i := range numSelectors {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := v.indexAssign(v.globals[globalIndex], val, selectors)
			for _, sel := range selectors {
				if sel.Type >= value.FirstArenaType && !sel.Static {
					v.alloc.ReleaseAllocated(sel)
				}
			}
			if val.Type >= value.FirstArenaType && !val.Static {
				v.alloc.ReleaseAllocated(val)
			}
			if e != nil {
				v.err = e
				return
			}

		case opcode.GetLocal:
			v.ip++
			n := int(v.curInsts[v.ip])
			e := v.stack[v.curFrame.basePointer+n]
			if e.Type == value.ValuePtr {
				e = **v.alloc.ResolveValuePtrValue(e)
			}
			if e.Type >= value.FirstArenaType && !e.Static {
				v.alloc.RetainAllocated(e) // increase ref count because we copy value to stack
			}
			v.stack[v.sp] = e // copy local value to stack
			v.sp++

		case opcode.SetLocal:
			n := int(v.curInsts[v.ip+1])
			v.ip++
			sp := v.curFrame.basePointer + n
			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			v.sp--
			val := v.stack[v.sp] // move value from stack to local slot (sp is decremented)
			if v.stack[sp].Type == value.ValuePtr {
				// if target slot is a free variable, update the pointee value so all referencing free variables can observe the change
				**v.alloc.ResolveValuePtrValue(v.stack[sp]) = val
			} else {
				t := v.stack[sp]
				if t.Type >= value.FirstArenaType && !t.Static {
					v.alloc.ReleaseAllocated(t) // release old value before overwriting it
				}
				v.stack[sp] = val // move val from local slot to stack
			}

		case opcode.DefineLocal:
			v.ip++
			v.sp--
			sp := v.curFrame.basePointer + int(v.curInsts[v.ip])
			t := v.stack[sp]
			if t.Type >= value.FirstArenaType && !t.Static {
				v.alloc.ReleaseAllocated(t) // release old value before overwriting it
			}
			v.stack[sp] = v.stack[v.sp] // move value from stack (sp is decremented)

		case opcode.SetSelLocal:
			localIndex := int(v.curInsts[v.ip+1])
			numSelectors := int(v.curInsts[v.ip+2])
			v.ip += 2
			// selectors and RHS value
			selectors := v.alloc.NewArray(numSelectors, true)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			dst := v.stack[v.curFrame.basePointer+localIndex]
			if dst.Type == value.ValuePtr {
				dst = **v.alloc.ResolveValuePtrValue(dst)
			}
			e := v.indexAssign(dst, val, selectors)
			for _, sel := range selectors {
				if sel.Type >= value.FirstArenaType && !sel.Static {
					v.alloc.ReleaseAllocated(sel)
				}
			}
			if val.Type >= value.FirstArenaType && !val.Static {
				v.alloc.ReleaseAllocated(val)
			}
			if e != nil {
				v.err = e
				return
			}

		case opcode.GetFreePtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			nv, err := v.alloc.NewValuePtrValue(v.curFrame.freeVars[n])
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.GetFree:
			v.ip++
			nv := *v.curFrame.freeVars[int(v.curInsts[v.ip])]
			if nv.Type >= value.FirstArenaType && !nv.Static {
				v.alloc.RetainAllocated(nv) // increase ref count because we copy value to stack
			}
			v.stack[v.sp] = nv // copy free variable value to stack
			v.sp++

		case opcode.SetFree:
			v.ip++
			n := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[n] = v.stack[v.sp-1] // move value from stack to free variable (sp is decremented)
			v.sp--

		case opcode.GetLocalPtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + n
			var freeVar *core.Value
			if v.stack[sp].Type == value.ValuePtr {
				freeVar = *v.alloc.ResolveValuePtrValue(v.stack[sp])
			} else {
				val := v.stack[sp]
				freeVar = &val
				nv, err := v.alloc.NewValuePtrValue(freeVar)
				if err != nil {
					v.err = err
					return
				}
				v.stack[sp] = nv
			}
			nv, err := v.alloc.NewValuePtrValue(freeVar)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.SetSelFree:
			v.ip += 2
			freeIndex := int(v.curInsts[v.ip-1])
			numSelectors := int(v.curInsts[v.ip])
			// selectors and RHS value
			selectors := v.alloc.NewArray(numSelectors, true)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := v.indexAssign(*v.curFrame.freeVars[freeIndex], val, selectors)
			for _, sel := range selectors {
				if sel.Type >= value.FirstArenaType && !sel.Static {
					v.alloc.ReleaseAllocated(sel)
				}
			}
			if val.Type >= value.FirstArenaType && !val.Static {
				v.alloc.ReleaseAllocated(val)
			}
			if e != nil {
				v.err = e
				return
			}

		case opcode.GetBuiltinFunction:
			v.ip++
			v.stack[v.sp] = core.BuiltinFunctionValue(uint64(v.curInsts[v.ip]))
			v.sp++

		case opcode.ImportBuiltinModule:
			v.ip++
			m, err := stdlib.GetModule(v.alloc, v.curInsts[v.ip])
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = m
			v.sp++

		case opcode.Closure:
			v.ip += 3
			numFree := int(v.curInsts[v.ip])
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				// Compiler guarantees each free-var operand is a ValuePtr pushed by GetLocalPtr/GetFreePtr.
				ptr := v.stack[v.sp-numFree+i]
				free[i] = *v.alloc.ResolveValuePtrValue(ptr)
				if ptr.Type >= value.FirstArenaType && !ptr.Static {
					v.alloc.ReleaseAllocated(ptr)
				}
			}
			v.sp -= numFree
			n := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			fn := v.static.CompiledFunctions[n]
			nv, err := v.alloc.NewCompiledFunctionValue(fn.Instructions, free, fn.SourceMap, fn.NumLocals, fn.MaxStack, fn.NumParameters, fn.VarArgs, fn.NamedResult)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.IteratorInit:
			l := v.stack[v.sp-1]
			v.sp--
			if !l.IsIterable(v.alloc) {
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = errs.NewNotIterableError(l.TypeName(v.alloc))
				return
			}
			it, err := l.Iterator(v.alloc)
			if err != nil {
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = err
				return
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			v.stack[v.sp] = it
			v.sp++

		case opcode.IteratorNext:
			it := v.stack[v.sp-1]
			res := core.BoolValue(it.Next(v.alloc))
			if it.Type >= value.FirstArenaType && !it.Static {
				v.alloc.ReleaseAllocated(it)
			}
			v.stack[v.sp-1] = res

		case opcode.IteratorKey:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Key(v.alloc)
			if err != nil {
				if it.Type >= value.FirstArenaType && !it.Static {
					v.alloc.ReleaseAllocated(it)
				}
				v.err = err
				return
			}
			if it.Type >= value.FirstArenaType && !it.Static {
				v.alloc.ReleaseAllocated(it)
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.IteratorValue:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Value(v.alloc)
			if err != nil {
				if it.Type >= value.FirstArenaType && !it.Static {
					v.alloc.ReleaseAllocated(it)
				}
				v.err = err
				return
			}
			if it.Type >= value.FirstArenaType && !it.Static {
				v.alloc.ReleaseAllocated(it)
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.BinaryOp:
			v.ip++
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
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
			res, err := l.BinaryOp(v.alloc, tok, r)
			if err != nil {
				v.sp -= 2
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				if r.Type >= value.FirstArenaType && !r.Static {
					v.alloc.ReleaseAllocated(r)
				}
				v.err = err
				return
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			if r.Type >= value.FirstArenaType && !r.Static {
				v.alloc.ReleaseAllocated(r)
			}
			v.stack[v.sp-2] = res
			v.sp--

		case opcode.Suspend:
			return

		case opcode.Format:
			n := (int(v.curInsts[v.ip+1]) << 8) | int(v.curInsts[v.ip+2])
			v.ip += 2
			fs := v.static.FormatSpecs[n]
			val := v.stack[v.sp-1]
			s, err := val.Format(v.alloc, fs.Spec)
			if err != nil {
				v.sp--
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				v.err = err
				return
			}
			nv, err := v.alloc.NewStringValue(s)
			if err != nil {
				v.sp--
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				v.err = err
				return
			}
			if val.Type >= value.FirstArenaType && !val.Static {
				v.alloc.ReleaseAllocated(val)
			}
			v.stack[v.sp-1] = nv

		case opcode.FormatDyn:
			specVal := v.stack[v.sp-1]
			val := v.stack[v.sp-2]
			v.sp -= 2
			if specVal.Type != value.String {
				v.err = errs.NewInvalidArgumentTypeError("f-string", "spec", "string", specVal.TypeName(v.alloc))
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				if specVal.Type >= value.FirstArenaType && !specVal.Static {
					v.alloc.ReleaseAllocated(specVal)
				}
				return
			}
			specText := *v.alloc.ResolveStringValue(specVal)
			parsed, err := fspec.Parse(specText)
			if err != nil {
				v.err = errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, fmt.Sprintf("f-string format spec %q: %v", specText, err))
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				if specVal.Type >= value.FirstArenaType && !specVal.Static {
					v.alloc.ReleaseAllocated(specVal)
				}
				return
			}
			s, err := val.Format(v.alloc, parsed)
			if err != nil {
				v.err = err
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				if specVal.Type >= value.FirstArenaType && !specVal.Static {
					v.alloc.ReleaseAllocated(specVal)
				}
				return
			}
			nv, err := v.alloc.NewStringValue(s)
			if err != nil {
				v.err = err
				if val.Type >= value.FirstArenaType && !val.Static {
					v.alloc.ReleaseAllocated(val)
				}
				if specVal.Type >= value.FirstArenaType && !specVal.Static {
					v.alloc.ReleaseAllocated(specVal)
				}
				return
			}
			if val.Type >= value.FirstArenaType && !val.Static {
				v.alloc.ReleaseAllocated(val)
			}
			if specVal.Type >= value.FirstArenaType && !specVal.Static {
				v.alloc.ReleaseAllocated(specVal)
			}
			v.stack[v.sp] = nv
			v.sp++

		case opcode.Select:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			val, err := l.Access(v.alloc, n, opcode.Select)
			if err != nil {
				if n.Type >= value.FirstArenaType && !n.Static {
					v.alloc.ReleaseAllocated(n)
				}
				if l.Type >= value.FirstArenaType && !l.Static {
					v.alloc.ReleaseAllocated(l)
				}
				v.err = err
				return
			}
			if n.Type >= value.FirstArenaType && !n.Static {
				v.alloc.ReleaseAllocated(n)
			}
			if l.Type >= value.FirstArenaType && !l.Static {
				v.alloc.ReleaseAllocated(l)
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.MethodCall:
			n := int(v.curInsts[v.ip+2]) | int(v.curInsts[v.ip+1])<<8
			numArgs := int(v.curInsts[v.ip+3])
			spread := v.curInsts[v.ip+4]
			v.ip += 4
			receiver := v.stack[v.sp-1-numArgs]
			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case value.Array:
					o := v.alloc.ResolveArrayValue(arg)
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
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName(v.alloc))
					return
				}
				receiver = v.stack[v.sp-1-numArgs]
			}

			name := v.static.Strings[n]
			res, err := receiver.MethodCall(v.alloc, v, name, v.stack[v.sp-numArgs:v.sp])
			if res.Type >= value.FirstArenaType && !res.Static {
				v.alloc.PinAllocated(res)
			}
			for i := v.sp - numArgs; i < v.sp; i++ {
				t := v.stack[i]
				if t.Type >= value.FirstArenaType && !t.Static {
					v.alloc.ReleaseAllocated(t)
				}
			}
			t := v.stack[v.sp-numArgs-1]
			if t.Type >= value.FirstArenaType && !t.Static {
				v.alloc.ReleaseAllocated(t)
			}
			v.sp -= numArgs + 1
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
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
		next, err := dst.Access(v.alloc, selectors[si], opcode.Index)
		if err != nil {
			return err
		}
		dst = next
	}
	return dst.Assign(v.alloc, selectors[0], src)
}

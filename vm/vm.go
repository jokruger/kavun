package vm

import (
	"fmt"
	"math"
	"sync/atomic"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/opcode"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/token"
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
	// VT_UNDEFINED when no error is in flight.
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
	alloc       *core.Arena  // object allocator used by arrays, records, iterators, errors, closures, and call helpers
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

// Reset resets the VM state to run new main function.
func (v *VM) Reset(alloc *core.Arena, bytecode *Bytecode, globals []core.Value) {
	if globals == nil {
		globals = make([]core.Value, GlobalsSize)
	}

	v.ip = -1
	v.sp = 0
	atomic.StoreInt64(&v.abort, 0)
	v.constants = bytecode.Constants
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

// Clear clears the remaining stack and frames to release references to heap objects and help GC.
// This step is not strictly necessary for correctness, but can help reduce memory usage and GC overhead when the same VM is reused for multiple runs.
func (v *VM) Clear() {
	for i := range v.frames {
		v.frames[i].fn = nil
		v.frames[i].freeVars = nil
		v.frames[i].defers = nil
		v.frames[i].inFlightErr = core.Undefined
		v.frames[i].deferredFor = nil
	}
	for i := range v.stack {
		v.stack[i].Ptr = nil
	}
}

// Allocator returns the allocator used by the VM.
func (v *VM) Allocator() *core.Arena {
	return v.alloc
}

// Abort aborts the execution.
func (v *VM) Abort() {
	atomic.StoreInt64(&v.abort, 1)
}

// IsStackEmpty tests if the stack is empty or not.
func (v *VM) IsStackEmpty() bool {
	return v.sp == 0
}

// Recover implements the core.VM interface. It returns the in-flight error of the surrounding "deferred-for" frame
// (and clears it) so the surrounding function returns normally; otherwise Undefined.
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
	if target.inFlightErr.Type == core.VT_UNDEFINED {
		return core.Undefined
	}
	err := target.inFlightErr
	target.inFlightErr = core.Undefined
	return err
}

// Call calls a compiled function with the given arguments and returns the result.
func (v *VM) Call(cfv core.Value, args []core.Value) (core.Value, error) {
	if cfv.Type != core.VT_COMPILED_FUNCTION {
		return core.Undefined, errs.NewInvalidArgumentTypeError("call", "function", "compiled function", cfv.TypeName(v.alloc))
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
			newArgs := v.alloc.NewArray(realArgs+1, true)
			copy(newArgs, args[:realArgs])
			t := v.alloc.NewArrayValue(args[realArgs:], true)
			newArgs[realArgs] = t
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
	// This is where OpReturn will write the return value
	v.stack[v.sp] = cfv // Use the function itself as placeholder
	v.sp++

	// Push arguments onto stack
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
		case opcode.StaticPrimitiveValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = v.static.Primitives[n]
			v.sp++

		case opcode.StaticDecimalValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(core.VT_DECIMAL, true, uint64(n))
			v.sp++

		case opcode.StaticStringValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(core.VT_STRING, true, uint64(n))
			v.sp++

		case opcode.StaticRunesValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(core.VT_RUNES, true, uint64(n))
			v.sp++

		case opcode.StaticFormatSpecValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(core.VT_FORMAT_SPEC, true, uint64(n))
			v.sp++

		case opcode.StaticCompiledFunctionValue:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = core.StaticValue(core.VT_COMPILED_FUNCTION, true, uint64(n))
			v.sp++

		case opcode.BComplement:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_INT: // fast track for integer
				v.stack[v.sp] = core.IntValue(^int64(l.Data))
				v.sp++
			default:
				res, err := l.UnaryOp(v.alloc, token.Xor)
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.Pop:
			v.sp--

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
			v.stack[v.sp] = core.BoolValue(l == r || l.Equal(v.alloc, r))
			v.sp++

		case opcode.NotEqual:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(!(l == r || l.Equal(v.alloc, r)))
			v.sp++

		case opcode.Minus:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_INT: // fast track for integers
				v.stack[v.sp] = core.IntValue(-int64(l.Data))
				v.sp++
			case core.VT_FLOAT: // fast track for floats
				v.stack[v.sp] = core.FloatValue(-math.Float64frombits(l.Data))
				v.sp++
			default:
				res, err := l.UnaryOp(v.alloc, token.Sub)
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++
			}

		case opcode.LNot:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_BOOL: // fast track for booleans
				v.stack[v.sp] = core.BoolValue(l.Data == 0)
				v.sp++
			default:
				v.stack[v.sp] = core.BoolValue(!l.IsTrue(v.alloc))
				v.sp++
			}

		case opcode.JumpFalsy:
			v.ip += 2
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_BOOL: // fast track for booleans
				if l.Data == 0 {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			default:
				if !l.IsTrue(v.alloc) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			}

		case opcode.AndJump:
			v.ip += 2
			l := v.stack[v.sp-1]
			switch l.Type {
			case core.VT_BOOL: // fast track for booleans
				if l.Data == 0 {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				} else {
					v.sp--
				}
			default:
				if !l.IsTrue(v.alloc) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				} else {
					v.sp--
				}
			}

		case opcode.OrJump:
			v.ip += 2
			l := v.stack[v.sp-1]
			switch l.Type {
			case core.VT_BOOL: // fast track for booleans
				if l.Data == 0 {
					v.sp--
				} else {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
					v.ip = pos - 1
				}
			default:
				if !l.IsTrue(v.alloc) {
					v.sp--
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
				elements = append(elements, v.stack[i])
			}
			v.sp -= n
			v.stack[v.sp] = v.alloc.NewArrayValue(elements, false)
			v.sp++

		case opcode.Record:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]core.Value, n)
			for i := v.sp - n; i < v.sp; i += 2 {
				l := v.stack[i]
				switch l.Type {
				case core.VT_STRING: // fast track for strings
					kv[*(*string)(l.Ptr)] = v.stack[i+1]
				default:
					key, ok := l.AsString(v.alloc)
					if !ok {
						v.err = errs.NewInvalidArgumentTypeError("record", "key", "string", v.stack[i].TypeName(v.alloc))
						return
					}
					kv[key] = v.stack[i+1]
				}
			}
			v.sp -= n
			v.stack[v.sp] = v.alloc.NewRecordValue(kv, false)
			v.sp++

		case opcode.Contains:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			res := core.BoolValue(r.Contains(v.alloc, l))
			v.stack[v.sp-2] = res
			v.sp--

		case opcode.Immutable:
			val := v.stack[v.sp-1]
			t, err := val.ToImmutable(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp-1] = t

		case opcode.Index:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res, err := l.Access(v.alloc, n, opcode.Index)
			if err != nil {
				v.err = err
				return
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
				v.err = err
				return
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
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case opcode.Call:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			val := v.stack[v.sp-1-numArgs]
			if val.Type != core.VT_COMPILED_FUNCTION && val.Type != core.VT_BUILTIN_FUNCTION && val.Type != core.VT_BUILTIN_CLOSURE && !val.IsCallable(v.alloc) {
				v.err = errs.NewNotCallableError(val.TypeName(v.alloc))
				return
			}

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case core.VT_ARRAY:
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
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName(v.alloc))
					return
				}
			}

			switch val.Type {
			case core.VT_COMPILED_FUNCTION: // special case for compiled functions
				callee := (*core.CompiledFunction)(val.Ptr)
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
						v.stack[spStart] = v.alloc.NewArrayValue(args, true)
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
				v.sp = v.sp - numArgs + callee.NumLocals
				if callee.HasNamedResult() {
					v.stack[v.curFrame.basePointer+callee.NamedResultSlot()] = core.Undefined
				}

			case core.VT_BUILTIN_FUNCTION: // fast track for built-in functions
				res, err := core.BuiltinFunctions[val.Data].Func(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			case core.VT_BUILTIN_CLOSURE: // fast track for built-in closure
				res, err := (*core.BuiltinClosure)(val.Ptr).Func(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1
				if err != nil {
					v.err = err
					return
				}
				v.stack[v.sp] = res
				v.sp++

			default:
				res, err := val.Call(v.alloc, v, v.stack[v.sp-numArgs:v.sp])
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
				if v.curFrame.inFlightErr.Type != core.VT_UNDEFINED {
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
			v.framesIndex--
			v.frames[v.framesIndex].defers = nil
			v.frames[v.framesIndex].inFlightErr = core.Undefined
			v.frames[v.framesIndex].deferredFor = nil
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip
			v.sp = v.frames[v.framesIndex].basePointer
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
			if methodIdx < 0 || methodIdx >= len(v.constants) {
				v.err = errs.NewInternalError(fmt.Sprintf("OpDeferMethod: invalid method constant index %d", methodIdx))
				return
			}
			nameVal := v.constants[methodIdx]
			if nameVal.Type != core.VT_STRING {
				v.err = errs.NewInternalError(fmt.Sprintf("OpDeferMethod: method name constant is not a string (got %s)", nameVal.TypeName(v.alloc)))
				return
			}
			methodName, _ := nameVal.AsString(v.alloc)
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
			v.stack[v.sp] = v.globals[n]
			v.sp++

		case opcode.SetGlobal:
			v.ip += 2
			v.sp--
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			v.globals[n] = v.stack[v.sp]

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
			if e != nil {
				v.err = e
				return
			}

		case opcode.GetLocal:
			v.ip++
			n := int(v.curInsts[v.ip])
			val := v.stack[v.curFrame.basePointer+n]
			if val.Type == core.VT_VALUE_PTR {
				val = *(*core.Value)(val.Ptr)
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.SetLocal:
			n := int(v.curInsts[v.ip+1])
			v.ip++
			sp := v.curFrame.basePointer + n
			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			val := v.stack[v.sp-1]
			v.sp--
			if v.stack[sp].Type == core.VT_VALUE_PTR {
				(*core.Value)(v.stack[sp].Ptr).Set(val)
				val = v.stack[sp]
			}
			v.stack[sp] = val // also use a copy of popped value

		case opcode.DefineLocal:
			v.ip++
			v.sp--
			v.stack[v.curFrame.basePointer+int(v.curInsts[v.ip])] = v.stack[v.sp]

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
			if dst.Type == core.VT_VALUE_PTR {
				dst = *(*core.Value)(dst.Ptr)
			}
			if e := v.indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case opcode.GetFreePtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			v.stack[v.sp] = v.alloc.NewValuePtrValue(v.curFrame.freeVars[n])
			v.sp++

		case opcode.GetFree:
			v.ip++
			n := int(v.curInsts[v.ip])
			v.stack[v.sp] = *v.curFrame.freeVars[n]
			v.sp++

		case opcode.SetFree:
			v.ip++
			n := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[n] = v.stack[v.sp-1]
			v.sp--

		case opcode.GetLocalPtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + n
			var freeVar *core.Value
			if v.stack[sp].Type == core.VT_VALUE_PTR {
				freeVar = (*core.Value)(v.stack[sp].Ptr)
			} else {
				localVal := v.stack[sp]
				freeVar = &localVal
				v.stack[sp] = v.alloc.NewValuePtrValue(freeVar)
			}
			v.stack[v.sp] = v.alloc.NewValuePtrValue(freeVar)
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
			constIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numFree := int(v.curInsts[v.ip])
			if v.constants[constIndex].Type != core.VT_COMPILED_FUNCTION {
				v.err = errs.NewInternalError(fmt.Sprintf("OpClosure: constant %d is not a function (got %s)", constIndex, v.constants[constIndex].TypeName(v.alloc)))
				return
			}
			fn := (*core.CompiledFunction)(v.constants[constIndex].Ptr)
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				if v.stack[v.sp-numFree+i].Type == core.VT_VALUE_PTR {
					free[i] = (*core.Value)(v.stack[v.sp-numFree+i].Ptr)
				} else {
					free[i] = &v.stack[v.sp-numFree+i]
				}
			}
			v.sp -= numFree
			v.stack[v.sp] = v.alloc.NewCompiledFunctionValue(fn.Instructions, free, fn.SourceMap, fn.NumLocals, fn.MaxStack, fn.NumParameters, fn.VarArgs, fn.NamedResult)
			v.sp++

		case opcode.IteratorInit:
			l := v.stack[v.sp-1]
			v.sp--
			if !l.IsIterable(v.alloc) {
				v.err = errs.NewNotIterableError(l.TypeName(v.alloc))
				return
			}
			it, err := l.Iterator(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = it
			v.sp++

		case opcode.IteratorNext:
			it := v.stack[v.sp-1]
			v.stack[v.sp-1] = core.BoolValue(it.Next(v.alloc))

		case opcode.IteratorKey:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Key(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.IteratorValue:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Value(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.BinaryOp:
			v.ip++
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
			if l.Type == core.VT_INT && r.Type == core.VT_INT {
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
			res, err := l.BinaryOp(v.Allocator(), tok, r)
			if err != nil {
				v.sp -= 2
				v.err = err
				return
			}
			v.stack[v.sp-2] = res
			v.sp--

		case opcode.Suspend:
			return

		case opcode.Format:
			specIdx := (int(v.curInsts[v.ip+1]) << 8) | int(v.curInsts[v.ip+2])
			v.ip += 2
			if specIdx < 0 || specIdx >= len(v.constants) {
				v.err = errs.NewInternalError(fmt.Sprintf("OpFormat: invalid format spec constant index %d", specIdx))
				return
			}
			specVal := v.constants[specIdx]
			if specVal.Type != core.VT_FORMAT_SPEC {
				v.err = errs.NewInternalError(fmt.Sprintf("OpFormat: constant %d is not a format spec (got %s)", specIdx, specVal.TypeName(v.alloc)))
				return
			}
			fs := (*core.FormatSpecValue)(specVal.Ptr)
			val := v.stack[v.sp-1]
			s, err := val.Format(v.alloc, fs.Spec)
			if err != nil {
				v.sp--
				v.err = err
				return
			}
			v.stack[v.sp-1] = v.alloc.NewStringValue(s)

		case opcode.FormatDyn:
			specVal := v.stack[v.sp-1]
			val := v.stack[v.sp-2]
			v.sp -= 2
			if specVal.Type != core.VT_STRING {
				v.err = errs.NewInvalidArgumentTypeError("f-string", "spec", "string", specVal.TypeName(v.alloc))
				return
			}
			specText := *(*string)(specVal.Ptr)
			parsed, err := fspec.Parse(specText)
			if err != nil {
				v.err = errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, fmt.Sprintf("f-string format spec %q: %v", specText, err))
				return
			}
			s, err := val.Format(v.alloc, parsed)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = v.alloc.NewStringValue(s)
			v.sp++

		case opcode.Select:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			val, err := l.Access(v.alloc, n, opcode.Select)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case opcode.MethodCall:
			methodConstIdx := int(v.curInsts[v.ip+2]) | int(v.curInsts[v.ip+1])<<8
			numArgs := int(v.curInsts[v.ip+3])
			spread := v.curInsts[v.ip+4]
			v.ip += 4

			if methodConstIdx < 0 || methodConstIdx >= len(v.constants) {
				v.err = errs.NewInternalError(fmt.Sprintf("OpMethodCall: invalid method constant index %d", methodConstIdx))
				return
			}

			receiver := v.stack[v.sp-1-numArgs]

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case core.VT_ARRAY:
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
					v.err = errs.NewInvalidArgumentTypeError("...", "spread", "array", arg.TypeName(v.alloc))
					return
				}
				receiver = v.stack[v.sp-1-numArgs]
			}

			name := v.constants[methodConstIdx]
			// method name can only be a string (due to the syntax of method call)
			if name.Type != core.VT_STRING {
				v.err = errs.NewInternalError(fmt.Sprintf("OpMethodCall: method name constant is not a string (got %s)", name.TypeName(v.alloc)))
				return
			}

			res, err := receiver.MethodCall(v.alloc, v, *(*string)(name.Ptr), v.stack[v.sp-numArgs:v.sp])
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

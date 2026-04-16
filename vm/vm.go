package vm

import (
	"fmt"
	"sync/atomic"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

var (
	callbackTrampolineInstructions = [...]byte{core.OpSuspend}
	callbackTrampolineFn           = &core.CompiledFunction{Instructions: callbackTrampolineInstructions[:]}
)

// frame represents a function call frame.
type frame struct {
	// Hot scalar fields first: read and written on every instruction fetch and return.
	ip          int // instruction pointer within fn.Instructions; -1 means "before first instruction"
	basePointer int // index into VM stack where this frame's locals start

	// Pointer fields: accessed on closure captures and function entry/exit.
	fn       *core.CompiledFunction // the function being executed
	freeVars []*core.Value          // captured free variables for closures
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
	constants   []core.Value   // constant pool used by OpConstant, method dispatch, closures, and other opcode operands
	globals     []core.Value   // global variable storage used by global load/store/select opcodes
	alloc       core.Allocator // object allocator used by arrays, records, iterators, errors, closures, and call helpers
	framesIndex int            // number of active frames; updated on calls, returns, and synthetic callback frames

	// Cold diagnostic state: only used when execution aborts or a stack trace is formatted.
	fileSet *parser.SourceFileSet // source positions for runtime stack traces
	err     error                 // last runtime error captured by run()

	// Large fixed-size arrays
	frames [MaxFrames]frame      // call frame stack
	stack  [StackSize]core.Value // operand stack
}

// NewVM creates a VM.
func NewVM(alloc core.Allocator, bytecode *Bytecode, globals []core.Value) *VM {
	if globals == nil {
		globals = make([]core.Value, GlobalsSize)
	}
	v := &VM{
		alloc:       alloc,
		constants:   bytecode.Constants,
		sp:          0,
		globals:     globals,
		fileSet:     bytecode.FileSet,
		framesIndex: 1,
		ip:          -1,
	}
	v.frames[0].fn = bytecode.MainFunction
	v.frames[0].ip = -1
	v.curFrame = &v.frames[0]
	v.curInsts = v.curFrame.fn.Instructions
	return v
}

// Allocator returns the allocator used by the VM.
func (v *VM) Allocator() core.Allocator {
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

// Call calls a compiled function with the given arguments and returns the result.
func (v *VM) Call(fn *core.CompiledFunction, args []core.Value) (core.Value, error) {
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
			t, err := v.alloc.NewArrayValue(args[realArgs:], true)
			if err != nil {
				return core.Undefined, err
			}
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
	if v.framesIndex+1 >= MaxFrames {
		v.err = errs.NewStackOverflowError("native callback frames")
		return core.Undefined, v.err
	}
	if v.sp+1+numArgs > StackSize {
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
	v.stack[v.sp] = core.CompiledFunctionValue(fn) // Use the function itself as placeholder
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
	v.curInsts = fn.Instructions
	v.ip = -1
	v.framesIndex++
	v.sp = v.sp - numArgs + fn.NumLocals

	// Execute the callback by calling run()
	// When callback returns (OpReturn), it will return to trampoline frame
	// Trampoline executes OpSuspend, which exits run()
	v.run()

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

	v.run()
	err = v.err
	if err != nil {
		filePos := v.fileSet.Position(v.curFrame.fn.SourcePos(v.ip - 1))
		err = fmt.Errorf("Runtime Error: %w\n\tat %s",
			err, filePos)
		for v.framesIndex > 1 {
			v.framesIndex--
			v.curFrame = &v.frames[v.framesIndex-1]
			filePos = v.fileSet.Position(v.curFrame.fn.SourcePos(v.curFrame.ip - 1))
			err = fmt.Errorf("%w\n\tat %s", err, filePos)
		}
		return err
	}
	return nil
}

func (v *VM) run() {
	for atomic.LoadInt64(&v.abort) == 0 {
		v.ip++
		switch v.curInsts[v.ip] {
		case core.OpConstant:
			v.ip += 2
			n := (int(v.curInsts[v.ip-1]) << 8) | int(v.curInsts[v.ip])
			v.stack[v.sp] = v.constants[n]
			v.sp++

		case core.OpBComplement:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_INT: // fast track for integer
				v.stack[v.sp] = core.IntValue(^core.ToInt(l))
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

		case core.OpPop:
			v.sp--

		case core.OpTrue:
			v.stack[v.sp] = core.True
			v.sp++

		case core.OpFalse:
			v.stack[v.sp] = core.False
			v.sp++

		case core.OpEqual:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(l == r || l.Equal(r))
			v.sp++

		case core.OpNotEqual:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(!(l == r || l.Equal(r)))
			v.sp++

		case core.OpMinus:
			v.sp--
			l := v.stack[v.sp]
			switch l.Type {
			case core.VT_INT: // fast track for integers
				v.stack[v.sp] = core.IntValue(-core.ToInt(l))
				v.sp++
			case core.VT_FLOAT: // fast track for floats
				v.stack[v.sp] = core.FloatValue(-core.ToFloat(l))
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

		case core.OpLNot:
			v.sp--
			l := v.stack[v.sp]
			if l.Type == core.VT_BOOL {
				// fast track for booleans
				v.stack[v.sp] = core.BoolValue(!core.ToBool(l))
			} else {
				v.stack[v.sp] = core.BoolValue(!l.IsTrue())
			}
			v.sp++

		case core.OpJumpFalsy:
			v.ip += 4
			v.sp--
			l := v.stack[v.sp]
			if l.Type == core.VT_BOOL {
				// fast track for booleans
				if !core.ToBool(l) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
					v.ip = pos - 1
				}
			} else if !l.IsTrue() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			}

		case core.OpAndJump:
			v.ip += 4
			l := v.stack[v.sp-1]
			if l.Type == core.VT_BOOL {
				// fast track for booleans
				if !core.ToBool(l) {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
					v.ip = pos - 1
				} else {
					v.sp--
				}
			} else if !l.IsTrue() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			} else {
				v.sp--
			}

		case core.OpOrJump:
			v.ip += 4
			l := v.stack[v.sp-1]
			if l.Type == core.VT_BOOL {
				// fast track for booleans
				if !core.ToBool(l) {
					v.sp--
				} else {
					pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
					v.ip = pos - 1
				}
			} else if !l.IsTrue() {
				v.sp--
			} else {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			}

		case core.OpJump:
			pos := int(v.curInsts[v.ip+4]) | int(v.curInsts[v.ip+3])<<8 | int(v.curInsts[v.ip+2])<<16 | int(v.curInsts[v.ip+1])<<24
			v.ip = pos - 1

		case core.OpNull:
			v.stack[v.sp] = core.Undefined
			v.sp++

		case core.OpArray:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			elements := make([]core.Value, 0, n)
			for i := v.sp - n; i < v.sp; i++ {
				elements = append(elements, v.stack[i])
			}
			v.sp -= n
			arr, err := v.alloc.NewArrayValue(elements, false)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = arr
			v.sp++

		case core.OpRecord:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]core.Value, n)
			for i := v.sp - n; i < v.sp; i += 2 {
				l := v.stack[i]
				if l.Type == core.VT_STRING {
					// fast track for strings
					kv[core.ToString(l).Value] = v.stack[i+1]
				} else {
					key, ok := l.AsString()
					if !ok {
						v.err = fmt.Errorf("record keys must be strings, got: %s", v.stack[i].TypeName())
						return
					}
					kv[key] = v.stack[i+1]
				}
			}
			v.sp -= n
			m, err := v.alloc.NewRecordValue(kv, false)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = m
			v.sp++

		case core.OpContains:
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			res := core.BoolValue(r.Contains(l))
			v.stack[v.sp-2] = res
			v.sp--

		case core.OpImmutable:
			val := v.stack[v.sp-1]
			t, err := val.Immutable(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp-1] = t

		case core.OpIndex:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2
			res, err := l.Access(v, n, core.OpIndex)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = res
			v.sp++

		case core.OpSliceIndex:
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

		case core.OpCall:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			val := v.stack[v.sp-1-numArgs]
			if val.Type != core.VT_COMPILED_FUNCTION && val.Type != core.VT_BUILTIN_FUNCTION && !val.IsCallable() {
				v.err = fmt.Errorf("not callable: %s", val.TypeName())
				return
			}

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case core.VT_ARRAY:
					o := (*core.Array)(arg.Ptr)
					for _, item := range o.Elements {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(o.Elements) - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arg.TypeName())
					return
				}
			}

			switch val.Type {
			case core.VT_COMPILED_FUNCTION: // special case for compiled functions
				callee := core.ToCompiledFunction(val)
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
						t, err := v.alloc.NewArrayValue(args, true)
						if err != nil {
							v.err = err
							return
						}
						v.stack[spStart] = t
						v.sp = spStart + 1
					}
				}
				if numArgs != int(callee.NumParameters) {
					if callee.VarArgs {
						v.err = fmt.Errorf("wrong number of arguments: want>=%d, got=%d", callee.NumParameters-1, numArgs)
					} else {
						v.err = fmt.Errorf("wrong number of arguments: want=%d, got=%d", callee.NumParameters, numArgs)
					}
					return
				}

				// test if it's tail-call
				if callee == v.curFrame.fn { // recursion
					nextOp := v.curInsts[v.ip+1]
					if nextOp == core.OpReturn || (nextOp == core.OpPop && core.OpReturn == v.curInsts[v.ip+2]) {
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] = v.stack[v.sp-numArgs+p]
						}
						v.sp -= numArgs + 1
						v.ip = -1 // reset IP to beginning of the frame
						continue
					}
				}
				if v.framesIndex >= MaxFrames {
					v.err = errs.ErrStackOverflow
					return
				}

				// update call frame
				v.curFrame.ip = v.ip // store current ip before call
				v.curFrame = &v.frames[v.framesIndex]
				v.curFrame.fn = callee
				v.curFrame.freeVars = callee.Free
				v.curFrame.basePointer = v.sp - numArgs
				v.curInsts = callee.Instructions
				v.ip = -1
				v.framesIndex++
				v.sp = v.sp - numArgs + callee.NumLocals

			case core.VT_BUILTIN_FUNCTION: // fast track for built-in functions
				res, err := core.ToBuiltinFunction(val).Func(v, v.stack[v.sp-numArgs:v.sp])
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

		case core.OpReturn:
			v.ip++
			var res core.Value // default is core.Undefined
			// if operand to return is 1, then return the value in stack, otherwise return undefined
			if v.curInsts[v.ip] == 1 {
				res = v.stack[v.sp-1]
			}
			v.framesIndex--
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip
			v.sp = v.frames[v.framesIndex].basePointer
			v.stack[v.sp-1] = res

		case core.OpGetGlobal:
			v.ip += 2
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			val := v.globals[n]
			v.stack[v.sp] = val
			v.sp++

		case core.OpSetGlobal:
			v.ip += 2
			v.sp--
			n := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			v.globals[n] = v.stack[v.sp]

		case core.OpSetSelGlobal:
			v.ip += 3
			globalIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := v.indexAssign(v.globals[globalIndex], val, selectors)
			if e != nil {
				v.err = e
				return
			}

		case core.OpGetLocal:
			v.ip++
			n := int(v.curInsts[v.ip])
			val := v.stack[v.curFrame.basePointer+n]
			if val.Type == core.VT_VALUE_PTR {
				val = *core.ToValuePtr(val)
			}
			v.stack[v.sp] = val
			v.sp++

		case core.OpSetLocal:
			n := int(v.curInsts[v.ip+1])
			v.ip++
			sp := v.curFrame.basePointer + n

			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			val := v.stack[v.sp-1]
			v.sp--
			if v.stack[sp].Type == core.VT_VALUE_PTR {
				core.ToValuePtr(v.stack[sp]).Set(val)
				val = v.stack[sp]
			}
			v.stack[sp] = val // also use a copy of popped value

		case core.OpDefineLocal:
			v.ip++
			n := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + n

			// local variables can be mutated by other actions
			// so always store the copy of popped value
			val := v.stack[v.sp-1]
			v.sp--
			v.stack[sp] = val

		case core.OpSetSelLocal:
			localIndex := int(v.curInsts[v.ip+1])
			numSelectors := int(v.curInsts[v.ip+2])
			v.ip += 2

			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			dst := v.stack[v.curFrame.basePointer+localIndex]
			if dst.Type == core.VT_VALUE_PTR {
				dst = *core.ToValuePtr(dst)
			}
			if e := v.indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case core.OpGetFreePtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			v.stack[v.sp] = core.ValuePtrValue(v.curFrame.freeVars[n])
			v.sp++

		case core.OpGetFree:
			v.ip++
			n := int(v.curInsts[v.ip])
			v.stack[v.sp] = *v.curFrame.freeVars[n]
			v.sp++

		case core.OpSetFree:
			v.ip++
			n := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[n] = v.stack[v.sp-1]
			v.sp--

		case core.OpGetLocalPtr:
			v.ip++
			n := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + n
			var freeVar *core.Value
			if v.stack[sp].Type == core.VT_VALUE_PTR {
				freeVar = core.ToValuePtr(v.stack[sp])
			} else {
				localVal := v.stack[sp]
				freeVar = &localVal
				v.stack[sp] = core.ValuePtrValue(freeVar)
			}
			v.stack[v.sp] = core.ValuePtrValue(freeVar)
			v.sp++

		case core.OpSetSelFree:
			v.ip += 2
			freeIndex := int(v.curInsts[v.ip-1])
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]core.Value, numSelectors)
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

		case core.OpGetBuiltin:
			v.ip++
			n := int(v.curInsts[v.ip])
			v.stack[v.sp] = BuiltinFuncs[n]
			v.sp++

		case core.OpClosure:
			v.ip += 3
			constIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numFree := int(v.curInsts[v.ip])
			if v.constants[constIndex].Type != core.VT_COMPILED_FUNCTION {
				v.err = fmt.Errorf("not function: %s", v.constants[constIndex].TypeName())
				return
			}
			fn := core.ToCompiledFunction(v.constants[constIndex])
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				if v.stack[v.sp-numFree+i].Type == core.VT_VALUE_PTR {
					free[i] = core.ToValuePtr(v.stack[v.sp-numFree+i])
				} else {
					free[i] = &v.stack[v.sp-numFree+i]
				}
			}
			v.sp -= numFree
			cl := &core.CompiledFunction{
				Instructions:  fn.Instructions,
				NumLocals:     fn.NumLocals,
				NumParameters: fn.NumParameters,
				VarArgs:       fn.VarArgs,
				SourceMap:     fn.SourceMap,
				Free:          free,
			}
			v.stack[v.sp] = core.CompiledFunctionValue(cl)
			v.sp++

		case core.OpIteratorInit:
			dst := v.stack[v.sp-1]
			v.sp--
			if !dst.IsIterable() {
				v.err = fmt.Errorf("not iterable: %s", dst.TypeName())
				return
			}
			it, err := dst.Iterator(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = it
			v.sp++

		case core.OpIteratorNext:
			it := v.stack[v.sp-1]
			v.sp--
			hasMore := it.Next()
			v.stack[v.sp] = core.BoolValue(hasMore)
			v.sp++

		case core.OpIteratorKey:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Key(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case core.OpIteratorValue:
			it := v.stack[v.sp-1]
			v.sp--
			val, err := it.Value(v.alloc)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case core.OpBinaryOp:
			v.ip++
			r := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
			res, e := l.BinaryOp(v.Allocator(), tok, r)
			if e != nil {
				v.sp -= 2
				v.err = e
				return
			}
			v.stack[v.sp-2] = res
			v.sp--

		case core.OpSuspend:
			return

		case core.OpSelect:
			n := v.stack[v.sp-1]
			l := v.stack[v.sp-2]
			v.sp -= 2

			val, err := l.Access(v, n, core.OpSelect)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case core.OpMethodCall:
			operands, read := core.ReadOperands(core.OpcodeOperands[core.OpMethodCall], v.curInsts[v.ip+1:])
			methodConstIdx := operands[0]
			numArgs := operands[1]
			spread := operands[2]
			v.ip += read

			if methodConstIdx < 0 || methodConstIdx >= len(v.constants) {
				v.err = fmt.Errorf("invalid method constant index: %d", methodConstIdx)
				return
			}

			receiver := v.stack[v.sp-1-numArgs]

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				switch arg.Type {
				case core.VT_ARRAY:
					o := (*core.Array)(arg.Ptr)
					for _, item := range o.Elements {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(o.Elements) - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arg.TypeName())
					return
				}
				receiver = v.stack[v.sp-1-numArgs]
			}

			methodConst := v.constants[methodConstIdx]
			methodName, ok := methodConst.AsString()
			if !ok {
				v.err = fmt.Errorf("invalid method name constant type: %s", methodConst.TypeName())
				return
			}

			ret, err := receiver.MethodCall(v, methodName, v.stack[v.sp-numArgs:v.sp])
			v.sp -= numArgs + 1

			if err != nil {
				v.err = err
				return
			}

			v.stack[v.sp] = ret
			v.sp++

		default:
			v.err = fmt.Errorf("unknown opcode: %d", v.curInsts[v.ip])
			return
		}
	}
}

func (v *VM) indexAssign(dst, src core.Value, selectors []core.Value) error {
	numSel := len(selectors)
	for si := numSel - 1; si > 0; si-- {
		next, err := dst.Access(v, selectors[si], core.OpIndex)
		if err != nil {
			return err
		}
		dst = next
	}
	return dst.Assign(selectors[0], src)
}

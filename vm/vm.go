package vm

import (
	"fmt"
	"sync/atomic"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
	"github.com/jokruger/gs/value"
)

var (
	callbackTrampolineInstructions = [...]byte{parser.OpSuspend}
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
type VM struct {
	// Dispatch state
	ip       int    // instruction pointer into curInsts
	sp       int    // stack pointer (index of next free slot)
	curInsts []byte // instructions of the current frame
	curFrame *frame // frame currently being executed
	aborting int64  // non-zero to abort execution; checked atomically each loop

	// Runtime state
	constants   []core.Value   // constant pool used by OpConstant, method dispatch, closures, and other opcode operands
	globals     []core.Value   // global variable storage used by global load/store/select opcodes
	alloc       core.Allocator // object allocator used by arrays, records, iterators, errors, closures, and call helpers
	allocs      int64          // remaining allocation budget; decremented whenever a new object-like value is created
	maxAllocs   int64          // configured allocation budget; copied into allocs at the start of Run
	framesIndex int            // number of active frames; updated on calls, returns, and synthetic callback frames

	// Cold diagnostic state: only used when execution aborts or a stack trace is formatted.
	fileSet *parser.SourceFileSet // source positions for runtime stack traces
	err     error                 // last runtime error captured by run()

	// Large fixed-size arrays
	frames [MaxFrames]frame      // call frame stack
	stack  [StackSize]core.Value // operand stack
}

// NewVM creates a VM.
func NewVM(alloc core.Allocator, bytecode *Bytecode, globals []core.Value, maxAllocs int64) *VM {
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
		maxAllocs:   maxAllocs,
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
	atomic.StoreInt64(&v.aborting, 1)
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
		if numArgs < fn.NumParameters-1 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("call", fmt.Sprintf("at least %d", fn.NumParameters-1), numArgs)
		}
		realArgs := fn.NumParameters - 1
		varArgs := numArgs - realArgs
		if varArgs >= 0 {
			newArgs := make([]core.Value, realArgs+1)
			copy(newArgs, args[:realArgs])
			newArgs[realArgs] = v.alloc.NewArrayValue(args[realArgs:], true)
			args = newArgs
			numArgs = realArgs + 1
		}
	} else if numArgs != fn.NumParameters {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("call", fmt.Sprintf("%d", fn.NumParameters), numArgs)
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
		v.err = core.NewStackOverflowError("native callback frames")
		return core.UndefinedValue(), v.err
	}
	if v.sp+1+numArgs > StackSize {
		v.err = core.ErrStackOverflow
		return core.UndefinedValue(), v.err
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
	v.allocs = v.maxAllocs + 1

	v.run()
	atomic.StoreInt64(&v.aborting, 0)
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
	for atomic.LoadInt64(&v.aborting) == 0 {
		v.ip++
		code := v.curInsts[v.ip]

		switch code {
		case parser.OpConstant:
			v.ip += 2
			cidx := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			v.stack[v.sp] = v.constants[cidx]
			v.sp++

		case parser.OpNull:
			v.stack[v.sp] = core.UndefinedValue()
			v.sp++

		case parser.OpBinaryOp:
			v.ip++
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
			res, e := left.BinaryOp(v, tok, right)
			if e != nil {
				v.sp -= 2
				v.err = e
				return
			}

			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}

			v.stack[v.sp-2] = res
			v.sp--

		case parser.OpEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(left.Equals(right))
			v.sp++

		case parser.OpNotEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.BoolValue(!left.Equals(right))
			v.sp++

		case parser.OpPop:
			v.sp--

		case parser.OpTrue:
			v.stack[v.sp] = core.BoolValue(true)
			v.sp++

		case parser.OpFalse:
			v.stack[v.sp] = core.BoolValue(false)
			v.sp++

		case parser.OpLNot:
			operand := v.stack[v.sp-1]
			v.sp--
			v.stack[v.sp] = core.BoolValue(operand.IsFalse())
			v.sp++

		case parser.OpBComplement:
			operand := v.stack[v.sp-1]
			v.sp--

			switch {
			case operand.IsInt():
				res := core.IntValue(^operand.Int())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: ^%s", operand.TypeName())
				return
			}

		case parser.OpMinus:
			operand := v.stack[v.sp-1]
			v.sp--

			switch {
			case operand.IsInt():
				res := core.IntValue(-operand.Int())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			case operand.IsFloat():
				res := core.FloatValue(-operand.Float())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: -%s", operand.TypeName())
				return
			}

		case parser.OpJumpFalsy:
			v.ip += 4
			v.sp--
			if v.stack[v.sp].IsFalse() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			}

		case parser.OpAndJump:
			v.ip += 4
			if v.stack[v.sp-1].IsFalse() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			} else {
				v.sp--
			}

		case parser.OpOrJump:
			v.ip += 4
			if v.stack[v.sp-1].IsFalse() {
				v.sp--
			} else {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			}

		case parser.OpJump:
			pos := int(v.curInsts[v.ip+4]) | int(v.curInsts[v.ip+3])<<8 | int(v.curInsts[v.ip+2])<<16 | int(v.curInsts[v.ip+1])<<24
			v.ip = pos - 1

		case parser.OpSetGlobal:
			v.ip += 2
			v.sp--
			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			v.globals[globalIndex] = v.stack[v.sp]

		case parser.OpSetSelGlobal:
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

		case parser.OpGetGlobal:
			v.ip += 2
			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			val := v.globals[globalIndex]
			v.stack[v.sp] = val
			v.sp++

		case parser.OpArray:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			elements := make([]core.Value, 0, numElements)
			for i := v.sp - numElements; i < v.sp; i++ {
				elements = append(elements, v.stack[i])
			}
			v.sp -= numElements

			arr := v.alloc.NewArray(elements, false)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = core.ObjectValue(arr)
			v.sp++

		case parser.OpRecord:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]core.Value, numElements)
			for i := v.sp - numElements; i < v.sp; i += 2 {
				key, ok := v.stack[i].AsString()
				if !ok {
					v.err = fmt.Errorf("record keys must be strings, got: %s", v.stack[i].TypeName())
					return
				}
				val := v.stack[i+1]
				kv[key] = val
			}
			v.sp -= numElements

			m := v.alloc.NewRecord(kv, false)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = core.ObjectValue(m)
			v.sp++

		case parser.OpError:
			val := v.stack[v.sp-1]
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp-1] = v.alloc.NewErrorValue(val)

		case parser.OpImmutable:
			val := v.stack[v.sp-1]
			if val.Kind() == core.V_OBJECT {
				switch val := val.Object().(type) {
				case *value.Array:
					t := v.alloc.NewArray(val.Value(), true)
					v.allocs--
					if v.allocs == 0 {
						v.err = core.ErrObjectAllocLimit
						return
					}
					v.stack[v.sp-1] = core.ObjectValue(t)
				case *value.Record:
					v.allocs--
					if v.allocs == 0 {
						v.err = core.ErrObjectAllocLimit
						return
					}
					v.stack[v.sp-1] = v.alloc.NewRecordValue(val.Value(), true)
				case *value.Map:
					v.allocs--
					if v.allocs == 0 {
						v.err = core.ErrObjectAllocLimit
						return
					}
					v.stack[v.sp-1] = v.alloc.NewMapValue(val.Value(), true)
				}
			}

		case parser.OpIndex, parser.OpSelect:
			index := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			val, err := left.Access(v, index, code)
			if err != nil {
				v.err = err
				return
			}
			v.stack[v.sp] = val
			v.sp++

		case parser.OpSliceIndex:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			left := v.stack[v.sp-3]
			v.sp -= 3

			var lowIdx int64
			if !low.IsUndefined() {
				if lowInt, ok := low.AsInt(); ok {
					lowIdx = lowInt
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", low.TypeName())
					return
				}
			}

			if left.Kind() != core.V_OBJECT {
				v.err = fmt.Errorf("not indexable: %s", left.TypeName())
				return
			}

			switch left := left.Object().(type) {
			case *value.Array:
				numElements := int64(left.Len())
				var highIdx int64
				if high.IsUndefined() {
					highIdx = numElements
				} else if highInt, ok := high.AsInt(); ok {
					highIdx = highInt
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				val := v.alloc.NewArray(left.Slice(int(lowIdx), int(highIdx)), false)
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = core.ObjectValue(val)
				v.sp++
			case *value.String:
				numElements := int64(left.Len())
				var highIdx int64
				if high.IsUndefined() {
					highIdx = numElements
				} else if highInt, ok := high.AsInt(); ok {
					highIdx = highInt
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				val := v.alloc.NewString(left.Substring(int(lowIdx), int(highIdx)))
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = core.ObjectValue(val)
				v.sp++
			case *value.Bytes:
				numElements := int64(left.Len())
				var highIdx int64
				if high.IsUndefined() {
					highIdx = numElements
				} else if highInt, ok := high.AsInt(); ok {
					highIdx = highInt
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				val := v.alloc.NewBytes(left.Slice(int(lowIdx), int(highIdx)))
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = core.ObjectValue(val)
				v.sp++
			default:
				v.err = fmt.Errorf("not indexable: %s", left.TypeName())
				return
			}

		case parser.OpCall:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			val := v.stack[v.sp-1-numArgs]
			if !val.IsCallable() {
				v.err = fmt.Errorf("not callable: %s", val.TypeName())
				return
			}

			if spread == 1 {
				v.sp--
				arg := v.stack[v.sp]
				if !arg.IsObject() {
					v.err = fmt.Errorf("spread operator requires an array, got: %s", arg.TypeName())
					return
				}
				switch arr := arg.Object().(type) {
				case *value.Array:
					for _, item := range arr.Value() {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += arr.Len() - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arr.TypeName())
					return
				}
			}

			switch {
			case val.IsCompiledFunction():
				callee := val.CompiledFunction()

				if callee.VarArgs {
					// if the closure is variadic, roll up all variadic parameters into an array
					realArgs := callee.NumParameters - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := make([]core.Value, varArgs)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						v.stack[spStart] = v.alloc.NewArrayValue(args, true)
						v.sp = spStart + 1
					}
				}
				if numArgs != callee.NumParameters {
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
					if nextOp == parser.OpReturn || (nextOp == parser.OpPop && parser.OpReturn == v.curInsts[v.ip+2]) {
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] = v.stack[v.sp-numArgs+p]
						}
						v.sp -= numArgs + 1
						v.ip = -1 // reset IP to beginning of the frame
						continue
					}
				}
				if v.framesIndex >= MaxFrames {
					v.err = core.ErrStackOverflow
					return
				}

				// update call frame
				v.curFrame.ip = v.ip // store current ip before call
				v.curFrame = &(v.frames[v.framesIndex])
				v.curFrame.fn = callee
				v.curFrame.freeVars = callee.Free
				v.curFrame.basePointer = v.sp - numArgs
				v.curInsts = callee.Instructions
				v.ip = -1
				v.framesIndex++
				v.sp = v.sp - numArgs + callee.NumLocals

			default:
				ret, e := val.Call(v, v.stack[v.sp-numArgs:v.sp])
				v.sp -= numArgs + 1

				// runtime error
				if e != nil {
					v.err = e
					return
				}

				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = ret
				v.sp++
			}

		case parser.OpMethodCall:
			operands, read := parser.ReadOperands(parser.OpcodeOperands[code], v.curInsts[v.ip+1:])
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
				if !arg.IsObject() {
					v.err = fmt.Errorf("spread operator requires an array, got: %s", arg.TypeName())
					return
				}
				switch arr := arg.Object().(type) {
				case *value.Array:
					for _, item := range arr.Value() {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += arr.Len() - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arr.TypeName())
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

			ret, err := receiver.Method(v, methodName, v.stack[v.sp-numArgs:v.sp])
			v.sp -= numArgs + 1

			if err != nil {
				v.err = err
				return
			}

			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = ret
			v.sp++

		case parser.OpReturn:
			v.ip++
			var retVal core.Value
			if int(v.curInsts[v.ip]) == 1 {
				retVal = v.stack[v.sp-1]
			} else {
				retVal = core.UndefinedValue()
			}
			//v.sp--
			v.framesIndex--
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip
			//v.sp = lastFrame.basePointer - 1
			v.sp = v.frames[v.framesIndex].basePointer
			// skip stack overflow check because (newSP) <= (oldSP)
			v.stack[v.sp-1] = retVal
			//v.sp++

		case parser.OpDefineLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + localIndex

			// local variables can be mutated by other actions
			// so always store the copy of popped value
			val := v.stack[v.sp-1]
			v.sp--
			v.stack[sp] = val

		case parser.OpSetLocal:
			localIndex := int(v.curInsts[v.ip+1])
			v.ip++
			sp := v.curFrame.basePointer + localIndex

			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			val := v.stack[v.sp-1]
			v.sp--
			if v.stack[sp].IsValuePtr() {
				v.stack[sp].ValuePtr().Set(val)
				val = v.stack[sp]
			}
			v.stack[sp] = val // also use a copy of popped value

		case parser.OpSetSelLocal:
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
			if dst.IsValuePtr() {
				dst = *dst.ValuePtr()
			}
			if e := v.indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case parser.OpGetLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			val := v.stack[v.curFrame.basePointer+localIndex]
			if val.IsValuePtr() {
				val = *val.ValuePtr()
			}
			v.stack[v.sp] = val
			v.sp++

		case parser.OpGetBuiltin:
			v.ip++
			builtinIndex := int(v.curInsts[v.ip])
			v.stack[v.sp] = BuiltinFuncs[builtinIndex]
			v.sp++

		case parser.OpClosure:
			v.ip += 3
			constIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numFree := int(v.curInsts[v.ip])
			if !v.constants[constIndex].IsCompiledFunction() {
				v.err = fmt.Errorf("not function: %s", v.constants[constIndex].TypeName())
				return
			}
			fn := v.constants[constIndex].CompiledFunction()
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				if v.stack[v.sp-numFree+i].IsValuePtr() {
					free[i] = v.stack[v.sp-numFree+i].ValuePtr()
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
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = core.CompiledFunctionValue(cl)
			v.sp++

		case parser.OpGetFreePtr:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			v.stack[v.sp] = core.ValuePtrValue(v.curFrame.freeVars[freeIndex])
			v.sp++

		case parser.OpGetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			v.stack[v.sp] = *v.curFrame.freeVars[freeIndex]
			v.sp++

		case parser.OpSetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[freeIndex] = v.stack[v.sp-1]
			v.sp--

		case parser.OpGetLocalPtr:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + localIndex
			var freeVar *core.Value
			if v.stack[sp].IsValuePtr() {
				freeVar = v.stack[sp].ValuePtr()
			} else {
				localVal := v.stack[sp]
				freeVar = &localVal
				v.stack[sp] = core.ValuePtrValue(freeVar)
			}
			v.stack[v.sp] = core.ValuePtrValue(freeVar)
			v.sp++

		case parser.OpSetSelFree:
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

		case parser.OpIteratorInit:
			dst := v.stack[v.sp-1]
			v.sp--
			if !dst.IsIterable() {
				v.err = fmt.Errorf("not iterable: %s", dst.TypeName())
				return
			}
			it := dst.Iterate(v.alloc)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = core.IteratorValue(it)
			v.sp++

		case parser.OpIteratorNext:
			it := v.stack[v.sp-1]
			v.sp--
			hasMore := it.Next()
			v.stack[v.sp] = core.BoolValue(hasMore)
			v.sp++

		case parser.OpIteratorKey:
			it := v.stack[v.sp-1]
			v.sp--
			val := it.Key(v.alloc)
			v.stack[v.sp] = val
			v.sp++

		case parser.OpIteratorValue:
			it := v.stack[v.sp-1]
			v.sp--
			val := it.Value(v.alloc)
			v.stack[v.sp] = val
			v.sp++

		case parser.OpSuspend:
			return

		default:
			v.err = fmt.Errorf("unknown opcode: %d", v.curInsts[v.ip])
			return
		}
	}
}

func (v *VM) indexAssign(dst, src core.Value, selectors []core.Value) error {
	numSel := len(selectors)
	for si := numSel - 1; si > 0; si-- {
		next, err := dst.Access(v, selectors[si], parser.OpIndex)
		if err != nil {
			return err
		}
		dst = next
	}
	return dst.Assign(selectors[0], src)
}

package vm

import (
	"fmt"
	"sync/atomic"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
	"github.com/jokruger/gs/value"
)

// frame represents a function call frame.
type frame struct {
	fn          *value.CompiledFunction
	freeVars    []*core.Value
	ip          int
	basePointer int
}

// VM is a virtual machine that executes the bytecode compiled by Compiler.
type VM struct {
	alloc       core.Allocator
	constants   []core.Value
	stack       [StackSize]core.Value
	sp          int
	globals     []core.Value
	fileSet     *parser.SourceFileSet
	frames      [MaxFrames]frame
	framesIndex int
	curFrame    *frame
	curInsts    []byte
	ip          int
	aborting    int64
	maxAllocs   int64
	allocs      int64
	err         error
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
func (v *VM) Call(fn core.Object, args ...core.Value) (core.Value, error) {
	switch f := fn.(type) {
	case *value.CompiledFunction:
		return v.call(f, args...)
	case *value.BuiltinFunction:
		return f.Call(v, args...)
	default:
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("vm.Call", "fn", "callable", fn.TypeName())
	}
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
			v.stack[v.sp] = core.NewUndefined()
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
			v.stack[v.sp] = core.NewBool(left.Equals(right))
			v.sp++

		case parser.OpNotEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			v.stack[v.sp] = core.NewBool(!left.Equals(right))
			v.sp++

		case parser.OpPop:
			v.sp--

		case parser.OpTrue:
			v.stack[v.sp] = core.NewBool(true)
			v.sp++

		case parser.OpFalse:
			v.stack[v.sp] = core.NewBool(false)
			v.sp++

		case parser.OpLNot:
			operand := v.stack[v.sp-1]
			v.sp--
			v.stack[v.sp] = core.NewBool(operand.IsFalse())
			v.sp++

		case parser.OpBComplement:
			operand := v.stack[v.sp-1]
			v.sp--

			switch {
			case operand.IsInt():
				res := core.NewInt(^operand.Int())
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
				res := core.NewInt(-operand.Int())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			case operand.IsFloat():
				res := core.NewFloat(-operand.Float())
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

			var elements []core.Value
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

			v.stack[v.sp] = core.NewObject(arr, false)
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
			v.stack[v.sp] = core.NewObject(m, false)
			v.sp++

		case parser.OpError:
			val := v.stack[v.sp-1]
			e := v.alloc.NewError(val)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp-1] = core.NewObject(e, false)

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
					v.stack[v.sp-1] = core.NewObject(t, false)
				case *value.Record:
					t := v.alloc.NewRecord(val.Value(), true)
					v.allocs--
					if v.allocs == 0 {
						v.err = core.ErrObjectAllocLimit
						return
					}
					v.stack[v.sp-1] = core.NewObject(t, false)
				case *value.Map:
					t := v.alloc.NewMap(val.Value(), true)
					v.allocs--
					if v.allocs == 0 {
						v.err = core.ErrObjectAllocLimit
						return
					}
					v.stack[v.sp-1] = core.NewObject(t, false)
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
				v.stack[v.sp] = core.NewObject(val, false)
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
				v.stack[v.sp] = core.NewObject(val, false)
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
				v.stack[v.sp] = core.NewObject(val, false)
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
				if arg.Kind() != core.V_OBJECT {
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

			if !val.IsObject() {
				v.err = fmt.Errorf("not callable: %s", val.TypeName())
				return
			}

			if callee, ok := val.Object().(*value.CompiledFunction); ok {
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
						v.stack[spStart] = core.NewObject(v.alloc.NewArray(args, true), false)
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
					if nextOp == parser.OpReturn ||
						(nextOp == parser.OpPop &&
							parser.OpReturn == v.curInsts[v.ip+2]) {
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] =
								v.stack[v.sp-numArgs+p]
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
			} else {
				var args []core.Value
				args = append(args, v.stack[v.sp-numArgs:v.sp]...)
				ret, e := val.Call(v, args...)
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

		case parser.OpReturn:
			v.ip++
			var retVal core.Value
			if int(v.curInsts[v.ip]) == 1 {
				retVal = v.stack[v.sp-1]
			} else {
				retVal = core.NewUndefined()
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
			fn, ok := v.constants[constIndex].Object().(*value.CompiledFunction)
			if !ok {
				v.err = fmt.Errorf("not function: %s", fn.TypeName())
				return
			}
			free := make([]*core.Value, numFree)
			for i := 0; i < numFree; i++ {
				if v.stack[v.sp-numFree+i].IsValuePtr() {
					free[i] = v.stack[v.sp-numFree+i].ValuePtr()
				} else {
					free[i] = &v.stack[v.sp-numFree+i]
				}
			}
			v.sp -= numFree
			cl := &value.CompiledFunction{
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
			v.stack[v.sp] = core.NewObject(cl, false)
			v.sp++

		case parser.OpGetFreePtr:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			v.stack[v.sp] = core.NewValuePtr(v.curFrame.freeVars[freeIndex])
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
				v.stack[sp] = core.NewValuePtr(freeVar)
			}
			v.stack[v.sp] = core.NewValuePtr(freeVar)
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
			iterator := dst.Iterate(v.alloc)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = core.NewObject(iterator, false)
			v.sp++

		case parser.OpIteratorNext:
			it := v.stack[v.sp-1]
			v.sp--
			hasMore := it.Next()
			v.stack[v.sp] = core.NewBool(hasMore)
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

func (v *VM) call(fn *value.CompiledFunction, args ...core.Value) (core.Value, error) {
	// Check argument count and roll up variadic args if needed
	numArgs := len(args)
	if fn.VarArgs {
		if numArgs < fn.NumParameters-1 {
			return core.NewUndefined(), core.NewWrongNumArgumentsError("call", fmt.Sprintf("at least %d", fn.NumParameters-1), numArgs)
		}
		realArgs := fn.NumParameters - 1
		varArgs := numArgs - realArgs
		if varArgs >= 0 {
			varArgsArray := make([]core.Value, varArgs)
			copy(varArgsArray, args[realArgs:])
			args = append(args[:realArgs], core.NewObject(v.alloc.NewArray(varArgsArray, true), false))
			numArgs = realArgs + 1
		}
	} else if numArgs != fn.NumParameters {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("call", fmt.Sprintf("%d", fn.NumParameters), numArgs)
	}

	// Save current VM state
	savedFramesIndex := v.framesIndex
	savedSp := v.sp
	savedIp := v.ip
	savedCurFrame := v.curFrame
	savedCurInsts := v.curInsts
	savedErr := v.err

	// Clear error for fresh call
	v.err = nil

	// Check if we have room for frames
	if v.framesIndex >= MaxFrames {
		v.err = core.ErrStackOverflow
		return core.NewUndefined(), v.err
	}

	// Create synthetic trampoline frame with just OpSuspend
	// This acts as the "caller" that the callback will return to
	trampolineFrame := &v.frames[v.framesIndex]
	trampolineFrame.fn = &value.CompiledFunction{
		Instructions:  []byte{parser.OpSuspend},
		NumLocals:     0,
		NumParameters: 0,
		VarArgs:       false,
		SourceMap:     nil,
		Free:          nil,
	}
	trampolineFrame.freeVars = nil
	trampolineFrame.ip = -1 // Will be set to 0 before OpSuspend executes
	trampolineFrame.basePointer = v.sp
	v.framesIndex++

	// Push callee slot (matches normal OpCall stack layout)
	// This is where OpReturn will write the return value
	if v.sp >= StackSize {
		v.err = core.ErrStackOverflow
		return core.NewUndefined(), v.err
	}
	v.stack[v.sp] = core.NewObject(fn, false) // Use the function itself as placeholder
	v.sp++

	// Push arguments onto stack
	if v.sp+numArgs > StackSize {
		v.err = core.ErrStackOverflow
		return core.NewUndefined(), v.err
	}
	for _, arg := range args {
		v.stack[v.sp] = arg
		v.sp++
	}

	// Set up callback frame (similar to OpCall for CompiledFunction)
	v.curFrame = &v.frames[v.framesIndex]
	v.curFrame.fn = fn
	v.curFrame.freeVars = fn.Free
	v.curFrame.basePointer = v.sp - numArgs // Points to first arg (after callee slot)
	v.curFrame.ip = -1
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
	v.framesIndex = savedFramesIndex
	v.sp = savedSp
	v.ip = savedIp
	v.curFrame = savedCurFrame
	v.curInsts = savedCurInsts

	// Preserve error from callback, but restore if no error
	err := v.err
	v.err = savedErr

	return result, err
}

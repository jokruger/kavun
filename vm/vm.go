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
	freeVars    []*value.ObjectPtr
	ip          int
	basePointer int
}

// VM is a virtual machine that executes the bytecode compiled by Compiler.
type VM struct {
	constants   []core.Object
	stack       [StackSize]core.Object
	sp          int
	globals     []core.Object
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
func NewVM(bytecode *Bytecode, globals []core.Object, maxAllocs int64) *VM {
	if globals == nil {
		globals = make([]core.Object, GlobalsSize)
	}
	v := &VM{
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

// Abort aborts the execution.
func (v *VM) Abort() {
	atomic.StoreInt64(&v.aborting, 1)
}

// IsStackEmpty tests if the stack is empty or not.
func (v *VM) IsStackEmpty() bool {
	return v.sp == 0
}

// Call calls a compiled function with the given arguments and returns the result.
func (v *VM) Call(fn core.Object, args ...core.Object) (core.Object, error) {
	switch f := fn.(type) {
	case *value.CompiledFunction:
		return v.call(f, args...)
	case *value.BuiltinFunction:
		return f.Call(v, args...)
	default:
		return nil, core.NewInvalidArgumentTypeError("vm.Call", "fn", "callable", fn)
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
			v.stack[v.sp] = value.UndefinedValue
			v.sp++

		case parser.OpBinaryOp:
			v.ip++
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
			res, e := left.BinaryOp(tok, right)
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
			if left.Equals(right) {
				v.stack[v.sp] = value.TrueValue
			} else {
				v.stack[v.sp] = value.FalseValue
			}
			v.sp++

		case parser.OpNotEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			if left.Equals(right) {
				v.stack[v.sp] = value.FalseValue
			} else {
				v.stack[v.sp] = value.TrueValue
			}
			v.sp++

		case parser.OpPop:
			v.sp--

		case parser.OpTrue:
			v.stack[v.sp] = value.TrueValue
			v.sp++

		case parser.OpFalse:
			v.stack[v.sp] = value.FalseValue
			v.sp++

		case parser.OpLNot:
			operand := v.stack[v.sp-1]
			v.sp--
			if operand.IsFalsy() {
				v.stack[v.sp] = value.TrueValue
			} else {
				v.stack[v.sp] = value.FalseValue
			}
			v.sp++

		case parser.OpBComplement:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *value.Int:
				var res core.Object = value.NewInt(^x.Value())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: ^%s",
					operand.TypeName())
				return
			}

		case parser.OpMinus:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *value.Int:
				var res core.Object = value.NewInt(-x.Value())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			case *value.Float:
				var res core.Object = value.NewFloat(-x.Value())
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: -%s",
					operand.TypeName())
				return
			}

		case parser.OpJumpFalsy:
			v.ip += 4
			v.sp--
			if v.stack[v.sp].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			}

		case parser.OpAndJump:
			v.ip += 4
			if v.stack[v.sp-1].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8 | int(v.curInsts[v.ip-2])<<16 | int(v.curInsts[v.ip-3])<<24
				v.ip = pos - 1
			} else {
				v.sp--
			}

		case parser.OpOrJump:
			v.ip += 4
			if v.stack[v.sp-1].IsFalsy() {
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
			selectors := make([]core.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := indexAssign(v.globals[globalIndex], val, selectors)
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

			var elements []core.Object
			for i := v.sp - numElements; i < v.sp; i++ {
				elements = append(elements, v.stack[i])
			}
			v.sp -= numElements

			var arr core.Object = value.NewArray(elements, false)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = arr
			v.sp++

		case parser.OpRecord:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]core.Object, numElements)
			for i := v.sp - numElements; i < v.sp; i += 2 {
				key := v.stack[i]
				val := v.stack[i+1]
				kv[key.(*value.String).Value()] = val
			}
			v.sp -= numElements

			var m core.Object = value.NewRecord(kv, false)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = m
			v.sp++

		case parser.OpError:
			val := v.stack[v.sp-1]
			var e core.Object = value.NewError(val)
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp-1] = e

		case parser.OpImmutable:
			val := v.stack[v.sp-1]
			switch val := val.(type) {
			case *value.Array:
				var t core.Object = value.NewArray(val.Value(), true)
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp-1] = t
			case *value.Record:
				var t core.Object = value.NewRecord(val.Value(), true)
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp-1] = t
			case *value.Map:
				var t core.Object = value.NewMap(val.Value(), true)
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp-1] = t
			}

		case parser.OpIndex, parser.OpSelect:
			index := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			val, err := left.Access(index, code)
			if err != nil {
				v.err = err
				return
			}
			if val == nil {
				val = value.UndefinedValue
			}
			v.stack[v.sp] = val
			v.sp++

		case parser.OpSliceIndex:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			left := v.stack[v.sp-3]
			v.sp -= 3

			var lowIdx int64
			if low != value.UndefinedValue {
				if lowInt, ok := low.(*value.Int); ok {
					lowIdx = lowInt.Value()
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						low.TypeName())
					return
				}
			}

			switch left := left.(type) {
			case *value.Array:
				numElements := int64(left.Len())
				var highIdx int64
				if high == value.UndefinedValue {
					highIdx = numElements
				} else if highInt, ok := high.(*value.Int); ok {
					highIdx = highInt.Value()
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
				var val core.Object = value.NewArray(left.Slice(int(lowIdx), int(highIdx)), false)
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			case *value.String:
				numElements := int64(left.Len())
				var highIdx int64
				if high == value.UndefinedValue {
					highIdx = numElements
				} else if highInt, ok := high.(*value.Int); ok {
					highIdx = highInt.Value()
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
				var val core.Object = value.NewString(left.Substring(int(lowIdx), int(highIdx)))
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			case *value.Bytes:
				numElements := int64(left.Len())
				var highIdx int64
				if high == value.UndefinedValue {
					highIdx = numElements
				} else if highInt, ok := high.(*value.Int); ok {
					highIdx = highInt.Value()
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
				var val core.Object = value.NewBytes(left.Slice(int(lowIdx), int(highIdx)))
				v.allocs--
				if v.allocs == 0 {
					v.err = core.ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
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
				switch arr := v.stack[v.sp].(type) {
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

			if callee, ok := val.(*value.CompiledFunction); ok {
				if callee.VarArgs {
					// if the closure is variadic, roll up all variadic parameters into an array
					realArgs := callee.NumParameters - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := make([]core.Object, varArgs)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						v.stack[spStart] = value.NewArray(args, false)
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
				var args []core.Object
				args = append(args, v.stack[v.sp-numArgs:v.sp]...)
				ret, e := val.Call(v, args...)
				v.sp -= numArgs + 1

				// runtime error
				if e != nil {
					v.err = e
					return
				}

				// nil return -> undefined
				if ret == nil {
					ret = value.UndefinedValue
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
			var retVal core.Object
			if int(v.curInsts[v.ip]) == 1 {
				retVal = v.stack[v.sp-1]
			} else {
				retVal = value.UndefinedValue
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
			if obj, ok := v.stack[sp].(*value.ObjectPtr); ok {
				*obj.Value = val
				val = obj
			}
			v.stack[sp] = val // also use a copy of popped value

		case parser.OpSetSelLocal:
			localIndex := int(v.curInsts[v.ip+1])
			numSelectors := int(v.curInsts[v.ip+2])
			v.ip += 2

			// selectors and RHS value
			selectors := make([]core.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			dst := v.stack[v.curFrame.basePointer+localIndex]
			if obj, ok := dst.(*value.ObjectPtr); ok {
				dst = *obj.Value
			}
			if e := indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case parser.OpGetLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			val := v.stack[v.curFrame.basePointer+localIndex]
			if obj, ok := val.(*value.ObjectPtr); ok {
				val = *obj.Value
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
			fn, ok := v.constants[constIndex].(*value.CompiledFunction)
			if !ok {
				v.err = fmt.Errorf("not function: %s", fn.TypeName())
				return
			}
			free := make([]*value.ObjectPtr, numFree)
			for i := 0; i < numFree; i++ {
				switch freeVar := (v.stack[v.sp-numFree+i]).(type) {
				case *value.ObjectPtr:
					free[i] = freeVar
				default:
					free[i] = &value.ObjectPtr{Value: &v.stack[v.sp-numFree+i]}
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
			v.stack[v.sp] = cl
			v.sp++

		case parser.OpGetFreePtr:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			val := v.curFrame.freeVars[freeIndex]
			v.stack[v.sp] = val
			v.sp++

		case parser.OpGetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			val := *v.curFrame.freeVars[freeIndex].Value
			v.stack[v.sp] = val
			v.sp++

		case parser.OpSetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[freeIndex].Value = v.stack[v.sp-1]
			v.sp--

		case parser.OpGetLocalPtr:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + localIndex
			val := v.stack[sp]
			var freeVar *value.ObjectPtr
			if obj, ok := val.(*value.ObjectPtr); ok {
				freeVar = obj
			} else {
				freeVar = &value.ObjectPtr{Value: &val}
				v.stack[sp] = freeVar
			}
			v.stack[v.sp] = freeVar
			v.sp++

		case parser.OpSetSelFree:
			v.ip += 2
			freeIndex := int(v.curInsts[v.ip-1])
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]core.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := indexAssign(*v.curFrame.freeVars[freeIndex].Value, val, selectors)
			if e != nil {
				v.err = e
				return
			}

		case parser.OpIteratorInit:
			var iterator core.Object
			dst := v.stack[v.sp-1]
			v.sp--
			if !dst.IsIterable() {
				v.err = fmt.Errorf("not iterable: %s", dst.TypeName())
				return
			}
			iterator = dst.Iterate()
			v.allocs--
			if v.allocs == 0 {
				v.err = core.ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = iterator
			v.sp++

		case parser.OpIteratorNext:
			iterator := v.stack[v.sp-1]
			v.sp--
			hasMore := iterator.(core.Iterator).Next()
			if hasMore {
				v.stack[v.sp] = value.TrueValue
			} else {
				v.stack[v.sp] = value.FalseValue
			}
			v.sp++

		case parser.OpIteratorKey:
			iterator := v.stack[v.sp-1]
			v.sp--
			val := iterator.(core.Iterator).Key()
			v.stack[v.sp] = val
			v.sp++

		case parser.OpIteratorValue:
			iterator := v.stack[v.sp-1]
			v.sp--
			val := iterator.(core.Iterator).Value()
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

func indexAssign(dst, src core.Object, selectors []core.Object) error {
	numSel := len(selectors)
	for sidx := numSel - 1; sidx > 0; sidx-- {
		next, err := dst.Access(selectors[sidx], parser.OpIndex)
		if err != nil {
			return err
		}
		dst = next
	}
	return dst.Assign(selectors[0], src)
}

func (v *VM) call(fn *value.CompiledFunction, args ...core.Object) (core.Object, error) {
	// Check argument count and roll up variadic args if needed
	numArgs := len(args)
	if fn.VarArgs {
		if numArgs < fn.NumParameters-1 {
			return nil, core.NewWrongNumArgumentsError("call", fmt.Sprintf("at least %d", fn.NumParameters-1), numArgs)
		}
		realArgs := fn.NumParameters - 1
		varArgs := numArgs - realArgs
		if varArgs >= 0 {
			varArgsArray := make([]core.Object, varArgs)
			copy(varArgsArray, args[realArgs:])
			args = append(args[:realArgs], value.NewArray(varArgsArray, false))
			numArgs = realArgs + 1
		}
	} else if numArgs != fn.NumParameters {
		return nil, core.NewWrongNumArgumentsError("call", fmt.Sprintf("%d", fn.NumParameters), numArgs)
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
		return nil, v.err
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
		return nil, v.err
	}
	v.stack[v.sp] = fn // Use the function itself as placeholder
	v.sp++

	// Push arguments onto stack
	if v.sp+numArgs > StackSize {
		v.err = core.ErrStackOverflow
		return nil, v.err
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
	var result core.Object
	if v.err == nil {
		// The return value is at savedSp (the callee slot position)
		result = v.stack[savedSp]
		if result == nil {
			result = value.UndefinedValue
		}
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

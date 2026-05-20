package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/errs"
)

type CompiledFunction struct {
	Instructions  []byte
	Free          []*Value
	SourceMap     map[int]Pos
	NumLocals     int // number of local variables (including function parameters)
	MaxStack      int // estimated maximum operand-stack depth which can be reached during execution
	NumParameters int8
	VarArgs       bool
	NamedResult   int8 // local-slot index of function's named result: 0 = no named result, N > 0 means slot N-1
}

func (o *CompiledFunction) Set(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals, maxStack int, numParameters int8, varArgs bool, namedResult int8) {
	o.Instructions = instructions
	o.Free = free
	o.SourceMap = sourceMap
	o.NumLocals = numLocals
	o.MaxStack = maxStack
	o.NumParameters = numParameters
	o.VarArgs = varArgs
	o.NamedResult = namedResult
}

// HasNamedResult reports whether the function declares a named result.
func (o *CompiledFunction) HasNamedResult() bool {
	return o.NamedResult != 0
}

// NamedResultSlot returns the local-slot index of the named result.
// Caller should check HasNamedResult first.
func (o *CompiledFunction) NamedResultSlot() int {
	return int(o.NamedResult) - 1
}

func (o *CompiledFunction) Size() int64 {
	return int64(len(o.Instructions) + len(o.Free) + len(o.SourceMap))
}

func (o *CompiledFunction) SourcePos(ip int) Pos {
	for ip >= 0 {
		if p, ok := o.SourceMap[ip]; ok {
			return p
		}
		ip--
	}
	return NoPos
}

// CompiledFunctionValue creates new boxed compiled function value.
func CompiledFunctionValue(f *CompiledFunction) Value {
	return Value{
		Type:      VT_COMPILED_FUNCTION,
		Immutable: true,
		Ptr:       unsafe.Pointer(f),
	}
}

// NewCompiledFunctionValue creates new (heap-allocated) compiled function value.
func NewCompiledFunctionValue(
	instructions []byte,
	free []*Value,
	sourceMap map[int]Pos,
	numLocals int,
	maxStack int,
	numParameters int8,
	varArgs bool,
	namedResult int8,
) Value {
	f := &CompiledFunction{}
	f.Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
	return CompiledFunctionValue(f)
}

var TypeCompiledFunction = ValueType{
	Name:         compiledFunctionTypeName,
	String:       func(v Value) string { return compiledFunctionTypeName(v) },
	EncodeBinary: compiledFunctionTypeEncodeBinary,
	DecodeBinary: compiledFunctionTypeDecodeBinary,
	IsTrue:       ConstHook(true),
	IsCallable:   ConstHook(true),
	IsVariadic:   func(v Value) bool { return (*CompiledFunction)(v.Ptr).VarArgs },
	Equal:        compiledFunctionTypeEqual,
	Arity:        func(v Value) int8 { return (*CompiledFunction)(v.Ptr).NumParameters },
	Call:         func(v Value, vm VM, args []Value) (Value, error) { return vm.Call((*CompiledFunction)(v.Ptr), args) },
	MethodCall:   compiledFunctionTypeMethodCall,
}

func compiledFunctionTypeEqual(v Value, r Value) bool {
	if r.Type != VT_COMPILED_FUNCTION {
		return false
	}
	a := (*CompiledFunction)(v.Ptr)
	b := (*CompiledFunction)(r.Ptr)
	return a == b
}

func compiledFunctionTypeName(v Value) string {
	o := (*CompiledFunction)(v.Ptr)
	if o.VarArgs {
		return fmt.Sprintf("<compiled-function/%d+>", o.NumParameters)
	}
	return fmt.Sprintf("<compiled-function/%d>", o.NumParameters)
}

func compiledFunctionTypeEncodeBinary(v Value) ([]byte, error) {
	f := (*CompiledFunction)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(f); err != nil {
		return nil, fmt.Errorf("compiled function: %w", err)
	}
	return buf.Bytes(), nil
}

func compiledFunctionTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var f CompiledFunction
	if err := dec.Decode(&f); err != nil {
		return fmt.Errorf("compiled function: %w", err)
	}
	v.Ptr = unsafe.Pointer(&f)
	return nil
}

func compiledFunctionTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

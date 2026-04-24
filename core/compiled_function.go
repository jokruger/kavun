package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"
)

type CompiledFunction struct {
	Instructions  []byte
	Free          []*Value
	SourceMap     map[int]Pos
	NumLocals     int // number of local variables (including function parameters)
	NumParameters int8
	VarArgs       bool
}

func (o *CompiledFunction) Set(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals int, numParameters int8, varArgs bool) {
	o.Instructions = instructions
	o.Free = free
	o.SourceMap = sourceMap
	o.NumLocals = numLocals
	o.NumParameters = numParameters
	o.VarArgs = varArgs
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
		Type:  VT_COMPILED_FUNCTION,
		Const: true,
		Ptr:   unsafe.Pointer(f),
	}
}

// NewCompiledFunctionValue creates new (heap-allocated) compiled function value.
func NewCompiledFunctionValue(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals int, numParameters int8, varArgs bool) Value {
	f := &CompiledFunction{}
	f.Set(instructions, free, sourceMap, numLocals, numParameters, varArgs)
	return CompiledFunctionValue(f)
}

/* CompiledFunction type methods */

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

func compiledFunctionTypeString(v Value) string {
	return compiledFunctionTypeName(v)
}

func compiledFunctionTypeArity(v Value) int8 {
	f := (*CompiledFunction)(v.Ptr)
	return f.NumParameters
}

func compiledFunctionTypeIsVariadic(v Value) bool {
	f := (*CompiledFunction)(v.Ptr)
	return f.VarArgs
}

func compiledFunctionTypeCall(v Value, vm VM, args []Value) (Value, error) {
	return vm.Call((*CompiledFunction)(v.Ptr), args)
}

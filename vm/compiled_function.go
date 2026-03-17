package vm

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

type CompiledFunction struct {
	value.Object
	Instructions  []byte
	NumLocals     int // number of local variables (including function parameters)
	NumParameters int
	VarArgs       bool
	SourceMap     map[int]core.Pos
	Free          []*ObjectPtr
}

func (o *CompiledFunction) TypeName() string {
	return "compiled-function"
}

func (o *CompiledFunction) String() string {
	return "<compiled-function>"
}

func (o *CompiledFunction) Arity() int {
	return o.NumParameters
}

func (o *CompiledFunction) IsVariadic() bool {
	return o.VarArgs
}

func (o *CompiledFunction) Size() int64 {
	return int64(len(o.Instructions) + len(o.SourceMap) + len(o.Free))
}

func (o *CompiledFunction) Copy() core.Object {
	return &CompiledFunction{
		Instructions:  append([]byte{}, o.Instructions...),
		NumLocals:     o.NumLocals,
		NumParameters: o.NumParameters,
		VarArgs:       o.VarArgs,
		Free:          append([]*ObjectPtr{}, o.Free...), // DO NOT Copy() of elements; these are variable pointers
	}
}

func (o *CompiledFunction) Equals(core.Object) bool {
	return false
}

func (o *CompiledFunction) SourcePos(ip int) core.Pos {
	for ip >= 0 {
		if p, ok := o.SourceMap[ip]; ok {
			return p
		}
		ip--
	}
	return core.NoPos
}

func (o *CompiledFunction) IsCallable() bool {
	return true
}

func (o *CompiledFunction) Call(vm core.VM, args ...core.Object) (core.Object, error) {
	return vm.Call(o, args...)
}

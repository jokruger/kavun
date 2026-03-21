package value

import (
	"fmt"

	"github.com/jokruger/gs/core"
)

type CompiledFunction struct {
	Object
	Instructions  []byte
	NumLocals     int // number of local variables (including function parameters)
	NumParameters int
	VarArgs       bool
	SourceMap     map[int]core.Pos
	Free          []*ObjectPtr
}

func (o *CompiledFunction) TypeName() string {
	if o.VarArgs {
		return fmt.Sprintf("<compiled-function/%d+>", o.NumParameters)
	}
	return fmt.Sprintf("<compiled-function/%d>", o.NumParameters)
}

func (o *CompiledFunction) String() string {
	return o.TypeName()
}

func (o *CompiledFunction) Arity() int {
	return o.NumParameters
}

func (o *CompiledFunction) IsVariadic() bool {
	return o.VarArgs
}

func (o *CompiledFunction) IsImmutable() bool {
	return true
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

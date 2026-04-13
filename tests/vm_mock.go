package tests

import (
	al "github.com/jokruger/gs/alloc"
	"github.com/jokruger/gs/core"
)

var Alloc = al.New()
var Vm = &VM{}

type VM struct{}

func (v *VM) Allocator() core.Allocator { return Alloc }
func (v *VM) Abort()                    {}
func (v *VM) IsStackEmpty() bool        { return false }
func (v *VM) Call(*core.CompiledFunction, []core.Value) (core.Value, error) {
	return core.Undefined, nil
}
func (v *VM) Run() error { return nil }

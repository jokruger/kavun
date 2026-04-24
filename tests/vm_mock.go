package tests

import (
	al "github.com/jokruger/kavun/alloc"
	"github.com/jokruger/kavun/core"
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

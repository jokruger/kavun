package tests

import (
	"github.com/jokruger/kavun/core"
)

var Vm = &VM{}

type VM struct{}

func (v *VM) Abort()             {}
func (v *VM) IsStackEmpty() bool { return false }
func (v *VM) Call(core.Value, []core.Value) (core.Value, error) {
	return core.Undefined, nil
}
func (v *VM) Run() error          { return nil }
func (v *VM) Recover() core.Value { return core.Undefined }

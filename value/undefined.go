package value

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Undefined struct {
	Object
}

func (o *Undefined) GobDecode(b []byte) error {
	if len(b) != 0 {
		core.NewDecodeBinarySizeError(o, 0, len(b))
	}
	return nil
}

func (o *Undefined) GobEncode() ([]byte, error) {
	return []byte{}, nil
}

func (o *Undefined) Next() bool {
	return false
}

func (o *Undefined) Key(core.Allocator) core.Object {
	return o
}

func (o *Undefined) Value(core.Allocator) core.Object {
	return o
}

func (o *Undefined) TypeName() string {
	return "undefined"
}

func (o *Undefined) String() string {
	return "undefined"
}

func (o *Undefined) Interface() any {
	return nil
}

func (o *Undefined) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Undefined) Equals(x core.Object) bool {
	return o == x
}

func (o *Undefined) Copy(alloc core.Allocator) core.Object {
	return alloc.NewUndefined()
}

func (o *Undefined) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	return vm.Allocator().NewUndefined(), nil
}

func (o *Undefined) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Undefined) Iterate(core.Allocator) core.Iterator {
	return o
}

func (o *Undefined) IsUndefined() bool {
	return true
}

func (o *Undefined) IsTrue() bool {
	return false
}

func (o *Undefined) IsFalse() bool {
	return true
}

func (o *Undefined) IsIterable() bool {
	return true
}

func (o *Undefined) IsImmutable() bool {
	return false
}

func (o *Undefined) AsBool() (bool, bool) {
	return false, true
}

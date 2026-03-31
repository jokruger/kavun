package value

import (
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Object struct {
}

func (o *Object) TypeName() string {
	return "<object>"
}

func (o *Object) String() string {
	return o.TypeName()
}

func (o *Object) Interface() any {
	return o
}

func (o *Object) Arity() int {
	return 0
}

func (o *Object) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Object) Equals(x core.Object) bool {
	return o == x
}

func (o *Object) Copy(core.Allocator) core.Object {
	return o
}

func (o *Object) Access(core.VM, core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Object) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Object) Iterate(core.Allocator) core.Iterator {
	return nil
}

func (o *Object) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Object) IsUndefined() bool {
	return false
}

func (o *Object) IsTrue() bool {
	return o != nil
}

func (o *Object) IsFalse() bool {
	return o == nil
}

func (o *Object) IsIterable() bool {
	return false
}

func (o *Object) IsCallable() bool {
	return false
}

func (o *Object) IsImmutable() bool {
	return false
}

func (o *Object) IsVariadic() bool {
	return false
}

func (o *Object) AsString() (string, bool) {
	return "", false
}

func (o *Object) AsInt() (int64, bool) {
	return 0, false
}

func (o *Object) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Object) AsBool() (bool, bool) {
	return false, false
}

func (o *Object) AsRune() (rune, bool) {
	return 0, false
}

func (o *Object) AsBytes() ([]byte, bool) {
	return nil, false
}

func (o *Object) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

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

func (o *Object) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Object) Equals(x core.Value) bool {
	if !x.IsObject() {
		return false
	}
	return o == x.Object()
}

func (o *Object) Copy(core.Allocator) core.Value {
	return core.UndefinedValue()
}

func (o *Object) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
}

func (o *Object) Access(core.VM, core.Value, core.Opcode) (core.Value, error) {
	return core.UndefinedValue(), core.NewNotAccessibleError(o.TypeName())
}

func (o *Object) Assign(core.Value, core.Value) error {
	return core.NewNotAssignableError(o.TypeName())
}

func (o *Object) Iterate(core.Allocator) core.Iterator {
	return nil
}

func (o *Object) Call(core.VM, []core.Value) (core.Value, error) {
	return core.UndefinedValue(), nil
}

func (o *Object) IsUndefined() bool {
	return false
}

func (o *Object) IsString() bool {
	return false
}

func (o *Object) IsInt() bool {
	return false
}

func (o *Object) IsFloat() bool {
	return false
}

func (o *Object) IsBool() bool {
	return false
}

func (o *Object) IsChar() bool {
	return false
}

func (o *Object) IsBytes() bool {
	return false
}

func (o *Object) IsTime() bool {
	return false
}

func (o *Object) IsArray() bool {
	return false
}

func (o *Object) IsError() bool {
	return false
}

func (o *Object) IsMap() bool {
	return false
}

func (o *Object) IsRecord() bool {
	return false
}

func (o *Object) IsCompiledFunction() bool {
	return false
}

func (o *Object) IsBuiltinFunction() bool {
	return false
}

func (o *Object) IsImmutable() bool {
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

func (o *Object) AsChar() (rune, bool) {
	return 0, false
}

func (o *Object) AsBytes() ([]byte, bool) {
	return nil, false
}

func (o *Object) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

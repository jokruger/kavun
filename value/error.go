package value

import (
	"errors"
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Error struct {
	Object
	value core.Value
}

func (o *Error) GobDecode(b []byte) error {
	var v core.Value
	err := v.GobDecode(b)
	if err != nil {
		return err
	}
	o.Set(v)
	return nil
}

func (o *Error) GobEncode() ([]byte, error) {
	return o.value.GobEncode()
}

func (o *Error) Set(value core.Value) {
	o.value = value
}

func (o *Error) Value() core.Value {
	return o.value
}

func (o *Error) TypeName() string {
	return "error"
}

func (o *Error) String() string {
	if o.value.IsUndefined() {
		return "error(undefined)"
	}
	return fmt.Sprintf("error(%s)", o.value.String())
}

func (o *Error) Interface() any {
	return errors.New(o.String())
}

func (o *Error) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	return core.NewUndefined(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Error) Equals(x core.Value) bool {
	if !x.IsError() {
		return false
	}
	return o.value.Equals(x.Object().(*Error).value)
}

func (o *Error) Copy(alloc core.Allocator) core.Value {
	return alloc.NewErrorValue(o.value.Copy(alloc))
}

func (o *Error) Access(vm core.VM, index core.Value, mode core.Opcode) (core.Value, error) {
	k, ok := index.AsString()
	if !ok {
		return core.NewUndefined(), core.NewInvalidIndexTypeError("error access", "string", index.TypeName())
	}

	switch k {
	case "value":
		return o.value, nil
	default:
		return core.NewUndefined(), core.NewInvalidSelectorError(o.TypeName(), k)
	}
}

func (o *Error) Assign(core.Value, core.Value) error {
	return core.NewNotAssignableError(o.TypeName())
}

func (o *Error) IsError() bool {
	return true
}

func (o *Error) IsTrue() bool {
	return false // error must be always false.
}

func (o *Error) IsFalse() bool {
	return true // error must be always false.
}

func (o *Error) AsString() (string, bool) {
	s, ok := o.value.AsString()
	if ok {
		return s, true
	}
	return "runtime error", true
}

func (o *Error) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

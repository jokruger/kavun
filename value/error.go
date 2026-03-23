package value

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Error struct {
	Object
	value core.Object
}

func (o *Error) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var hasValue bool
	if err := dec.Decode(&hasValue); err != nil {
		return err
	}
	if !hasValue {
		o.Set(nil)
		return nil
	}

	var v core.Object
	if err := dec.Decode(&v); err != nil {
		return err
	}
	o.Set(v)
	return nil
}

func (o *Error) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	hasValue := o.value != nil
	if err := enc.Encode(hasValue); err != nil {
		return nil, err
	}
	if hasValue {
		if err := enc.Encode(&o.value); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (o *Error) Set(value core.Object) {
	o.value = value
}

func (o *Error) Value() core.Object {
	return o.value
}

func (o *Error) TypeName() string {
	return "error"
}

func (o *Error) String() string {
	if o.value == nil {
		return "error(undefined)"
	}
	return fmt.Sprintf("error(%s)", o.value.String())
}

func (o *Error) Interface() any {
	return errors.New(o.String())
}

func (o *Error) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Error) Equals(x core.Object) bool {
	if other, ok := x.(*Error); ok {
		if o.value == nil && other.value == nil {
			return true
		}
		return o.value.Equals(other.value)
	}
	return false
}

func (o *Error) Copy(alloc core.Allocator) core.Object {
	if o.value == nil {
		return alloc.NewError(nil)
	}
	return alloc.NewError(o.value.Copy(alloc))
}

func (o *Error) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	alloc := vm.Allocator()

	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("error access", "string", index)
	}

	switch k {
	case "value":
		if o.value == nil {
			return alloc.NewUndefined(), nil
		}
		return o.value.Copy(alloc), nil
	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *Error) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Error) IsTrue() bool {
	return false // error is always false.
}

func (o *Error) IsFalse() bool {
	return true // error is always false.
}

func (o *Error) IsImmutable() bool {
	return true
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

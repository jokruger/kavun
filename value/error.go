package value

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Error struct {
	value core.Object
}

func NewError(value core.Object) *Error {
	o := &Error{}
	o.Set(value)
	return o
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
	if o.value != nil {
		return fmt.Sprintf("error: %s", o.value.String())
	}
	return "error"
}

func (o *Error) Interface() any {
	return errors.New(o.String())
}

func (o *Error) Arity() int {
	return 0
}

func (o *Error) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Error) Equals(x core.Object) bool {
	return o == x
}

func (o *Error) Copy() core.Object {
	return NewError(o.value.Copy())
}

func (o *Error) Access(index core.Object, mode core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("error access", "string", index)
	}

	switch k {
	case "value":
		return o.value, nil
	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *Error) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Error) Iterate() core.Iterator {
	return nil
}

func (o *Error) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Error) IsFalsy() bool {
	return true // error is always false.
}

func (o *Error) IsIterable() bool {
	return false
}

func (o *Error) IsCallable() bool {
	return false
}

func (o *Error) IsImmutable() bool {
	return true
}

func (o *Error) IsVariadic() bool {
	return false
}

func (o *Error) AsString() (string, bool) {
	return o.String(), true
}

func (o *Error) AsInt() (int64, bool) {
	return 0, false
}

func (o *Error) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Error) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Error) AsRune() (rune, bool) {
	return 0, false
}

func (o *Error) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Error) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

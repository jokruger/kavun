package value

import (
	"errors"
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
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

func (o *Error) BinaryOp(token.Token, core.Object) (core.Object, error) {
	return nil, gse.ErrInvalidOperator
}

func (o *Error) Equals(x core.Object) bool {
	return o == x
}

func (o *Error) Copy() core.Object {
	return NewError(o.value.Copy())
}

func (o *Error) IndexGet(index core.Object) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, gse.ErrInvalidIndexOnError
	}

	switch k {
	case "value":
		return o.value, nil
	default:
		return nil, gse.ErrInvalidIndexOnError
	}
}

func (o *Error) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
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
	return false
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

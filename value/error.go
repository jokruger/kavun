package value

import (
	"errors"
	"fmt"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
)

type Error struct {
	ObjectImpl
	Value core.Object
}

func (o *Error) TypeName() string {
	return "error"
}

func (o *Error) String() string {
	if o.Value != nil {
		return fmt.Sprintf("error: %s", o.Value.String())
	}
	return "error"
}

func (o *Error) IsFalsy() bool {
	return true // error is always false.
}

func (o *Error) Copy() core.Object {
	return &Error{Value: o.Value.Copy()}
}

func (o *Error) Equals(x core.Object) bool {
	return o == x // pointer equality
}

func (o *Error) IndexGet(index core.Object) (res core.Object, err error) {
	if strIdx, _ := index.ToString(); strIdx != "value" {
		err = gse.ErrInvalidIndexOnError
		return
	}
	res = o.Value
	return
}

func (o *Error) ToString() (string, bool) {
	return o.String(), true
}

func (o *Error) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Error) ToInterface() any {
	return errors.New(o.String())
}

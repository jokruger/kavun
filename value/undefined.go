package value

import "github.com/jokruger/gs/core"

type Undefined struct {
	ObjectImpl
}

func (o *Undefined) TypeName() string {
	return "undefined"
}

func (o *Undefined) String() string {
	return "<undefined>"
}

func (o *Undefined) Copy() core.Object {
	return o
}

func (o *Undefined) IsFalsy() bool {
	return true
}

func (o *Undefined) Equals(x core.Object) bool {
	return o == x
}

func (o *Undefined) IndexGet(core.Object) (core.Object, error) {
	return UndefinedValue, nil
}

func (o *Undefined) Iterate() core.Iterator {
	return o
}

func (o *Undefined) CanIterate() bool {
	return true
}

func (o *Undefined) Next() bool {
	return false
}

func (o *Undefined) Key() core.Object {
	return o
}

func (o *Undefined) Value() core.Object {
	return o
}

func (o *Undefined) ToBool() (bool, bool) {
	return false, true
}

func (o *Undefined) ToInterface() any {
	return nil
}

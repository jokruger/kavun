package value

import "github.com/jokruger/gs/core"

type ObjectPtr struct {
	ObjectImpl
	Value *core.Object
}

func (o *ObjectPtr) TypeName() string {
	return "<free-var>"
}

func (o *ObjectPtr) String() string {
	return "free-var"
}

func (o *ObjectPtr) Copy() core.Object {
	return o
}

func (o *ObjectPtr) IsFalsy() bool {
	return o.Value == nil
}

func (o *ObjectPtr) Equals(x core.Object) bool {
	return o == x
}

func (o *ObjectPtr) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *ObjectPtr) ToInterface() any {
	return o.Value
}

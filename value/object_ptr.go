package value

import (
	"github.com/jokruger/gs/core"
)

/*
	ObjectPtr is different from other objects as it is created and managed by compiler and VM directly.
*/

type ObjectPtr struct {
	Object
	Value *core.Object
}

func (o *ObjectPtr) TypeName() string {
	return "<free-var>"
}

func (o *ObjectPtr) String() string {
	return "free-var"
}

func (o *ObjectPtr) IsTrue() bool {
	return o.Value != nil
}

func (o *ObjectPtr) IsFalse() bool {
	return o.Value == nil
}

func (o *ObjectPtr) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

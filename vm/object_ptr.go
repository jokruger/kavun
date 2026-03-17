package vm

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

type ObjectPtr struct {
	value.Object
	Value *core.Object
}

func (o *ObjectPtr) TypeName() string {
	return "<free-var>"
}

func (o *ObjectPtr) String() string {
	return "free-var"
}

func (o *ObjectPtr) IsFalsy() bool {
	return o.Value == nil
}

func (o *ObjectPtr) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

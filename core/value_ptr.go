package core

import (
	"fmt"
	"unsafe"
)

// ValuePtrValue creates new boxed value pointer value.
func ValuePtrValue(p *Value) Value {
	return Value{
		Type: VT_VALUE_PTR,
		Ptr:  unsafe.Pointer(p),
	}
}

// TypeValuePtr is the value ptr type descriptor.
var TypeValuePtr = ValueType{
	Name: func(v Value) string {
		return fmt.Sprintf("<value_ptr:%s>", v.TypeName())
	},
}

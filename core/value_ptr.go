package core

import (
	"fmt"
	"unsafe"
)

// ValuePtrValue creates new boxed value pointer value.
func ValuePtrValue(p *Value) Value {
	return Value{
		Ptr:  unsafe.Pointer(p),
		Type: VT_VALUE_PTR,
	}
}

// ToValuePtr converts boxed value pointer value to *Value. It is a caller's responsibility to ensure the type is correct.
func ToValuePtr(v Value) *Value {
	return (*Value)(v.Ptr)
}

/* ValuePtr type methods */

func valuePtrTypeName(v Value) string {
	return fmt.Sprintf("<value_ptr:%s>", v.TypeName())
}

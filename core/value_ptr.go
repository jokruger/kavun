package core

import (
	"fmt"
	"unsafe"
)

func ValuePtrValue(p *Value) Value {
	return Value{
		Ptr:  unsafe.Pointer(p),
		Type: VT_VALUE_PTR,
	}
}

func toValuePtr(v Value) *Value {
	return (*Value)(v.Ptr)
}

func valuePtrTypeName(v Value) string {
	return fmt.Sprintf("<value_ptr:%s>", v.TypeName())
}

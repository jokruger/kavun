package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

const valuePtrTypeName = "value-ptr"

func NewValuePtrValue(p *Value) Value {
	return Value{
		Type: value.ValuePtr,
		Ptr:  unsafe.Pointer(p),
	}
}

var TypeValuePtr = ValueTypeDescr{
	Name: func(v Value) string { return fmt.Sprintf("<%s:%s>", valuePtrTypeName, v.TypeName()) }, // PURE by contract
}

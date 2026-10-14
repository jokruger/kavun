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
	Name: valuePtrName, // PURE by contract
}

// valuePtrName names the POINTEE: naming the pointer value itself would recurse without end (a Go stack
// overflow no recover catches), and the name is read on error paths that must never make things worse.
func valuePtrName(v Value) string {
	p := (*Value)(v.Ptr)
	if p == nil {
		return fmt.Sprintf("<%s:nil>", valuePtrTypeName)
	}
	return fmt.Sprintf("<%s:%s>", valuePtrTypeName, p.TypeName())
}

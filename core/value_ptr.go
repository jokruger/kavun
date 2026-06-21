package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

const valuePtrTypeName = "value-ptr"

func (a *Arena) MustNewValuePtrValue(p *Value) Value {
	v, err := a.NewValuePtrValue(p)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewValuePtrValue(p *Value) (Value, error) {
	if ref, poolPtr, ok := a.arena.New(value.ValuePtr); ok {
		a.PinAny(*p) // mark pointed value as unmanaged because it's now also owned by the pointer value
		*(**Value)(poolPtr) = p
		return Value{Type: value.ValuePtr, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(valuePtrTypeName)
}

// TypeValuePtr is the value ptr type descriptor.
var TypeValuePtr = ValueTypeDescr{
	Name: func(v Value) string {
		return fmt.Sprintf("<%s:%s>", valuePtrTypeName, v.TypeName())
	},
}

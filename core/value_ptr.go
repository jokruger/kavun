package core

import (
	"fmt"
)

// TypeValuePtr is the value ptr type descriptor.
var TypeValuePtr = ValueTypeDescr{
	Pin:     func(a *Arena, v Value) { a.PinValuePtrValue(v) },
	Retain:  func(a *Arena, v Value) { a.RetainValuePtrValue(v) },
	Release: func(a *Arena, v Value) { a.ReleaseValuePtrValue(v) },
	Name: func(a *Arena, v Value) string {
		return fmt.Sprintf("<value_ptr:%s>", v.TypeName(a))
	},
}

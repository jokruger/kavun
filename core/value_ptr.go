package core

import (
	"fmt"
)

const valuePtrTypeName = "value-ptr"

// TypeValuePtr is the value ptr type descriptor.
var TypeValuePtr = ValueTypeDescr{
	Name: func(a *Arena, v Value) string {
		return fmt.Sprintf("<%s:%s>", valuePtrTypeName, v.TypeName(a))
	},
}

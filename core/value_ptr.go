package core

import (
	"fmt"
)

// TypeValuePtr is the value ptr type descriptor.
var TypeValuePtr = ValueType{
	Name: func(a *Arena, v Value) string {
		return fmt.Sprintf("<value_ptr:%s>", v.TypeName(a))
	},
}

package vm

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

// CountObjects returns the number of objects that a given object o contains.
// For scalar value types, it will always be 1. For compound value types,
// this will include its elements and all of their elements recursively.
func CountObjects(o core.Object) (c int) {
	c = 1
	switch o := o.(type) {
	case *value.Array:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.ImmutableArray:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.Map:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.ImmutableMap:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.Error:
		c += CountObjects(o.Value)
	}
	return
}

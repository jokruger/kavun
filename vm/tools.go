package vm

import (
	"github.com/jokruger/gs/core"
)

// CountObjects returns the number of objects that a given object o contains.
// For scalar value types, it will always be 1. For compound value types, this will include its elements and all of their elements recursively.
func CountObjects(o core.Value) (c int) {
	c = 1

	switch o.Type {
	case core.VT_ARRAY:
		o := core.ToArray(o)
		for _, v := range o.Elements {
			c += CountObjects(v)
		}

	case core.VT_RECORD:
		o := core.ToRecord(o)
		for _, v := range o.Elements {
			c += CountObjects(v)
		}

	case core.VT_MAP:
		o := core.ToMap(o)
		for _, v := range o.Elements {
			c += CountObjects(v)
		}

	case core.VT_ERROR:
		o := core.ToError(o)
		c += CountObjects(o.Payload)
	}

	return
}

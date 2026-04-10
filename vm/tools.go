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
		o := (*core.Array)(o.Ptr)
		for _, v := range o.Value() {
			c += CountObjects(v)
		}

	case core.VT_RECORD:
		o := (*core.Record)(o.Ptr)
		for _, v := range o.Value() {
			c += CountObjects(v)
		}

	case core.VT_MAP:
		o := (*core.Map)(o.Ptr)
		for _, v := range o.Value() {
			c += CountObjects(v)
		}

	case core.VT_ERROR:
		o := (*core.Error)(o.Ptr)
		c += CountObjects(o.Value())
	}

	return
}

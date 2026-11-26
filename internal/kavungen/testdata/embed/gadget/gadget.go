// Package gadget is an embedder type: a whole number with members of the embedder's own names.
package gadget

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
)

// TypeGadget is the gadget type's id, in the embedder range.
const TypeGadget = value.FirstUserDefinedType + 8

// New answers a gadget holding n.
func New(n int64) core.Value {
	return core.Value{Type: TypeGadget, Data: uint64(n)}
}

// A whole number, held by the embedder; render it with [`string`](kavun:type/string) text.
//
// Its members are the embedder's own names beside the builtin ones: see [`double`](#double).
//
//kavun:type gadget

// init registers the type. It runs before the init functions of kavun_gen.go, whose name sorts after this file's: the
// member names its table uses are bound earlier still, while the package's variables are initialized.
func init() {
	core.SetHostVersion(1, 0)
	if err := core.SetValueType(TypeGadget, core.ValueTypeDescr{
		Name:    func(core.Value) string { return "gadget" },
		String:  func(v core.Value) string { return fmt.Sprintf("gadget(%d)", int64(v.Data)) },
		Methods: gadgetMethods,
	}); err != nil {
		panic(err)
	}
}

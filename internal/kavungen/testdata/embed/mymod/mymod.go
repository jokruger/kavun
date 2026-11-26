// Package mymod is an embedder module: a function and a constant, named by the member names package gadget declares.
package mymod

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// ModuleID is the module's type id, in the embedder range.
const ModuleID = value.FirstUserDefinedType + 9

// The embedder's own module, beside Kavun's: a [gadget](kavun:type/gadget)'s companion.
//
//kavun:module mymod id=ModuleID pure

// Answer twice n.
//
//kavun:fn double slot=0 pure (n) -> int
func modDouble(_ core.VM, args []core.Value) (core.Value, error) {
	n, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("mymod.double", "first", "int(compatible)", args[0].TypeName())
	}
	return core.IntValue(2 * n), nil
}

// The module's label, a [`string`](kavun:type/string).
//
//kavun:const label -> string
var constLabel = core.NewStringValue("embedded")

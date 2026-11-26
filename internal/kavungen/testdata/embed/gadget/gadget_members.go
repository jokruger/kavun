package gadget

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
)

// Answer twice the gadget's number.
//
// ```go norun the fixture's examples are not run
// g.double()
// ```
//
//kavun:member double pure () -> int
func gadgetDouble(_ core.VM, v core.Value, _ member.ID, _ []core.Value) (core.Value, error) {
	return core.IntValue(2 * int64(v.Data)), nil
}

// Answer the gadget's number.
//
//kavun:member len pure () -> int
func gadgetLen(_ core.VM, v core.Value, _ member.ID, _ []core.Value) (core.Value, error) {
	return core.IntValue(int64(v.Data)), nil
}

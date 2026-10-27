package ext

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
)

// Answer the number of parts.
//
//kavun:member len pure () -> int
func gadgetLen(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	return v, nil
}

// Render the gadget.
//
//kavun:member format pure ([spec]) -> string
func gadgetFormat(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	return core.FormatWithSpec(v, id, args)
}

package os

import "github.com/jokruger/kavun/core"

// The operating system: a world module.
//
//kavun:module os impure

// Answer an environment variable.
//
//kavun:fn get_env slot=3 impure (name) -> string
func osGetenv(_ core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

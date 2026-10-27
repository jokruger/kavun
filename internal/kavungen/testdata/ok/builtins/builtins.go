package vm

import "github.com/jokruger/kavun/core"

// The free builtin functions: the builtins root has no summary, its whole block is the page's intro.
//
//kavun:builtins

// Convert to an int.
//
// - a string is parsed
// - a float must be whole
//
//kavun:builtin int slot=6 pure (x[, default]) -> int
func builtinInt(vm core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

// Answer the length.
//
// #### Containers
//
// | type | length |
// | --- | --- |
// | array | elements |
//
//kavun:builtin len slot=0 pure (x) -> int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

// Recover a raised error.
//
//kavun:builtin recover slot=41 impure () -> value
func builtinRecover(vm core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

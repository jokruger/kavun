package path

import "github.com/jokruger/kavun/core"

// Path text: joining and splitting file paths.
//
// A module page lists its functions alphabetically, whatever their source order.
//
//kavun:module path pure

// Join the parts with the separator.
//
// ```go
// path.join("a", "b")   // "a/b"
// ```
//
//kavun:fn join slot=0 pure (...parts) -> string
func pathJoin(_ core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

// Answer the last element.
//
//kavun:fn base slot=1 pure (p) -> string
func pathBase(_ core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }

// moduleExtras adds what the directives do not declare.
func moduleExtras(m *core.ModuleBuilder) {}

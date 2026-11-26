package path

import "github.com/jokruger/kavun/core"

// Path text: joining and splitting file paths.
//
// A module page lists its functions alphabetically, whatever their source order, then its constants, alphabetically
// too: [`path_separator`](#path_separator) joins the parts.
//
//kavun:module path id=value.ModulePath pure

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

// The separator of path elements, written by [`join`](#join).
//
// The list separator is [`path_list_separator`](kavun:module/path/path_list_separator).
//
// ```go
// path.path_separator   // => '/'
// ```
//
//kavun:const path_separator -> rune
var constPathSeparator = core.RuneValue('/')

var (
	// The separator of the paths in a list.
	//
	//kavun:const path_list_separator -> rune
	constPathListSeparator = core.RuneValue(':')
)

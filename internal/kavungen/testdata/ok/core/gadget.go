package core

// A gadget: the fixture for a type with no members file — its page is this block alone.
//
// ## Fields
//
// A gadget is read through its fields; the free builtins document the rest:
//
// | builtin | reading |
// | --- | --- |
// | `len(g)` | the field count |
//
// An example that is not run keeps its `norun` marker in the source only; the page shows a plain `go` fence:
//
// ```go norun — g is a placeholder
// len(g)
// ```
//
//kavun:type gadget

// TypeGadget is the gadget's descriptor; with no members file, it has no generated table.
var TypeGadget = ValueTypeDescr{}

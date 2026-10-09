// Package member is the process-wide registry of member names: every name a script can write after a dot
// (`x.name(...)`, `x.name`) may be bound to a small integer ID, which the compiler emits beside the name and a
// type's Methods/Properties tables are indexed by. The name stays authoritative: an unbound name (ID Unknown) is
// still answered through the type's named hooks, so binding a name is a speed-up, never a requirement.
//
// Once defined, member names and their IDs are permanent compatibility commitments: every future release must
// preserve all existing name/ID bindings. Releases may add new bindings, but must never rename, remove, renumber,
// or reuse existing ones, because user code may depend on them.
//
// Binding happens at init time only — builtin names by core, an embedder's names by its own init(), before any
// source or bytecode that uses them is compiled or loaded. The registry is not guarded for concurrent writes;
// reads (Name, Lookup) after init are safe from any goroutine.
package member

import "fmt"

// ID identifies a member name. Unknown (0) means "not bound": the name is resolved at the call site instead.
type ID uint16

const (
	Unknown          ID = 0
	FirstUserDefined ID = 2048 // embedder range FirstUserDefined…Max-1; 1…FirstUserDefined-1 is builtin
	Max              ID = 4096 // refused at registration; also the ceiling of a type's table length
)

var (
	names [Max]string
	ids   = make(map[string]ID)
)

// Define binds id to name. It panics on id == Unknown, id >= Max, an empty name, a name already bound or an id
// already bound — every one of them a host setup mistake found at init, before any script runs.
func Define(id ID, name string) {
	switch {
	case id == Unknown:
		panic(fmt.Sprintf("member.Define(%d, %q): id 0 is Unknown and cannot be bound", id, name))
	case id >= Max:
		panic(fmt.Sprintf("member.Define(%d, %q): id out of range (max: %d)", id, name, Max-1))
	case name == "":
		panic(fmt.Sprintf("member.Define(%d, %q): empty name", id, name))
	case names[id] != "":
		panic(fmt.Sprintf("member.Define(%d, %q): id already bound to %q", id, name, names[id]))
	}
	if prev, ok := ids[name]; ok {
		panic(fmt.Sprintf("member.Define(%d, %q): name already bound to id %d", id, name, prev))
	}
	names[id] = name
	ids[name] = id
}

// Name answers the name bound to id, or "" for Unknown and an unbound id.
func Name(id ID) string {
	if id >= Max {
		return ""
	}
	return names[id]
}

// String answers Name(id): Go code that names a member by its constant writes `member.X.String()` beside it, so
// the constant and the spelling can never disagree.
func (id ID) String() string {
	return Name(id)
}

// Lookup answers the ID bound to name, or Unknown when the name is unbound.
func Lookup(name string) ID {
	return ids[name]
}

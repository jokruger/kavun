package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// The host configuration a VM carries across runs (Reset keeps it): which builtin modules an import may load, and
// the standard streams.

const (
	grantPure  = uint8(iota) // default: only the pure modules (core.ModuleDef.Pure)
	grantNamed               // SetAllowedModules: exactly the named modules
	grantAll                 // AllowAllModules: every module
)

// SetAllowedModules grants exactly the named builtin modules to the scripts this VM runs; an import of any other
// raises module_not_allowed. Without a call only the pure modules are granted. This is the run-time half of the
// grant (the compiler checks the same rule), so bytecode compiled elsewhere or decoded cannot bypass it.
func (v *VM) SetAllowedModules(names ...string) error {
	var grant [4]uint64
	for _, name := range names {
		def, ok := core.LookupModule(name)
		if !ok {
			// a host setup mistake, not a script fault: fatal, kind "host"
			return errs.NewHostError(fmt.Sprintf("(SetAllowedModules) unknown module '%s'", name))
		}
		grant[def.Type/64] |= 1 << (def.Type % 64)
	}
	v.grant = grant
	v.grantMode = grantNamed
	return nil
}

// AllowAllModules grants every builtin module, world-touching ones included.
func (v *VM) AllowAllModules() {
	v.grantMode = grantAll
}

func (v *VM) moduleAllowed(def *core.ModuleDef) bool {
	switch v.grantMode {
	case grantAll:
		return true
	case grantNamed:
		return v.grant[def.Type/64]&(1<<(def.Type%64)) != 0
	default:
		return def.Pure
	}
}

// defaultStdin is os.Stdin behind one buffer shared by every VM that reads it, so no VM's buffer swallows input
// another one should see.
var defaultStdin = bufio.NewReader(os.Stdin)

// SetStdin sets the reader the io module reads from (default os.Stdin). The VM reads it through one buffer, so a
// line read followed by a read of the rest loses nothing.
func (v *VM) SetStdin(r io.Reader) {
	if br, ok := r.(*bufio.Reader); ok {
		v.stdin = br
		return
	}
	v.stdin = bufio.NewReader(r)
}

// SetStdout sets the writer the io module prints to (default os.Stdout).
func (v *VM) SetStdout(w io.Writer) {
	v.stdout = w
}

// Stdin answers the reader the io module reads from: a *bufio.Reader, so every reader of it shares one buffer.
func (v *VM) Stdin() io.Reader {
	return v.stdin
}

// Stdout answers the writer the io module prints to.
func (v *VM) Stdout() io.Writer {
	return v.stdout
}

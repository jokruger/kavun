package exec

import (
	"fmt"
	"os/exec"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
)

// exec.command is a prepared external command. It is owned by the exec module — answered by exec.exec(name,
// args...) — and registered by it (core.ModuleBuilder.Type). It is configured (set_dir, set_env, …) and
// then run once; two values are equal only when they are the same command.

const commandTypeName = "exec.command"

// NewCommandValue wraps a prepared command.
func NewCommandValue(c *exec.Cmd) core.Value {
	return core.Value{Type: value.ExecCommand, Immutable: true, Ptr: unsafe.Pointer(c)}
}

// CommandOf unwraps an exec.command value.
func CommandOf(v core.Value) *exec.Cmd {
	return (*exec.Cmd)(v.Ptr)
}

// A prepared external command, as `exec.exec(name, args...)` answers it.
//
// ## Overview
//
// `exec.command` is owned by the [`exec`](../stdlib/exec.md) module. Configure it (`set_dir`, `set_env`, `set_path`),
// then run it once — `run()`, `output()`, `combined_output()`, or `start()` followed by `wait()`. A refusal or a failed
// run raises `io` naming the member. It renders as `<exec.command:NAME>`; no `string()`.
//
// ```go
// exec := import("exec")
// string(exec.exec("echo", "hi").output())   // => "hi\n" — output() answers bytes
// c := exec.exec("sh", "-c", "exit 3")
// c.start()
// c.process().wait().success            // => false
// ```
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `==` / `!=` | the same command (identity) |
// | `!` | a command is always truthy |
// | everything else | raises |
//
//kavun:type exec.command

// TypeCommand is the exec.command type descriptor.
var TypeCommand = core.ValueTypeDescr{
	Name:   core.ConstHook(commandTypeName),         // PURE by contract
	String: commandTypeString,                       // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  core.SamePtrEqual,                       // PURE by contract

	Methods: commandMethods,
}

// A command renders as <exec.command:NAME>, the name it was given.
func commandTypeString(v core.Value) string {
	c := CommandOf(v)
	name := c.Path
	if len(c.Args) > 0 {
		name = c.Args[0]
	}
	return fmt.Sprintf("<%s:%s>", commandTypeName, name)
}

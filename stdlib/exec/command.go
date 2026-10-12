package exec

import (
	"fmt"
	"os/exec"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
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

// TypeCommand is the exec.command type descriptor.
var TypeCommand = core.ValueTypeDescr{
	Name:   core.ConstHook(commandTypeName),         // PURE by contract
	String: commandTypeString,                       // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  core.SamePtrEqual,                       // PURE by contract

	Methods: []core.Method{
		members.IsTrue:         core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.Format:         core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:           core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:         core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Run:            core.NewMethod(commandRun, 0, core.Fixed),
		members.Start:          core.NewMethod(commandStart, 0, core.Fixed),
		members.Wait:           core.NewMethod(commandWait, 0, core.Fixed),
		members.Output:         core.NewMethod(commandOutput, 0, core.Fixed),
		members.CombinedOutput: core.NewMethod(commandCombinedOutput, 0, core.Fixed),
		members.SetPath:        core.NewMethod(commandSetPath, 1, core.Fixed),
		members.SetDir:         core.NewMethod(commandSetDir, 1, core.Fixed),
		members.SetEnv:         core.NewMethod(commandSetEnv, 1, core.Fixed),
		members.Process:        core.NewMethod(commandProcess, 0, core.Fixed),
	},
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

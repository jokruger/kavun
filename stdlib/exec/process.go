package exec

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
)

// exec.process is a running (or finished) process. It is owned by the exec module — answered by
// exec.start_process, exec.find_process and command.process() — and registered by it (core.ModuleBuilder.Type). Two values are equal when they name the same process (pid).

const processTypeName = "exec.process"

// NewProcessValue wraps a process.
func NewProcessValue(p *os.Process) core.Value {
	return core.Value{Type: value.ExecProcess, Immutable: true, Ptr: unsafe.Pointer(p)}
}

// ProcessOf unwraps an exec.process value.
func ProcessOf(v core.Value) *os.Process {
	return (*os.Process)(v.Ptr)
}

// TypeProcess is the exec.process type descriptor.
var TypeProcess = core.ValueTypeDescr{
	Name:   core.ConstHook(processTypeName),         // PURE by contract
	String: processTypeString,                       // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  processTypeEqual,                        // PURE by contract

	Methods: []core.Method{
		members.IsTrue:  core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.Format:  core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:    core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:  core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.PID:     core.NewMethod(processPid, 0, core.Fixed),
		members.Kill:    core.NewMethod(processKill, 0, core.Fixed),
		members.Signal:  core.NewMethod(processSignal, 1, core.Fixed),
		members.Wait:    core.NewMethod(processWait, 0, core.Fixed),
		members.Release: core.NewMethod(processRelease, 0, core.Fixed),
	},
}

// A process renders as <exec.process:PID>.
func processTypeString(v core.Value) string {
	return fmt.Sprintf("<%s:%d>", processTypeName, ProcessOf(v).Pid)
}

func processTypeEqual(v core.Value, other core.Value, final bool) bool {
	if other.Type == value.ExecProcess {
		return ProcessOf(v).Pid == ProcessOf(other).Pid
	}
	if final {
		return false
	}
	return core.ValueTypes[other.Type].Equal(other, v, true)
}

// NewWaitValue is the frozen record process.wait() answers: facts about how the process ended, not behaviour.
func NewWaitValue(s *os.ProcessState) core.Value {
	return core.NewRecordValue(map[string]core.Value{
		"pid":     core.IntValue(int64(s.Pid())),
		"exited":  core.BoolValue(s.Exited()),
		"success": core.BoolValue(s.Success()),
		"status":  core.NewStringValue(s.String()), // "exit status 1", "signal: killed", …
	}, true)
}

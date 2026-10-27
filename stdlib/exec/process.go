package exec

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/jokruger/kavun/core"
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

// A process, as `exec.start_process`, `exec.find_process` and `command.process()` answer it.
//
// ## Overview
//
// `exec.process` is owned by the [`exec`](../stdlib/exec.md) module. Every member reaches the operating system; a
// refusal raises `io` naming the member. It renders as `<exec.process:PID>`; no `string()`.
//
// ```go
// exec := import("exec")
// p := exec.start_process("/bin/sleep", ["sleep", "0"], "", [])
// p.pid() > 0                // => true
// st := p.wait()             // {pid: …, exited: true, success: true, status: "exit status 0"}
// st.success                 // => true
// ```
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `==` / `!=` | the same process (by pid) |
// | `!` | a process is always truthy |
// | everything else | raises |
//
//kavun:type exec.process

// TypeProcess is the exec.process type descriptor.
var TypeProcess = core.ValueTypeDescr{
	Name:   core.ConstHook(processTypeName),         // PURE by contract
	String: processTypeString,                       // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  processTypeEqual,                        // PURE by contract

	Methods: processMethods,
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

package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The exec module runs external programs: exec.command (prepare, configure, run) and exec.process (a running
// process). Both types live in core.

// Run external programs.
//
// ```go
// exec := import("exec")
// string(exec.exec("echo", "hi").output())   // => "hi\n" — output() answers bytes
// p := exec.start_process("/bin/sleep", ["sleep", "0"], "", [])
// p.wait().success                        // => true
// ```
//
// The two types are documented on their own pages: [`exec.command`](../types/exec_command.md) (`run`, `start`, `wait`,
// `output`, `combined_output`, `set_path`, `set_dir`, `set_env`, `process`) and
// [`exec.process`](../types/exec_process.md) (`pid`, `kill`, `signal`, `wait`, `release`). `process.wait()` answers a
// frozen record of facts: `{pid, exited, success, status}`.
//
// **Failures.** A program that cannot be found or run, or ends badly for `run`/`output`, raises `io` naming the call
// (`(exec.command.run) exit status 1`).
//
//kavun:module exec impure

// moduleExtras declares what the //kavun:fn directives do not: the module's constants and owned types.
func moduleExtras(m *core.ModuleBuilder) {
	m.Type(value.ExecCommand, TypeCommand)
	m.Type(value.ExecProcess, TypeProcess)
}

// Prepare a command; configure it, then run it once.
//
// `name` is a `string`.
//
//kavun:fn exec slot=0 impure (name, ...args) -> exec.command
func execExec(vm core.VM, args []core.Value) (core.Value, error) {
	name, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("exec.exec", "first", "string(compatible)", args[0].TypeName())
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := arg.AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("exec.exec", fmt.Sprintf("args[%d]", idx), "string(compatible)", arg.TypeName())
		}
		execArgs = append(execArgs, execArg)
	}
	return NewCommandValue(exec.Command(name, execArgs...)), nil
}

// Search `PATH` for an executable named `file`.
//
// `file` is a `string`.
//
//kavun:fn exec_look_path slot=1 impure (file) -> string
func execLookPath(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("exec.exec_look_path", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := exec.LookPath(s1)
	if err != nil {
		return modkit.RaiseIO("exec.exec_look_path", err)
	}
	return core.NewStringValue(res), nil
}

// Start a process.
//
// `name` is a `string`, `argv` is an array of strings, `dir` is a `string` and `env` is an array of strings.
//
//kavun:fn start_process slot=2 impure (name, argv, dir, env) -> exec.process
func execStartProcess(vm core.VM, args []core.Value) (core.Value, error) {
	name, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("exec.start_process", "first", "string(compatible)", args[0].TypeName())
	}
	argv, err := StringsArg("exec.start_process", "second", args[1])
	if err != nil {
		return core.Undefined, err
	}

	dir, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("exec.start_process", "third", "string(compatible)", args[2].TypeName())
	}

	env, err := StringsArg("exec.start_process", "fourth", args[3])
	if err != nil {
		return core.Undefined, err
	}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Dir: dir,
		Env: env,
	})
	if err != nil {
		return modkit.RaiseIO("exec.start_process", err)
	}
	return NewProcessValue(proc), nil
}

// Answer the process with that pid.
//
// `pid` is an `int`.
//
//kavun:fn find_process slot=3 impure (pid) -> exec.process
func execFindProcess(vm core.VM, args []core.Value) (core.Value, error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("exec.find_process", "first", "int(compatible)", args[0].TypeName())
	}
	proc, err := os.FindProcess(int(i1))
	if err != nil {
		return modkit.RaiseIO("exec.find_process", err)
	}
	return NewProcessValue(proc), nil
}

package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The exec module runs external programs: exec.command (prepare, configure, run) and exec.process (a running
// process). Both types live in core.

func init() {
	m := core.NewModule(value.ModuleExec, "exec", false)
	m.Type(value.ExecCommand, TypeCommand)
	m.Type(value.ExecProcess, TypeProcess)
	m.Fn(members.Exec, 0, execExec, 1, core.Variadic)
	m.Fn(members.ExecLookPath, 1, execLookPath, 1, core.Fixed)
	m.Fn(members.StartProcess, 2, execStartProcess, 4, core.Fixed)
	m.Fn(members.FindProcess, 3, execFindProcess, 1, core.Fixed)
	// 4..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func execExec(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("exec.exec", "at least 1", len(args))
	}
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

func execLookPath(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("exec.exec_look_path", "1", len(args))
	}
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

func execStartProcess(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("exec.start_process", "4", len(args))
	}
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

func execFindProcess(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("exec.find_process", "1", len(args))
	}
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

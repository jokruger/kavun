package stdlib

import (
	"os/exec"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func makeOSExecCommand(a *core.Arena, vm core.VM, cmd *exec.Cmd) (core.Value, error) {
	// combined_output() => bytes/error
	cmdCombinedOutput, err := a.NewBuiltinClosureValue("combined_output", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.combined_output", "0", len(args))
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(a, err)
		}
		return a.NewBytesValue(res, false)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// output() => bytes/error
	cmdOutput, err := a.NewBuiltinClosureValue("output", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.output", "0", len(args))
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(a, err)
		}
		return a.NewBytesValue(res, false)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// run() => error
	cmdRun, err := a.NewBuiltinClosureValue("run", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.run", "0", len(args))
		}
		return wrapError(a, cmd.Run())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// start() => error
	cmdStart, err := a.NewBuiltinClosureValue("start", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.start", "0", len(args))
		}
		return wrapError(a, cmd.Start())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// wait() => error
	cmdWait, err := a.NewBuiltinClosureValue("wait", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.wait", "0", len(args))
		}
		return wrapError(a, cmd.Wait())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// set_path(path string)
	cmdSetPath, err := a.NewBuiltinClosureValue("set_path", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_path", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_path", "first", "string(compatible)", args[0].TypeName())
		}
		cmd.Path = s1
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// set_dir(dir string)
	cmdSetDir, err := a.NewBuiltinClosureValue("set_dir", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_dir", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_dir", "first", "string(compatible)", args[0].TypeName())
		}
		cmd.Dir = s1
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// set_env(env array(string))
	cmdSetEnv, err := a.NewBuiltinClosureValue("set_env", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_env", "1", len(args))
		}

		var env []string
		var err error

		if args[0].Type != value.Array {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_env", "first", "array(string)", args[0].TypeName())
		}
		arr := a.ResolveArrayValue(args[0])
		env, err = stringArray(a, arr.Elements, "first")
		if err != nil {
			return core.Undefined, err
		}

		cmd.Env = env
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// process() => idict(process)
	cmdProcess, err := a.NewBuiltinClosureValue("process", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.process", "0", len(args))
		}
		return makeOSProcess(a, vm, cmd.Process)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
		"combined_output": cmdCombinedOutput,
		"output":          cmdOutput,
		"run":             cmdRun,
		"start":           cmdStart,
		"wait":            cmdWait,
		"set_path":        cmdSetPath,
		"set_dir":         cmdSetDir,
		"set_env":         cmdSetEnv,
		"process":         cmdProcess,
	}, true)
	if err != nil {
		return core.Undefined, err
	}

	return m, nil
}

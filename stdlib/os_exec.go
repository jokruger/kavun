package stdlib

import (
	"os/exec"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeOSExecCommand(vm core.VM, cmd *exec.Cmd) (core.Value, error) {
	alloc := vm.Allocator()

	// combined_output() => bytes/error
	cmdCombinedOutput, err := alloc.NewBuiltinFunctionValue("combined_output", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.combined_output", "0", len(args))
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(vm, err)
		}
		if len(res) > core.MaxBytesLen {
			return core.Undefined, errs.NewBytesLimitError("os.exec.combined_output")
		}
		return vm.Allocator().NewBytesValue(res)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// output() => bytes/error
	cmdOutput, err := alloc.NewBuiltinFunctionValue("output", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.output", "0", len(args))
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(vm, err)
		}
		if len(res) > core.MaxBytesLen {
			return core.Undefined, errs.NewBytesLimitError("os.exec.output")
		}
		return vm.Allocator().NewBytesValue(res)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// run() => error
	cmdRun, err := alloc.NewBuiltinFunctionValue("run", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.run", "0", len(args))
		}
		return wrapError(vm, cmd.Run())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// start() => error
	cmdStart, err := alloc.NewBuiltinFunctionValue("start", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.start", "0", len(args))
		}
		return wrapError(vm, cmd.Start())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// wait() => error
	cmdWait, err := alloc.NewBuiltinFunctionValue("wait", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.wait", "0", len(args))
		}
		return wrapError(vm, cmd.Wait())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// set_path(path string)
	cmdSetPath, err := alloc.NewBuiltinFunctionValue("set_path", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	cmdSetDir, err := alloc.NewBuiltinFunctionValue("set_dir", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	cmdSetEnv, err := alloc.NewBuiltinFunctionValue("set_env", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_env", "1", len(args))
		}

		var env []string
		var err error

		if args[0].Type != core.VT_ARRAY {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_env", "first", "array(string)", args[0].TypeName())
		}
		arr := (*core.Array)(args[0].Ptr)
		env, err = stringArray(arr.Elements, "first")
		if err != nil {
			return core.Undefined, err
		}

		cmd.Env = env
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// process() => imap(process)
	cmdProcess, err := alloc.NewBuiltinFunctionValue("process", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.process", "0", len(args))
		}
		return makeOSProcess(vm, cmd.Process)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := vm.Allocator().NewRecordValue(map[string]core.Value{
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

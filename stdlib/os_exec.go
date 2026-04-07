package stdlib

import (
	"os/exec"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeOSExecCommand(vm core.VM, cmd *exec.Cmd) *value.Record {
	cmdRun := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.run", "0", len(args))
		}
		return wrapError(vm, cmd.Run()), nil
	}

	cmdStart := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.start", "0", len(args))
		}
		return wrapError(vm, cmd.Start()), nil
	}

	cmdWait := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.wait", "0", len(args))
		}
		return wrapError(vm, cmd.Wait()), nil
	}

	cmdCombinedOutput := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.combined_output", "0", len(args))
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(vm, err), nil
		}
		if len(res) > core.MaxBytesLen {
			return core.UndefinedValue(), core.NewBytesLimitError("os.exec.combined_output")
		}
		return vm.Allocator().NewBytesValue(res), nil
	}

	cmdOutput := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.output", "0", len(args))
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(vm, err), nil
		}
		if len(res) > core.MaxBytesLen {
			return core.UndefinedValue(), core.NewBytesLimitError("os.exec.output")
		}
		return vm.Allocator().NewBytesValue(res), nil
	}

	cmdSetPath := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.set_path", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("os.exec.set_path", "first", "string(compatible)", args[0].TypeName())
		}
		cmd.Path = s1
		return core.UndefinedValue(), nil
	}

	cmdSetDir := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.set_dir", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("os.exec.set_dir", "first", "string(compatible)", args[0].TypeName())
		}
		cmd.Dir = s1
		return core.UndefinedValue(), nil
	}

	cmdSetEnv := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.set_env", "1", len(args))
		}

		var env []string
		var err error

		if !args[0].IsArray() {
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("os.exec.set_env", "first", "array(string)", args[0].TypeName())
		}

		env, err = stringArray(args[0].Object().(*value.Array).Value(), "first")
		if err != nil {
			return core.UndefinedValue(), err
		}

		cmd.Env = env
		return core.UndefinedValue(), nil
	}

	cmdProcess := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("os.exec.process", "0", len(args))
		}
		t := makeOSProcess(vm, cmd.Process)
		return core.ObjectValue(t), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecord(map[string]core.Value{
		"combined_output": alloc.NewBuiltinFunctionValue("combined_output", cmdCombinedOutput, 0, false), // combined_output() => bytes/error
		"output":          alloc.NewBuiltinFunctionValue("output", cmdOutput, 0, false),                  // output() => bytes/error
		"run":             alloc.NewBuiltinFunctionValue("run", cmdRun, 0, false),                        // run() => error
		"start":           alloc.NewBuiltinFunctionValue("start", cmdStart, 0, false),                    // start() => error
		"wait":            alloc.NewBuiltinFunctionValue("wait", cmdWait, 0, false),                      // wait() => error
		"set_path":        alloc.NewBuiltinFunctionValue("set_path", cmdSetPath, 1, false),               // set_path(path string)
		"set_dir":         alloc.NewBuiltinFunctionValue("set_dir", cmdSetDir, 1, false),                 // set_dir(dir string)
		"set_env":         alloc.NewBuiltinFunctionValue("set_env", cmdSetEnv, 1, false),                 // set_env(env array(string))
		"process":         alloc.NewBuiltinFunctionValue("process", cmdProcess, 0, false),                // process() => imap(process)
	}, true).(*value.Record)
}

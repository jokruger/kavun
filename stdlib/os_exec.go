package stdlib

import (
	"os/exec"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func makeOSExecCommand(vm core.VM, cmd *exec.Cmd) (core.Value, error) {
	// combined_output() => bytes/error
	cmdCombinedOutput := core.NewBuiltinClosureValue("combined_output", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.combined_output", "0", len(args))
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(err)
		}
		return core.NewBytesValue(res, false), nil
	}, 0, false)

	// output() => bytes/error
	cmdOutput := core.NewBuiltinClosureValue("output", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.output", "0", len(args))
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(err)
		}
		return core.NewBytesValue(res, false), nil
	}, 0, false)

	// run() => error
	cmdRun := core.NewBuiltinClosureValue("run", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.run", "0", len(args))
		}
		return wrapError(cmd.Run())
	}, 0, false)

	// start() => error
	cmdStart := core.NewBuiltinClosureValue("start", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.start", "0", len(args))
		}
		return wrapError(cmd.Start())
	}, 0, false)

	// wait() => error
	cmdWait := core.NewBuiltinClosureValue("wait", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.wait", "0", len(args))
		}
		return wrapError(cmd.Wait())
	}, 0, false)

	// set_path(path string)
	cmdSetPath := core.NewBuiltinClosureValue("set_path", func(vm core.VM, args []core.Value) (core.Value, error) {
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

	// set_dir(dir string)
	cmdSetDir := core.NewBuiltinClosureValue("set_dir", func(vm core.VM, args []core.Value) (core.Value, error) {
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

	// set_env(env array(string))
	cmdSetEnv := core.NewBuiltinClosureValue("set_env", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_env", "1", len(args))
		}

		var env []string
		var err error

		if args[0].Type != value.Array {
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

	// process() => idict(process)
	cmdProcess := core.NewBuiltinClosureValue("process", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.process", "0", len(args))
		}
		return makeOSProcess(vm, cmd.Process)
	}, 0, false)

	m := core.NewRecordValue(map[string]core.Value{
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

	return m, nil
}

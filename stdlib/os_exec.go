package stdlib

import (
	"os/exec"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSExecCommand(a *core.Arena, vm core.VM, cmd *exec.Cmd) (core.Value, error) {
	// combined_output() => bytes/error
	cmdCombinedOutput := a.NewBuiltinClosureValue("combined_output", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.combined_output", "0", len(args))
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(a, err)
		}
		return a.NewBytesValue(res, false), nil
	}, 0, false)

	// output() => bytes/error
	cmdOutput := a.NewBuiltinClosureValue("output", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.output", "0", len(args))
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(a, err)
		}
		return a.NewBytesValue(res, false), nil
	}, 0, false)

	// run() => error
	cmdRun := a.NewBuiltinClosureValue("run", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.run", "0", len(args))
		}
		return wrapError(a, cmd.Run())
	}, 0, false)

	// start() => error
	cmdStart := a.NewBuiltinClosureValue("start", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.start", "0", len(args))
		}
		return wrapError(a, cmd.Start())
	}, 0, false)

	// wait() => error
	cmdWait := a.NewBuiltinClosureValue("wait", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.wait", "0", len(args))
		}
		return wrapError(a, cmd.Wait())
	}, 0, false)

	// set_path(path string)
	cmdSetPath := a.NewBuiltinClosureValue("set_path", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_path", "1", len(args))
		}
		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_path", "first", "string(compatible)", args[0].TypeName(a))
		}
		cmd.Path = s1
		return core.Undefined, nil
	}, 1, false)

	// set_dir(dir string)
	cmdSetDir := a.NewBuiltinClosureValue("set_dir", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_dir", "1", len(args))
		}
		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_dir", "first", "string(compatible)", args[0].TypeName(a))
		}
		cmd.Dir = s1
		return core.Undefined, nil
	}, 1, false)

	// set_env(env array(string))
	cmdSetEnv := a.NewBuiltinClosureValue("set_env", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.set_env", "1", len(args))
		}

		var env []string
		var err error

		if args[0].Type != core.VT_ARRAY {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec.set_env", "first", "array(string)", args[0].TypeName(a))
		}
		arr := (*core.Array)(args[0].Ptr)
		env, err = stringArray(a, arr.Elements, "first")
		if err != nil {
			return core.Undefined, err
		}

		cmd.Env = env
		return core.Undefined, nil
	}, 1, false)

	// process() => idict(process)
	cmdProcess := a.NewBuiltinClosureValue("process", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.exec.process", "0", len(args))
		}
		return makeOSProcess(a, vm, cmd.Process)
	}, 0, false)

	m := a.NewRecordValue(map[string]core.Value{
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

package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSProcessState(a *core.Arena, vm core.VM, state *os.ProcessState) (core.Value, error) {
	stateExited := a.NewBuiltinFunctionValue("exited", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.exited", "0", len(args))
		}
		return core.BoolValue(state.Exited()), nil
	}, 0, false)

	statePid := a.NewBuiltinFunctionValue("pid", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.pid", "0", len(args))
		}
		return core.IntValue(int64(state.Pid())), nil
	}, 0, false)

	stateString := a.NewBuiltinFunctionValue("string", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.string", "0", len(args))
		}
		s := state.String()
		return a.NewStringValue(s), nil
	}, 0, false)

	stateSuccess := a.NewBuiltinFunctionValue("success", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.success", "0", len(args))
		}
		return core.BoolValue(state.Success()), nil
	}, 0, false)

	m := a.NewRecordValue(map[string]core.Value{
		"exited":  stateExited,
		"pid":     statePid,
		"string":  stateString,
		"success": stateSuccess,
	}, true)

	return m, nil
}

func makeOSProcess(a *core.Arena, vm core.VM, proc *os.Process) (core.Value, error) {
	procKill := a.NewBuiltinFunctionValue("kill", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.kill", "0", len(args))
		}
		return wrapError(proc.Kill())
	}, 0, false)

	procRelease := a.NewBuiltinFunctionValue("release", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.release", "0", len(args))
		}
		return wrapError(proc.Release())
	}, 0, false)

	procSignal := a.NewBuiltinFunctionValue("signal", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.signal", "1", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.process.signal", "first", "int(compatible)", args[0].TypeName(a))
		}
		return wrapError(proc.Signal(syscall.Signal(i1)))
	}, 1, false)

	procWait := a.NewBuiltinFunctionValue("wait", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.wait", "0", len(args))
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(err)
		}
		return makeOSProcessState(a, vm, state)
	}, 0, false)

	m := a.NewRecordValue(map[string]core.Value{
		"kill":    procKill,
		"release": procRelease,
		"signal":  procSignal,
		"wait":    procWait,
	}, true)

	return m, nil
}

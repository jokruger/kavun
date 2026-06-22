package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSProcessState(vm core.VM, state *os.ProcessState) (core.Value, error) {
	stateExited, err := a.NewBuiltinClosureValue("exited", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.exited", "0", len(args))
		}
		return core.BoolValue(state.Exited()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	statePid, err := a.NewBuiltinClosureValue("pid", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.pid", "0", len(args))
		}
		return core.IntValue(int64(state.Pid())), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	stateString, err := a.NewBuiltinClosureValue("string", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.string", "0", len(args))
		}
		s := state.String()
		return a.NewStringValue(s)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	stateSuccess, err := a.NewBuiltinClosureValue("success", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.success", "0", len(args))
		}
		return core.BoolValue(state.Success()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
		"exited":  stateExited,
		"pid":     statePid,
		"string":  stateString,
		"success": stateSuccess,
	}, true)
	if err != nil {
		return core.Undefined, err
	}

	return m, nil
}

func makeOSProcess(vm core.VM, proc *os.Process) (core.Value, error) {
	procKill, err := a.NewBuiltinClosureValue("kill", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.kill", "0", len(args))
		}
		return wrapError(a, proc.Kill())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	procRelease, err := a.NewBuiltinClosureValue("release", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.release", "0", len(args))
		}
		return wrapError(a, proc.Release())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	procSignal, err := a.NewBuiltinClosureValue("signal", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.signal", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.process.signal", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(a, proc.Signal(syscall.Signal(i1)))
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	procWait, err := a.NewBuiltinClosureValue("wait", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.wait", "0", len(args))
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(err)
		}
		return makeOSProcessState(a, vm, state)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
		"kill":    procKill,
		"release": procRelease,
		"signal":  procSignal,
		"wait":    procWait,
	}, true)
	if err != nil {
		return core.Undefined, err
	}

	return m, nil
}

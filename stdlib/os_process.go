package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeOSProcessState(vm core.VM, state *os.ProcessState) (core.Value, error) {
	alloc := vm.Allocator()

	stateExited, err := alloc.NewBuiltinFunctionValue("exited", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.exited", "0", len(args))
		}
		return core.BoolValue(state.Exited()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	statePid, err := alloc.NewBuiltinFunctionValue("pid", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.pid", "0", len(args))
		}
		return core.IntValue(int64(state.Pid())), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	stateString, err := alloc.NewBuiltinFunctionValue("string", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.string", "0", len(args))
		}
		s := state.String()
		if len(s) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("os.state.string")
		}
		return vm.Allocator().NewStringValue(s)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	stateSuccess, err := alloc.NewBuiltinFunctionValue("success", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.success", "0", len(args))
		}
		return core.BoolValue(state.Success()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := vm.Allocator().NewRecordValue(map[string]core.Value{
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
	alloc := vm.Allocator()

	procKill, err := alloc.NewBuiltinFunctionValue("kill", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.kill", "0", len(args))
		}
		return wrapError(vm, proc.Kill())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	procRelease, err := alloc.NewBuiltinFunctionValue("release", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.release", "0", len(args))
		}
		return wrapError(vm, proc.Release())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	procSignal, err := alloc.NewBuiltinFunctionValue("signal", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.signal", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.process.signal", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(vm, proc.Signal(syscall.Signal(i1)))
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	procWait, err := alloc.NewBuiltinFunctionValue("wait", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.wait", "0", len(args))
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(vm, err)
		}
		return makeOSProcessState(vm, state)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := vm.Allocator().NewRecordValue(map[string]core.Value{
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

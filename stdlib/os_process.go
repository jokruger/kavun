package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeOSProcessState(vm core.VM, state *os.ProcessState) core.Value {
	statePid := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.pid", "0", len(args))
		}
		return core.IntValue(int64(state.Pid())), nil
	}

	stateExited := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.exited", "0", len(args))
		}
		return core.BoolValue(state.Exited()), nil
	}

	stateSuccess := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.success", "0", len(args))
		}
		return core.BoolValue(state.Success()), nil
	}

	stateString := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.state.string", "0", len(args))
		}
		s := state.String()
		if len(s) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("os.state.string")
		}
		return vm.Allocator().NewStringValue(s), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecordValue(map[string]core.Value{
		"exited":  alloc.NewBuiltinFunctionValue("exited", stateExited, 0, false),
		"pid":     alloc.NewBuiltinFunctionValue("pid", statePid, 0, false),
		"string":  alloc.NewBuiltinFunctionValue("string", stateString, 0, false),
		"success": alloc.NewBuiltinFunctionValue("success", stateSuccess, 0, false),
	}, true)
}

func makeOSProcess(vm core.VM, proc *os.Process) core.Value {
	procKill := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.kill", "0", len(args))
		}
		return wrapError(vm, proc.Kill()), nil
	}

	procRelease := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.release", "0", len(args))
		}
		return wrapError(vm, proc.Release()), nil
	}

	procSignal := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.signal", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.process.signal", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(vm, proc.Signal(syscall.Signal(i1))), nil
	}

	procWait := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.process.wait", "0", len(args))
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(vm, err), nil
		}
		return makeOSProcessState(vm, state), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecordValue(map[string]core.Value{
		"kill":    alloc.NewBuiltinFunctionValue("kill", procKill, 0, false),
		"release": alloc.NewBuiltinFunctionValue("release", procRelease, 0, false),
		"signal":  alloc.NewBuiltinFunctionValue("signal", procSignal, 1, false),
		"wait":    alloc.NewBuiltinFunctionValue("wait", procWait, 0, false),
	}, true)
}

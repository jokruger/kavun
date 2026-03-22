package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeOSProcessState(vm core.VM, state *os.ProcessState) *value.Record {
	statePid := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.state.pid", "0", len(args))
		}
		return vm.Allocator().NewInt(int64(state.Pid())), nil
	}

	stateExited := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.state.exited", "0", len(args))
		}
		return vm.Allocator().NewBool(state.Exited()), nil
	}

	stateSuccess := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.state.success", "0", len(args))
		}
		return vm.Allocator().NewBool(state.Success()), nil
	}

	stateString := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.state.string", "0", len(args))
		}
		s := state.String()
		if len(s) > core.MaxStringLen {
			return nil, core.NewStringLimitError("os.state.string")
		}
		return vm.Allocator().NewString(s), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecord(map[string]core.Object{
		"exited":  alloc.NewBuiltinFunction("exited", stateExited, 0, false),
		"pid":     alloc.NewBuiltinFunction("pid", statePid, 0, false),
		"string":  alloc.NewBuiltinFunction("string", stateString, 0, false),
		"success": alloc.NewBuiltinFunction("success", stateSuccess, 0, false),
	}, true).(*value.Record)
}

func makeOSProcess(vm core.VM, proc *os.Process) *value.Record {
	procKill := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.process.kill", "0", len(args))
		}
		return wrapError(vm, proc.Kill()), nil
	}

	procRelease := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.process.release", "0", len(args))
		}
		return wrapError(vm, proc.Release()), nil
	}

	procSignal := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.process.signal", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.process.signal", "first", "int(compatible)", args[0])
		}
		return wrapError(vm, proc.Signal(syscall.Signal(i1))), nil
	}

	procWait := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.process.wait", "0", len(args))
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(vm, err), nil
		}
		return makeOSProcessState(vm, state), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecord(map[string]core.Object{
		"kill":    alloc.NewBuiltinFunction("kill", procKill, 0, false),
		"release": alloc.NewBuiltinFunction("release", procRelease, 0, false),
		"signal":  alloc.NewBuiltinFunction("signal", procSignal, 1, false),
		"wait":    alloc.NewBuiltinFunction("wait", procWait, 0, false),
	}, true).(*value.Record)
}

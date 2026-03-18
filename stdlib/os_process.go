package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeOSProcessState(state *os.ProcessState) *value.Map {
	statePid := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return value.NewInt(int64(state.Pid())), nil
	}

	stateExited := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		if state.Exited() {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}

	stateSuccess := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		if state.Success() {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}

	stateString := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		s := state.String()
		if len(s) > core.MaxStringLen {
			return nil, core.StringLimit("os.state.string")
		}
		return value.NewString(s), nil
	}

	return value.NewMap(map[string]core.Object{
		"exited":  value.NewBuiltinFunction("exited", stateExited, 0, false),
		"pid":     value.NewBuiltinFunction("pid", statePid, 0, false),
		"string":  value.NewBuiltinFunction("string", stateString, 0, false),
		"success": value.NewBuiltinFunction("success", stateSuccess, 0, false),
	}, true)
}

func makeOSProcess(proc *os.Process) *value.Map {
	procKill := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(proc.Kill()), nil
	}

	procRelease := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(proc.Release()), nil
	}

	procSignal := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.InvalidArgumentType("os.process.signal", "first", "int(compatible)", args[0])
		}
		return wrapError(proc.Signal(syscall.Signal(i1))), nil
	}

	procWait := func(args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		state, err := proc.Wait()
		if err != nil {
			return wrapError(err), nil
		}
		return makeOSProcessState(state), nil
	}

	return value.NewMap(map[string]core.Object{
		"kill":    value.NewBuiltinFunction("kill", procKill, 0, false),
		"release": value.NewBuiltinFunction("release", procRelease, 0, false),
		"signal":  value.NewBuiltinFunction("signal", procSignal, 1, false),
		"wait":    value.NewBuiltinFunction("wait", procWait, 0, false),
	}, true)
}

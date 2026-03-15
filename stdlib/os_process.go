package stdlib

import (
	"os"
	"syscall"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeOSProcessState(state *os.ProcessState) *value.ImmutableMap {
	statePid := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return &value.Int{Value: int64(state.Pid())}, nil
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

	return &value.ImmutableMap{
		Value: map[string]core.Object{
			"exited": &value.BuiltinFunction{
				Name:  "exited",
				Value: stateExited,
			},
			"pid": &value.BuiltinFunction{
				Name:  "pid",
				Value: statePid,
			},
			"string": &value.BuiltinFunction{
				Name:  "string",
				Value: FuncARS(state.String),
			},
			"success": &value.BuiltinFunction{
				Name:  "success",
				Value: stateSuccess,
			},
		},
	}
}

func makeOSProcess(proc *os.Process) *value.ImmutableMap {
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

	return &value.ImmutableMap{
		Value: map[string]core.Object{
			"kill": &value.BuiltinFunction{
				Name:  "kill",
				Value: procKill,
			},
			"release": &value.BuiltinFunction{
				Name:  "release",
				Value: procRelease,
			},
			"signal": &value.BuiltinFunction{
				Name: "signal",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 1 {
						return nil, gse.ErrWrongNumArguments
					}
					i1, ok := args[0].AsInt()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return wrapError(proc.Signal(syscall.Signal(i1))), nil
				},
			},
			"wait": &value.BuiltinFunction{
				Name: "wait",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 0 {
						return nil, gse.ErrWrongNumArguments
					}
					state, err := proc.Wait()
					if err != nil {
						return wrapError(err), nil
					}
					return makeOSProcessState(state), nil
				},
			},
		},
	}
}

package stdlib

import (
	"os/exec"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeOSExecCommand(cmd *exec.Cmd) *value.ImmutableMap {
	cmdRun := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(cmd.Run()), nil
	}

	cmdStart := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(cmd.Start()), nil
	}

	cmdWait := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(cmd.Wait()), nil
	}

	return &value.ImmutableMap{
		Value: map[string]core.Object{
			// combined_output() => bytes/error
			"combined_output": &value.BuiltinFunction{
				Name:  "combined_output",
				Value: FuncARYE(cmd.CombinedOutput),
			},
			// output() => bytes/error
			"output": &value.BuiltinFunction{
				Name:  "output",
				Value: FuncARYE(cmd.Output),
			}, //
			// run() => error
			"run": &value.BuiltinFunction{
				Name:  "run",
				Value: cmdRun,
			}, //
			// start() => error
			"start": &value.BuiltinFunction{
				Name:  "start",
				Value: cmdStart,
			}, //
			// wait() => error
			"wait": &value.BuiltinFunction{
				Name:  "wait",
				Value: cmdWait,
			}, //
			// set_path(path string)
			"set_path": &value.BuiltinFunction{
				Name: "set_path",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 1 {
						return nil, gse.ErrWrongNumArguments
					}
					s1, ok := args[0].AsString()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Path = s1
					return value.UndefinedValue, nil
				},
			},
			// set_dir(dir string)
			"set_dir": &value.BuiltinFunction{
				Name: "set_dir",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 1 {
						return nil, gse.ErrWrongNumArguments
					}
					s1, ok := args[0].AsString()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Dir = s1
					return value.UndefinedValue, nil
				},
			},
			// set_env(env array(string))
			"set_env": &value.BuiltinFunction{
				Name: "set_env",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 1 {
						return nil, gse.ErrWrongNumArguments
					}

					var env []string
					var err error
					switch arg0 := args[0].(type) {
					case *value.Array:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					case *value.ImmutableArray:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					default:
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "array",
							Found:    arg0.TypeName(),
						}
					}
					cmd.Env = env
					return value.UndefinedValue, nil
				},
			},
			// process() => imap(process)
			"process": &value.BuiltinFunction{
				Name: "process",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 0 {
						return nil, gse.ErrWrongNumArguments
					}
					return makeOSProcess(cmd.Process), nil
				},
			},
		},
	}
}

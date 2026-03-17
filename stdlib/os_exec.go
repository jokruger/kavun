package stdlib

import (
	"os/exec"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeOSExecCommand(cmd *exec.Cmd) *value.Map {
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

	cmdCombinedOutput := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		res, err := cmd.CombinedOutput()
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return value.NewBytes(res), nil
	}

	cmdOutput := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		res, err := cmd.Output()
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return value.NewBytes(res), nil
	}

	cmdSetPath := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		}
		cmd.Path = s1
		return value.UndefinedValue, nil
	}

	cmdSetDir := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		}
		cmd.Dir = s1
		return value.UndefinedValue, nil
	}

	cmdSetEnv := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}

		var env []string
		var err error
		switch arg0 := args[0].(type) {
		case *value.Array:
			env, err = stringArray(arg0.Value(), "first")
			if err != nil {
				return nil, err
			}
		default:
			return nil, gse.ErrInvalidArgumentType{Name: "first", Expected: "array", Found: arg0.TypeName()}
		}
		cmd.Env = env
		return value.UndefinedValue, nil
	}

	cmdProcess := func(args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return makeOSProcess(cmd.Process), nil
	}

	return value.NewMap(map[string]core.Object{
		"combined_output": value.NewBuiltinFunction("combined_output", cmdCombinedOutput, 0, false), // combined_output() => bytes/error
		"output":          value.NewBuiltinFunction("output", cmdOutput, 0, false),                  // output() => bytes/error
		"run":             value.NewBuiltinFunction("run", cmdRun, 0, false),                        // run() => error
		"start":           value.NewBuiltinFunction("start", cmdStart, 0, false),                    // start() => error
		"wait":            value.NewBuiltinFunction("wait", cmdWait, 0, false),                      // wait() => error
		"set_path":        value.NewBuiltinFunction("set_path", cmdSetPath, 1, false),               // set_path(path string)
		"set_dir":         value.NewBuiltinFunction("set_dir", cmdSetDir, 1, false),                 // set_dir(dir string)
		"set_env":         value.NewBuiltinFunction("set_env", cmdSetEnv, 1, false),                 // set_env(env array(string))
		"process":         value.NewBuiltinFunction("process", cmdProcess, 0, false),                // process() => imap(process)
	}, true)
}

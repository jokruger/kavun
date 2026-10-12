package exec

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// exec.command's members: one function per entry of TypeCommand.Methods that is not a shared member. Every one
// touches the world or configures the command (IMPURE); a refusal from the world is an io error named
// exec.command.<member>.

func commandContext(id member.ID) string {
	return "exec.command." + id.String()
}

// commandDone answers undefined for a statement-like member, or the io error.
func commandDone(id member.ID, err error) (core.Value, error) {
	if err != nil {
		return core.Undefined, errs.NewIOError(commandContext(id), err)
	}
	return core.Undefined, nil
}

// IMPURE by contract
func commandRun(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	return commandDone(id, CommandOf(v).Run())
}

// IMPURE by contract
func commandStart(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	return commandDone(id, CommandOf(v).Start())
}

// IMPURE by contract
func commandWait(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	return commandDone(id, CommandOf(v).Wait())
}

// IMPURE by contract — output() runs the command and answers its stdout
func commandOutput(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	out, err := CommandOf(v).Output()
	if err != nil {
		return core.Undefined, errs.NewIOError(commandContext(id), err)
	}
	return core.NewBytesValue(out, false), nil
}

// IMPURE by contract — combined_output() runs the command and answers its stdout and stderr together
func commandCombinedOutput(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	out, err := CommandOf(v).CombinedOutput()
	if err != nil {
		return core.Undefined, errs.NewIOError(commandContext(id), err)
	}
	return core.NewBytesValue(out, false), nil
}

// IMPURE by contract (configures the command)
func commandSetPath(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "1", len(args))
	}
	s, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(commandContext(id), "first", "string(compatible)", args[0].TypeName())
	}
	CommandOf(v).Path = s
	return core.Undefined, nil
}

// IMPURE by contract (configures the command)
func commandSetDir(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "1", len(args))
	}
	s, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(commandContext(id), "first", "string(compatible)", args[0].TypeName())
	}
	CommandOf(v).Dir = s
	return core.Undefined, nil
}

// IMPURE by contract (configures the command) — set_env(["K=V", …])
func commandSetEnv(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "1", len(args))
	}
	env, err := StringsArg(commandContext(id), "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	CommandOf(v).Env = env
	return core.Undefined, nil
}

// IMPURE by contract — process() is the started command's process; before start() there is none, and that raises
func commandProcess(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(commandContext(id), "0", len(args))
	}
	p := CommandOf(v).Process
	if p == nil {
		return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the command has not been started", commandContext(id)))
	}
	return NewProcessValue(p), nil
}

// StringsArg reads an array of strings argument (a command's argv or environment).
func StringsArg(name, pos string, a core.Value) ([]string, error) {
	if a.Type != value.Array {
		return nil, errs.NewInvalidArgumentTypeError(name, pos, "array(string)", a.TypeName())
	}
	elems := (*core.Array)(a.Ptr).Elements
	out := make([]string, 0, len(elems))
	for i, e := range elems {
		s, ok := e.AsString()
		if !ok {
			return nil, errs.NewInvalidArgumentTypeError(name, fmt.Sprintf("%s[%d]", pos, i), "string(compatible)", e.TypeName())
		}
		out = append(out, s)
	}
	return out, nil
}

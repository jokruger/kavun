package exec

import (
	"syscall"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// exec.process's members: one function per entry of TypeProcess.Methods that is not a shared member. Every one
// touches the world (IMPURE); a refusal from the world is an io error named exec.process.<member>.

func processContext(id member.ID) string {
	return "exec.process." + id.String()
}

// IMPURE by contract (the pid of a live process)
func processPid(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(processContext(id), "0", len(args))
	}
	return core.IntValue(int64(ProcessOf(v).Pid)), nil
}

// IMPURE by contract
func processKill(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(processContext(id), "0", len(args))
	}
	if err := ProcessOf(v).Kill(); err != nil {
		return core.Undefined, errs.NewIOError(processContext(id), err)
	}
	return core.Undefined, nil
}

// IMPURE by contract
func processSignal(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(processContext(id), "1", len(args))
	}
	sig, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(processContext(id), "first", "int(compatible)", args[0].TypeName())
	}
	if err := ProcessOf(v).Signal(syscall.Signal(sig)); err != nil {
		return core.Undefined, errs.NewIOError(processContext(id), err)
	}
	return core.Undefined, nil
}

// IMPURE by contract — wait() blocks until the process ends and answers the frozen {pid, exited, success, status}
func processWait(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(processContext(id), "0", len(args))
	}
	state, err := ProcessOf(v).Wait()
	if err != nil {
		return core.Undefined, errs.NewIOError(processContext(id), err)
	}
	return NewWaitValue(state), nil
}

// IMPURE by contract
func processRelease(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(processContext(id), "0", len(args))
	}
	if err := ProcessOf(v).Release(); err != nil {
		return core.Undefined, errs.NewIOError(processContext(id), err)
	}
	return core.Undefined, nil
}

package os

import (
	"os"
	"runtime"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The os module is the process and its environment: arguments, exit, environment variables, identity, the
// working directory. Files are the fs module's, processes the exec module's, path text the path module's.

func init() {
	m := core.NewModule(value.ModuleOs, "os", false)
	m.Const(members.Platform, core.NewStringValue(runtime.GOOS))
	m.Const(members.Arch, core.NewStringValue(runtime.GOARCH))
	m.Const(members.PathSeparator, core.RuneValue(os.PathSeparator))
	m.Const(members.PathListSeparator, core.RuneValue(os.PathListSeparator))
	m.Const(members.DevNull, core.NewStringValue(os.DevNull))
	m.Fn(members.Args, 0, osArgs, 0, core.Fixed)
	m.Fn(members.Exit, 1, osExit, 1, core.Fixed)
	m.Fn(members.Environ, 2, osEnviron, 0, core.Fixed)
	m.Fn(members.GetEnv, 3, osGetenv, 1, core.Fixed)
	m.Fn(members.SetEnv, 4, osSetenv, 2, core.Fixed)
	m.Fn(members.UnsetEnv, 5, osUnsetenv, 1, core.Fixed)
	m.Fn(members.LookupEnv, 6, osLookupEnv, 1, core.Fixed)
	m.Fn(members.ExpandEnv, 7, osExpandEnv, 1, core.Fixed)
	m.Fn(members.ClearEnv, 8, osClearenv, 0, core.Fixed)
	m.Fn(members.Hostname, 9, osHostname, 0, core.Fixed)
	m.Fn(members.GetWd, 10, osGetwd, 0, core.Fixed)
	m.Fn(members.Chdir, 11, osChdir, 1, core.Fixed)
	m.Fn(members.GetPID, 12, osGetpid, 0, core.Fixed)
	m.Fn(members.GetPpid, 13, osGetppid, 0, core.Fixed)
	m.Fn(members.GetUID, 14, osGetuid, 0, core.Fixed)
	m.Fn(members.GetEuid, 15, osGeteuid, 0, core.Fixed)
	m.Fn(members.GetGid, 16, osGetgid, 0, core.Fixed)
	m.Fn(members.GetEgid, 17, osGetegid, 0, core.Fixed)
	m.Fn(members.GetGroups, 18, osGetgroups, 0, core.Fixed)
	m.Fn(members.GetPageSize, 19, osGetpagesize, 0, core.Fixed)
	// 20..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func osArgs(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.args", "0", len(args))
	}
	arr := make([]core.Value, 0, len(os.Args))
	for _, osArg := range os.Args {
		arr = append(arr, core.NewStringValue(osArg))
	}
	return core.NewArrayValue(arr, false), nil
}

func osExit(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.exit", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.exit", "first", "int(compatible)", args[0].TypeName())
	}
	os.Exit(int(i1))
	return core.Undefined, nil
}

func osEnviron(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.environ", "0", len(args))
	}
	env := os.Environ()
	arr := make([]core.Value, 0, len(env))
	for _, elem := range env {
		arr = append(arr, core.NewStringValue(elem))
	}
	return core.NewArrayValue(arr, false), nil
}

func osGetenv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.get_env", "first", "string(compatible)", args[0].TypeName())
	}
	s := os.Getenv(s1)
	return core.NewStringValue(s), nil
}

func osSetenv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.set_env", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.set_env", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.set_env", "second", "string(compatible)", args[1].TypeName())
	}
	return modkit.Done("os.set_env", os.Setenv(s1, s2))
}

func osUnsetenv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.unset_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.unset_env", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("os.unset_env", os.Unsetenv(s1))
}

func osLookupEnv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.lookup_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.lookup_env", "first", "string(compatible)", args[0].TypeName())
	}
	res, ok := os.LookupEnv(s1)
	if !ok {
		return core.False, nil
	}
	return core.NewStringValue(res), nil
}

func osExpandEnv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.expand_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.expand_env", "first", "string(compatible)", args[0].TypeName())
	}
	s := os.Expand(s1, func(k string) string {
		return os.Getenv(k)
	})
	return core.NewStringValue(s), nil
}

func osClearenv(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.clear_env", "0", len(args))
	}
	os.Clearenv()
	return core.Undefined, nil
}

func osHostname(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.hostname", "0", len(args))
	}
	res, err := os.Hostname()
	if err != nil {
		return modkit.RaiseIO("os.hostname", err)
	}
	return core.NewStringValue(res), nil
}

func osGetwd(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_wd", "0", len(args))
	}
	res, err := os.Getwd()
	if err != nil {
		return modkit.RaiseIO("os.get_wd", err)
	}
	return core.NewStringValue(res), nil
}

func osChdir(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.chdir", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chdir", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("os.chdir", os.Chdir(s1))
}

func osGetpid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_pid", "0", len(args))
	}
	return core.IntValue(int64(os.Getpid())), nil
}

func osGetppid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_ppid", "0", len(args))
	}
	return core.IntValue(int64(os.Getppid())), nil
}

func osGetuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_uid", "0", len(args))
	}
	return core.IntValue(int64(os.Getuid())), nil
}

func osGeteuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_euid", "0", len(args))
	}
	return core.IntValue(int64(os.Geteuid())), nil
}

func osGetgid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_gid", "0", len(args))
	}
	return core.IntValue(int64(os.Getgid())), nil
}

func osGetegid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_egid", "0", len(args))
	}
	return core.IntValue(int64(os.Getegid())), nil
}

func osGetgroups(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_groups", "0", len(args))
	}
	res, err := os.Getgroups()
	if err != nil {
		return modkit.RaiseIO("os.get_groups", err)
	}
	arr := make([]core.Value, 0, len(res))
	for _, v := range res {
		arr = append(arr, core.IntValue(int64(v)))
	}
	return core.NewArrayValue(arr, false), nil
}

func osGetpagesize(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_page_size", "0", len(args))
	}
	return core.IntValue(int64(os.Getpagesize())), nil
}

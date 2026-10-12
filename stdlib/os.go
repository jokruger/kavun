package stdlib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func init() {
	m := core.NewModule(value.ModuleOs, "os", false)
	m.Const(members.PathSeparator, core.RuneValue(os.PathSeparator))
	m.Const(members.PathListSeparator, core.RuneValue(os.PathListSeparator))
	m.Const(members.ORd, core.IntValue(int64(os.O_RDONLY)))
	m.Const(members.OWr, core.IntValue(int64(os.O_WRONLY)))
	m.Const(members.ORdwr, core.IntValue(int64(os.O_RDWR)))
	m.Const(members.OAppend, core.IntValue(int64(os.O_APPEND)))
	m.Const(members.OCreate, core.IntValue(int64(os.O_CREATE)))
	m.Const(members.OExcl, core.IntValue(int64(os.O_EXCL)))
	m.Const(members.OSync, core.IntValue(int64(os.O_SYNC)))
	m.Const(members.OTrunc, core.IntValue(int64(os.O_TRUNC)))
	m.Const(members.ModeDir, core.IntValue(int64(os.ModeDir)))
	m.Const(members.ModeAppend, core.IntValue(int64(os.ModeAppend)))
	m.Const(members.ModeExclusive, core.IntValue(int64(os.ModeExclusive)))
	m.Const(members.ModeTemporary, core.IntValue(int64(os.ModeTemporary)))
	m.Const(members.ModeSymlink, core.IntValue(int64(os.ModeSymlink)))
	m.Const(members.ModeDevice, core.IntValue(int64(os.ModeDevice)))
	m.Const(members.ModeNamedPipe, core.IntValue(int64(os.ModeNamedPipe)))
	m.Const(members.ModeSocket, core.IntValue(int64(os.ModeSocket)))
	m.Const(members.ModeSetUID, core.IntValue(int64(os.ModeSetuid)))
	m.Const(members.ModeSetGui, core.IntValue(int64(os.ModeSetgid)))
	m.Const(members.ModeCharDevice, core.IntValue(int64(os.ModeCharDevice)))
	m.Const(members.ModeSticky, core.IntValue(int64(os.ModeSticky)))
	m.Const(members.ModeType, core.IntValue(int64(os.ModeType)))
	m.Const(members.ModePerm, core.IntValue(int64(os.ModePerm)))
	m.Const(members.SeekSet, core.IntValue(int64(io.SeekStart)))
	m.Const(members.SeekCur, core.IntValue(int64(io.SeekCurrent)))
	m.Const(members.SeekEnd, core.IntValue(int64(io.SeekEnd)))
	m.Const(members.Platform, core.NewStringValue(runtime.GOOS))
	m.Const(members.Arch, core.NewStringValue(runtime.GOARCH))
	m.Const(members.DevNull, core.NewStringValue(os.DevNull))
	// 42..127 reserved
	m.Fn(members.Args, 0, osArgs, 0, core.Fixed)                  // args() => array(string)
	m.Fn(members.Chdir, 1, osChdir, 1, core.Fixed)                // chdir(dir string) => error
	m.Fn(members.Chmod, 2, osChmod, 2, core.Fixed)                // chmod(name string, mode int) => error
	m.Fn(members.Chown, 3, osChown, 3, core.Fixed)                // chown(name string, uid int, gid int) => error
	m.Fn(members.ClearEnv, 4, osClearenv, 0, core.Fixed)          // clear_env()
	m.Fn(members.Environ, 5, osEnviron, 0, core.Fixed)            // environ() => array(string)
	m.Fn(members.Exit, 6, osExit, 1, core.Fixed)                  // exit(code int)
	m.Fn(members.ExpandEnv, 7, osExpandEnv, 1, core.Fixed)        // expand_env(s string) => string
	m.Fn(members.GetEgid, 8, osGetegid, 0, core.Fixed)            // get_egid() => int
	m.Fn(members.GetEnv, 9, osGetenv, 1, core.Fixed)              // get_env(s string) => string
	m.Fn(members.GetEuid, 10, osGeteuid, 0, core.Fixed)           // get_euid() => int
	m.Fn(members.GetGid, 11, osGetgid, 0, core.Fixed)             // get_gid() => int
	m.Fn(members.GetGroups, 12, osGetgroups, 0, core.Fixed)       // get_groups() => array(string)/error
	m.Fn(members.GetPageSize, 13, osGetpagesize, 0, core.Fixed)   // get_page_size() => int
	m.Fn(members.GetPID, 14, osGetpid, 0, core.Fixed)             // get_pid() => int
	m.Fn(members.GetPpid, 15, osGetppid, 0, core.Fixed)           // get_ppid() => int
	m.Fn(members.GetUID, 16, osGetuid, 0, core.Fixed)             // get_uid() => int
	m.Fn(members.GetWd, 17, osGetwd, 0, core.Fixed)               // get_wd() => string/error
	m.Fn(members.Hostname, 18, osHostname, 0, core.Fixed)         // hostname() => string/error
	m.Fn(members.Lchown, 19, osLchown, 3, core.Fixed)             // lchown(name string, uid int, gid int) => error
	m.Fn(members.Link, 20, osLink, 2, core.Fixed)                 // link(oldName string, newName string) => error
	m.Fn(members.LookupEnv, 21, osLookupEnv, 1, core.Fixed)       // lookup_env(key string) => string/false
	m.Fn(members.Mkdir, 22, osMkdir, 2, core.Fixed)               // mkdir(name string, perm int) => error
	m.Fn(members.MkdirAll, 23, osMkdirAll, 2, core.Fixed)         // mkdir_all(name string, perm int) => error
	m.Fn(members.ReadLink, 24, osReadlink, 1, core.Fixed)         // read_link(name string) => string/error
	m.Fn(members.Remove, 25, osRemove, 1, core.Fixed)             // remove(name string) => error
	m.Fn(members.RemoveAll, 26, osRemoveAll, 1, core.Fixed)       // remove_all(name string) => error
	m.Fn(members.Rename, 27, osRename, 2, core.Fixed)             // rename(oldPath string, newPath string) => error
	m.Fn(members.SetEnv, 28, osSetenv, 2, core.Fixed)             // set_env(key string, value string) => error
	m.Fn(members.Symlink, 29, osSymlink, 2, core.Fixed)           // symlink(oldName string newName string) => error
	m.Fn(members.TempDir, 30, osTempDir, 0, core.Fixed)           // temp_dir() => string
	m.Fn(members.Truncate, 31, osTruncate, 2, core.Fixed)         // truncate(name string, size int) => error
	m.Fn(members.UnsetEnv, 32, osUnsetenv, 1, core.Fixed)         // unset_env(key string) => error
	m.Fn(members.Create, 33, osCreate, 1, core.Fixed)             // create(name string) => idict(file)/error
	m.Fn(members.Open, 34, osOpen, 1, core.Fixed)                 // open(name string) => idict(file)/error
	m.Fn(members.OpenFile, 35, osOpenFile, 3, core.Fixed)         // open_file(name string, flag int, perm int) => idict(file)/error
	m.Fn(members.FindProcess, 36, osFindProcess, 1, core.Fixed)   // find_process(pid int) => idict(process)/error
	m.Fn(members.StartProcess, 37, osStartProcess, 4, core.Fixed) // start_process(name string, argv array(string), dir string, env array(string)) => idict(process)/error
	m.Fn(members.ExecLookPath, 38, execLookPath, 1, core.Fixed)   // exec_look_path(file) => string/error
	m.Fn(members.Exec, 39, osExec, 1, core.Variadic)              // exec(name, args...) => command
	m.Fn(members.Stat, 40, osStat, 1, core.Fixed)                 // stat(name) => idict(fileinfo)/error
	m.Fn(members.ReadFile, 41, osReadFile, 1, core.Fixed)         // readfile(name) => array(byte)/error
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func osChmod(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.chmod", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chmod", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chmod", "second", "int(compatible)", args[1].TypeName())
	}
	return done("os.chmod", os.Chmod(s1, os.FileMode(i2)))
}

func osMkdir(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.mkdir", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.mkdir", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.mkdir", "second", "int(compatible)", args[1].TypeName())
	}
	return done("os.mkdir", os.Mkdir(s1, os.FileMode(i2)))
}

func osMkdirAll(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.mkdir_all", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.mkdir_all", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.mkdir_all", "second", "int(compatible)", args[1].TypeName())
	}
	return done("os.mkdir_all", os.MkdirAll(s1, os.FileMode(i2)))
}

func osLchown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.lchown", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.lchown", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.lchown", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.lchown", "third", "int(compatible)", args[2].TypeName())
	}
	return done("os.lchown", os.Lchown(s1, int(i2), int(i3)))
}

func osChown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.chown", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chown", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chown", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chown", "third", "int(compatible)", args[2].TypeName())
	}
	return done("os.chown", os.Chown(s1, int(i2), int(i3)))
}

func osTruncate(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.truncate", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.truncate", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.truncate", "second", "int(compatible)", args[1].TypeName())
	}
	return done("os.truncate", os.Truncate(s1, i2))
}

func osSymlink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.symlink", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.symlink", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.symlink", "second", "string(compatible)", args[1].TypeName())
	}
	return done("os.symlink", os.Symlink(s1, s2))
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
	return done("os.set_env", os.Setenv(s1, s2))
}

func osRename(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.rename", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.rename", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.rename", "second", "string(compatible)", args[1].TypeName())
	}
	return done("os.rename", os.Rename(s1, s2))
}

func osLink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.link", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.link", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.link", "second", "string(compatible)", args[1].TypeName())
	}
	return done("os.link", os.Link(s1, s2))
}

func osUnsetenv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.unset_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.unset_env", "first", "string(compatible)", args[0].TypeName())
	}
	return done("os.unset_env", os.Unsetenv(s1))
}

func osRemoveAll(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.remove_all", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.remove_all", "first", "string(compatible)", args[0].TypeName())
	}
	return done("os.remove_all", os.RemoveAll(s1))
}

func osRemove(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.remove", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.remove", "first", "string(compatible)", args[0].TypeName())
	}
	return done("os.remove", os.Remove(s1))
}

func osChdir(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.chdir", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chdir", "first", "string(compatible)", args[0].TypeName())
	}
	return done("os.chdir", os.Chdir(s1))
}

func execLookPath(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.exec_look_path", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec_look_path", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := exec.LookPath(s1)
	if err != nil {
		return raiseIO("os.exec_look_path", err)
	}
	return core.NewStringValue(res), nil
}

func osReadlink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.read_link", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.read_link", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Readlink(s1)
	if err != nil {
		return raiseIO("os.read_link", err)
	}
	return core.NewStringValue(res), nil
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

func osGetgroups(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_groups", "0", len(args))
	}
	res, err := os.Getgroups()
	if err != nil {
		return raiseIO("os.get_groups", err)
	}
	arr := make([]core.Value, 0, len(res))
	for _, v := range res {
		arr = append(arr, core.IntValue(int64(v)))
	}
	return core.NewArrayValue(arr, false), nil
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

func osHostname(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.hostname", "0", len(args))
	}
	res, err := os.Hostname()
	if err != nil {
		return raiseIO("os.hostname", err)
	}
	return core.NewStringValue(res), nil
}

func osGetwd(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_wd", "0", len(args))
	}
	res, err := os.Getwd()
	if err != nil {
		return raiseIO("os.get_wd", err)
	}
	return core.NewStringValue(res), nil
}

func osTempDir(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.temp_dir", "0", len(args))
	}
	s := os.TempDir()
	return core.NewStringValue(s), nil
}

func osGetuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_uid", "0", len(args))
	}
	return core.IntValue(int64(os.Getuid())), nil
}

func osGetppid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_ppid", "0", len(args))
	}
	return core.IntValue(int64(os.Getppid())), nil
}

func osGetpid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_pid", "0", len(args))
	}
	return core.IntValue(int64(os.Getpid())), nil
}

func osGetpagesize(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_page_size", "0", len(args))
	}
	return core.IntValue(int64(os.Getpagesize())), nil
}

func osGetgid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_gid", "0", len(args))
	}
	return core.IntValue(int64(os.Getgid())), nil
}

func osGeteuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_euid", "0", len(args))
	}
	return core.IntValue(int64(os.Geteuid())), nil
}

func osGetegid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_egid", "0", len(args))
	}
	return core.IntValue(int64(os.Getegid())), nil
}

func osClearenv(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.clear_env", "0", len(args))
	}
	os.Clearenv()
	return core.Undefined, nil
}

func osReadFile(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.read_file", "1", len(args))
	}
	fname, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.read_file", "first", "string(compatible)", args[0].TypeName())
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return raiseIO("os.read_file", err)
	}
	return core.NewBytesValue(bytes, false), nil
}

func osStat(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.stat", "1", len(args))
	}

	fname, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.stat", "first", "string(compatible)", args[0].TypeName())
	}

	stat, err := os.Stat(fname)
	if err != nil {
		return raiseIO("os.stat", err)
	}

	fstat := core.NewRecordValue(map[string]core.Value{
		"name":      core.NewStringValue(stat.Name()),
		"mtime":     core.NewTimeValue(stat.ModTime()),
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.BoolValue(stat.IsDir()),
	}, true)

	return fstat, nil
}

func osCreate(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.create", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.create", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Create(s1)
	if err != nil {
		return raiseIO("os.create", err)
	}
	return makeOSFile(vm, res)
}

func osOpen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.open", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.open", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Open(s1)
	if err != nil {
		return raiseIO("os.open", err)
	}
	return makeOSFile(vm, res)
}

func osOpenFile(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.open_file", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.open_file", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.open_file", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.open_file", "third", "int(compatible)", args[2].TypeName())
	}
	res, err := os.OpenFile(s1, int(i2), os.FileMode(i3))
	if err != nil {
		return raiseIO("os.open_file", err)
	}
	return makeOSFile(vm, res)
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

func osExec(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.exec", "at least 1", len(args))
	}
	name, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec", "first", "string(compatible)", args[0].TypeName())
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := arg.AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.exec", fmt.Sprintf("args[%d]", idx), "string(compatible)", arg.TypeName())
		}
		execArgs = append(execArgs, execArg)
	}
	return makeOSExecCommand(vm, exec.Command(name, execArgs...))
}

func osFindProcess(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.find_process", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.find_process", "first", "int(compatible)", args[0].TypeName())
	}
	proc, err := os.FindProcess(int(i1))
	if err != nil {
		return raiseIO("os.find_process", err)
	}
	return makeOSProcess(vm, proc)
}

func osStartProcess(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.start_process", "4", len(args))
	}
	name, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.start_process", "first", "string(compatible)", args[0].TypeName())
	}
	var argv []string
	var err error
	if args[1].Type != value.Array {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.start_process", "second", "array(string)", args[1].TypeName())
	}
	arr := (*core.Array)(args[1].Ptr)
	argv, err = stringArray(arr.Elements, "second")
	if err != nil {
		return core.Undefined, err
	}

	dir, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.start_process", "third", "string(compatible)", args[2].TypeName())
	}

	var env []string
	if args[3].Type != value.Array {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.start_process", "fourth", "array(string)", args[3].TypeName())
	}
	arr = (*core.Array)(args[3].Ptr)
	env, err = stringArray(arr.Elements, "fourth")
	if err != nil {
		return core.Undefined, err
	}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Dir: dir,
		Env: env,
	})
	if err != nil {
		return raiseIO("os.start_process", err)
	}
	return makeOSProcess(vm, proc)
}

func stringArray(arr []core.Value, argName string) ([]string, error) {
	ss := make([]string, 0, len(arr))
	for idx, elem := range arr {
		str, ok := elem.AsString()
		if !ok {
			return nil, errs.NewInvalidArgumentTypeError("os.start_process", fmt.Sprintf("%s[%d]", argName, idx), "string(compatible)", elem.TypeName())
		}
		ss = append(ss, str)
	}
	return ss, nil
}

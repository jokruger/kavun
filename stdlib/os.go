package stdlib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

var osModule = map[string]core.Value{
	"platform":            core.NewStringValue(runtime.GOOS),
	"arch":                core.NewStringValue(runtime.GOARCH),
	"dev_null":            core.NewStringValue(os.DevNull),
	"path_separator":      core.RuneValue(os.PathSeparator),
	"path_list_separator": core.RuneValue(os.PathListSeparator),
	"o_rd":                core.IntValue(int64(os.O_RDONLY)),
	"o_wr":                core.IntValue(int64(os.O_WRONLY)),
	"o_rdwr":              core.IntValue(int64(os.O_RDWR)),
	"o_append":            core.IntValue(int64(os.O_APPEND)),
	"o_create":            core.IntValue(int64(os.O_CREATE)),
	"o_excl":              core.IntValue(int64(os.O_EXCL)),
	"o_sync":              core.IntValue(int64(os.O_SYNC)),
	"o_trunc":             core.IntValue(int64(os.O_TRUNC)),
	"mode_dir":            core.IntValue(int64(os.ModeDir)),
	"mode_append":         core.IntValue(int64(os.ModeAppend)),
	"mode_exclusive":      core.IntValue(int64(os.ModeExclusive)),
	"mode_temporary":      core.IntValue(int64(os.ModeTemporary)),
	"mode_symlink":        core.IntValue(int64(os.ModeSymlink)),
	"mode_device":         core.IntValue(int64(os.ModeDevice)),
	"mode_named_pipe":     core.IntValue(int64(os.ModeNamedPipe)),
	"mode_socket":         core.IntValue(int64(os.ModeSocket)),
	"mode_set_uid":        core.IntValue(int64(os.ModeSetuid)),
	"mode_set_gui":        core.IntValue(int64(os.ModeSetgid)),
	"mode_char_device":    core.IntValue(int64(os.ModeCharDevice)),
	"mode_sticky":         core.IntValue(int64(os.ModeSticky)),
	"mode_type":           core.IntValue(int64(os.ModeType)),
	"mode_perm":           core.IntValue(int64(os.ModePerm)),
	"seek_set":            core.IntValue(int64(io.SeekStart)),
	"seek_cur":            core.IntValue(int64(io.SeekCurrent)),
	"seek_end":            core.IntValue(int64(io.SeekEnd)),

	"args":           core.NewBuiltinFunctionValue("args", osArgs, 0, false),                  // args() => array(string)
	"chdir":          core.NewBuiltinFunctionValue("chdir", osChdir, 1, false),                // chdir(dir string) => error
	"chmod":          core.NewBuiltinFunctionValue("chmod", osChmod, 2, false),                // chmod(name string, mode int) => error
	"chown":          core.NewBuiltinFunctionValue("chown", osChown, 3, false),                // chown(name string, uid int, gid int) => error
	"clear_env":      core.NewBuiltinFunctionValue("clear_env", osClearenv, 0, false),         // clear_env()
	"environ":        core.NewBuiltinFunctionValue("environ", osEnviron, 0, false),            // environ() => array(string)
	"exit":           core.NewBuiltinFunctionValue("exit", osExit, 1, false),                  // exit(code int)
	"expand_env":     core.NewBuiltinFunctionValue("expand_env", osExpandEnv, 1, false),       // expand_env(s string) => string
	"get_egid":       core.NewBuiltinFunctionValue("get_egid", osGetegid, 0, false),           // get_egid() => int
	"get_env":        core.NewBuiltinFunctionValue("get_env", osGetenv, 1, false),             // get_env(s string) => string
	"get_euid":       core.NewBuiltinFunctionValue("get_euid", osGeteuid, 0, false),           // get_euid() => int
	"get_gid":        core.NewBuiltinFunctionValue("get_gid", osGetgid, 0, false),             // get_gid() => int
	"get_groups":     core.NewBuiltinFunctionValue("get_groups", osGetgroups, 0, false),       // get_groups() => array(string)/error
	"get_page_size":  core.NewBuiltinFunctionValue("get_page_size", osGetpagesize, 0, false),  // get_page_size() => int
	"get_pid":        core.NewBuiltinFunctionValue("get_pid", osGetpid, 0, false),             // get_pid() => int
	"get_ppid":       core.NewBuiltinFunctionValue("get_ppid", osGetppid, 0, false),           // get_ppid() => int
	"get_uid":        core.NewBuiltinFunctionValue("get_uid", osGetuid, 0, false),             // get_uid() => int
	"get_wd":         core.NewBuiltinFunctionValue("get_wd", osGetwd, 0, false),               // get_wd() => string/error
	"hostname":       core.NewBuiltinFunctionValue("hostname", osHostname, 0, false),          // hostname() => string/error
	"lchown":         core.NewBuiltinFunctionValue("lchown", osLchown, 3, false),              // lchown(name string, uid int, gid int) => error
	"link":           core.NewBuiltinFunctionValue("link", osLink, 2, false),                  // link(oldName string, newName string) => error
	"lookup_env":     core.NewBuiltinFunctionValue("lookup_env", osLookupEnv, 1, false),       // lookup_env(key string) => string/false
	"mkdir":          core.NewBuiltinFunctionValue("mkdir", osMkdir, 2, false),                // mkdir(name string, perm int) => error
	"mkdir_all":      core.NewBuiltinFunctionValue("mkdir_all", osMkdirAll, 2, false),         // mkdir_all(name string, perm int) => error
	"read_link":      core.NewBuiltinFunctionValue("read_link", osReadlink, 1, false),         // read_link(name string) => string/error
	"remove":         core.NewBuiltinFunctionValue("remove", osRemove, 1, false),              // remove(name string) => error
	"remove_all":     core.NewBuiltinFunctionValue("remove_all", osRemoveAll, 1, false),       // remove_all(name string) => error
	"rename":         core.NewBuiltinFunctionValue("rename", osRename, 2, false),              // rename(oldPath string, newPath string) => error
	"set_env":        core.NewBuiltinFunctionValue("set_env", osSetenv, 2, false),             // set_env(key string, value string) => error
	"symlink":        core.NewBuiltinFunctionValue("symlink", osSymlink, 2, false),            // symlink(oldName string newName string) => error
	"temp_dir":       core.NewBuiltinFunctionValue("temp_dir", osTempDir, 0, false),           // temp_dir() => string
	"truncate":       core.NewBuiltinFunctionValue("truncate", osTruncate, 2, false),          // truncate(name string, size int) => error
	"unset_env":      core.NewBuiltinFunctionValue("unset_env", osUnsetenv, 1, false),         // unset_env(key string) => error
	"create":         core.NewBuiltinFunctionValue("create", osCreate, 1, false),              // create(name string) => imap(file)/error
	"open":           core.NewBuiltinFunctionValue("open", osOpen, 1, false),                  // open(name string) => imap(file)/error
	"open_file":      core.NewBuiltinFunctionValue("open_file", osOpenFile, 3, false),         // open_file(name string, flag int, perm int) => imap(file)/error
	"find_process":   core.NewBuiltinFunctionValue("find_process", osFindProcess, 1, false),   // find_process(pid int) => imap(process)/error
	"start_process":  core.NewBuiltinFunctionValue("start_process", osStartProcess, 4, false), // start_process(name string, argv array(string), dir string, env array(string)) => imap(process)/error
	"exec_look_path": core.NewBuiltinFunctionValue("exec_look_path", execLookPath, 1, false),  // exec_look_path(file) => string/error
	"exec":           core.NewBuiltinFunctionValue("exec", osExec, 1, true),                   // exec(name, args...) => command
	"stat":           core.NewBuiltinFunctionValue("stat", osStat, 1, false),                  // stat(name) => imap(fileinfo)/error
	"read_file":      core.NewBuiltinFunctionValue("read_file", osReadFile, 1, false),         // readfile(name) => array(byte)/error
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
	return wrapError(vm, os.Chmod(s1, os.FileMode(i2)))
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
	return wrapError(vm, os.Mkdir(s1, os.FileMode(i2)))
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
	return wrapError(vm, os.MkdirAll(s1, os.FileMode(i2)))
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
	return wrapError(vm, os.Lchown(s1, int(i2), int(i3)))
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
	return wrapError(vm, os.Chown(s1, int(i2), int(i3)))
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
	return wrapError(vm, os.Truncate(s1, i2))
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
	return wrapError(vm, os.Symlink(s1, s2))
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
	return wrapError(vm, os.Setenv(s1, s2))
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
	return wrapError(vm, os.Rename(s1, s2))
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
	return wrapError(vm, os.Link(s1, s2))
}

func osUnsetenv(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.unset_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.unset_env", "first", "string(compatible)", args[0].TypeName())
	}
	return wrapError(vm, os.Unsetenv(s1))
}

func osRemoveAll(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.remove_all", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.remove_all", "first", "string(compatible)", args[0].TypeName())
	}
	return wrapError(vm, os.RemoveAll(s1))
}

func osRemove(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.remove", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.remove", "first", "string(compatible)", args[0].TypeName())
	}
	return wrapError(vm, os.Remove(s1))
}

func osChdir(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.chdir", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chdir", "first", "string(compatible)", args[0].TypeName())
	}
	return wrapError(vm, os.Chdir(s1))
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
		return wrapError(vm, err)
	}
	return vm.Allocator().NewStringValue(res), nil
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
		return wrapError(vm, err)
	}
	return vm.Allocator().NewStringValue(res), nil
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
	return vm.Allocator().NewStringValue(s), nil
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
		return wrapError(vm, err)
	}
	alloc := vm.Allocator()
	arr := alloc.NewArray(len(res), false)
	for _, v := range res {
		arr = append(arr, core.IntValue(int64(v)))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func osEnviron(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.environ", "0", len(args))
	}
	env := os.Environ()
	alloc := vm.Allocator()
	arr := alloc.NewArray(len(env), false)
	for _, elem := range env {
		t := alloc.NewStringValue(elem)
		arr = append(arr, t)
	}
	return alloc.NewArrayValue(arr, false), nil
}

func osHostname(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.hostname", "0", len(args))
	}
	res, err := os.Hostname()
	if err != nil {
		return wrapError(vm, err)
	}
	return vm.Allocator().NewStringValue(res), nil
}

func osGetwd(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.get_wd", "0", len(args))
	}
	res, err := os.Getwd()
	if err != nil {
		return wrapError(vm, err)
	}
	return vm.Allocator().NewStringValue(res), nil
}

func osTempDir(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.temp_dir", "0", len(args))
	}
	s := os.TempDir()
	return vm.Allocator().NewStringValue(s), nil
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
		return wrapError(vm, err)
	}
	return vm.Allocator().NewBytesValue(bytes), nil
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
		return wrapError(vm, err)
	}

	alloc := vm.Allocator()
	name := alloc.NewStringValue(stat.Name())
	d := alloc.NewTime()
	*d = stat.ModTime()
	mt := core.TimeValue(d)

	fstat := alloc.NewRecordValue(map[string]core.Value{
		"name":      name,
		"mtime":     mt,
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
		return wrapError(vm, err)
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
		return wrapError(vm, err)
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
		return wrapError(vm, err)
	}
	return makeOSFile(vm, res)
}

func osArgs(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("os.args", "0", len(args))
	}
	alloc := vm.Allocator()
	arr := alloc.NewArray(len(os.Args), false)
	for _, osArg := range os.Args {
		t := alloc.NewStringValue(osArg)
		arr = append(arr, t)
	}
	return alloc.NewArrayValue(arr, false), nil
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
	return vm.Allocator().NewStringValue(res), nil
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
	return vm.Allocator().NewStringValue(s), nil
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
		return wrapError(vm, err)
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
	if args[1].Type != core.VT_ARRAY {
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
	if args[3].Type != core.VT_ARRAY {
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
		return wrapError(vm, err)
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

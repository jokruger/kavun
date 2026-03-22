package stdlib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var osModule = map[string]core.Object{
	"platform":            value.NewStaticString(runtime.GOOS),
	"arch":                value.NewStaticString(runtime.GOARCH),
	"o_rdonly":            value.NewStaticInt(int64(os.O_RDONLY)),
	"o_wronly":            value.NewStaticInt(int64(os.O_WRONLY)),
	"o_rdwr":              value.NewStaticInt(int64(os.O_RDWR)),
	"o_append":            value.NewStaticInt(int64(os.O_APPEND)),
	"o_create":            value.NewStaticInt(int64(os.O_CREATE)),
	"o_excl":              value.NewStaticInt(int64(os.O_EXCL)),
	"o_sync":              value.NewStaticInt(int64(os.O_SYNC)),
	"o_trunc":             value.NewStaticInt(int64(os.O_TRUNC)),
	"mode_dir":            value.NewStaticInt(int64(os.ModeDir)),
	"mode_append":         value.NewStaticInt(int64(os.ModeAppend)),
	"mode_exclusive":      value.NewStaticInt(int64(os.ModeExclusive)),
	"mode_temporary":      value.NewStaticInt(int64(os.ModeTemporary)),
	"mode_symlink":        value.NewStaticInt(int64(os.ModeSymlink)),
	"mode_device":         value.NewStaticInt(int64(os.ModeDevice)),
	"mode_named_pipe":     value.NewStaticInt(int64(os.ModeNamedPipe)),
	"mode_socket":         value.NewStaticInt(int64(os.ModeSocket)),
	"mode_setuid":         value.NewStaticInt(int64(os.ModeSetuid)),
	"mode_setgui":         value.NewStaticInt(int64(os.ModeSetgid)),
	"mode_char_device":    value.NewStaticInt(int64(os.ModeCharDevice)),
	"mode_sticky":         value.NewStaticInt(int64(os.ModeSticky)),
	"mode_type":           value.NewStaticInt(int64(os.ModeType)),
	"mode_perm":           value.NewStaticInt(int64(os.ModePerm)),
	"path_separator":      value.NewStaticChar(os.PathSeparator),
	"path_list_separator": value.NewStaticChar(os.PathListSeparator),
	"dev_null":            value.NewStaticString(os.DevNull),
	"seek_set":            value.NewStaticInt(int64(io.SeekStart)),
	"seek_cur":            value.NewStaticInt(int64(io.SeekCurrent)),
	"seek_end":            value.NewStaticInt(int64(io.SeekEnd)),

	"args":           value.NewStaticBuiltinFunction("args", osArgs, 0, false),                  // args() => array(string)
	"chdir":          value.NewStaticBuiltinFunction("chdir", osChdir, 1, false),                // chdir(dir string) => error
	"chmod":          value.NewStaticBuiltinFunction("chmod", osChmod, 2, false),                // chmod(name string, mode int) => error
	"chown":          value.NewStaticBuiltinFunction("chown", osChown, 3, false),                // chown(name string, uid int, gid int) => error
	"clearenv":       value.NewStaticBuiltinFunction("clearenv", osClearenv, 0, false),          // clearenv()
	"environ":        value.NewStaticBuiltinFunction("environ", osEnviron, 0, false),            // environ() => array(string)
	"exit":           value.NewStaticBuiltinFunction("exit", osExit, 1, false),                  // exit(code int)
	"expand_env":     value.NewStaticBuiltinFunction("expand_env", osExpandEnv, 1, false),       // expand_env(s string) => string
	"getegid":        value.NewStaticBuiltinFunction("getegid", osGetegid, 0, false),            // getegid() => int
	"getenv":         value.NewStaticBuiltinFunction("getenv", osGetenv, 1, false),              // getenv(s string) => string
	"geteuid":        value.NewStaticBuiltinFunction("geteuid", osGeteuid, 0, false),            // geteuid() => int
	"getgid":         value.NewStaticBuiltinFunction("getgid", osGetgid, 0, false),              // getgid() => int
	"getgroups":      value.NewStaticBuiltinFunction("getgroups", osGetgroups, 0, false),        // getgroups() => array(string)/error
	"getpagesize":    value.NewStaticBuiltinFunction("getpagesize", osGetpagesize, 0, false),    // getpagesize() => int
	"getpid":         value.NewStaticBuiltinFunction("getpid", osGetpid, 0, false),              // getpid() => int
	"getppid":        value.NewStaticBuiltinFunction("getppid", osGetppid, 0, false),            // getppid() => int
	"getuid":         value.NewStaticBuiltinFunction("getuid", osGetuid, 0, false),              // getuid() => int
	"getwd":          value.NewStaticBuiltinFunction("getwd", osGetwd, 0, false),                // getwd() => string/error
	"hostname":       value.NewStaticBuiltinFunction("hostname", osHostname, 0, false),          // hostname() => string/error
	"lchown":         value.NewStaticBuiltinFunction("lchown", osLchown, 3, false),              // lchown(name string, uid int, gid int) => error
	"link":           value.NewStaticBuiltinFunction("link", osLink, 2, false),                  // link(oldname string, newname string) => error
	"lookup_env":     value.NewStaticBuiltinFunction("lookup_env", osLookupEnv, 1, false),       // lookup_env(key string) => string/false
	"mkdir":          value.NewStaticBuiltinFunction("mkdir", osMkdir, 2, false),                // mkdir(name string, perm int) => error
	"mkdir_all":      value.NewStaticBuiltinFunction("mkdir_all", osMkdirAll, 2, false),         // mkdir_all(name string, perm int) => error
	"readlink":       value.NewStaticBuiltinFunction("readlink", osReadlink, 1, false),          // readlink(name string) => string/error
	"remove":         value.NewStaticBuiltinFunction("remove", osRemove, 1, false),              // remove(name string) => error
	"remove_all":     value.NewStaticBuiltinFunction("remove_all", osRemoveAll, 1, false),       // remove_all(name string) => error
	"rename":         value.NewStaticBuiltinFunction("rename", osRename, 2, false),              // rename(oldpath string, newpath string) => error
	"setenv":         value.NewStaticBuiltinFunction("setenv", osSetenv, 2, false),              // setenv(key string, value string) => error
	"symlink":        value.NewStaticBuiltinFunction("symlink", osSymlink, 2, false),            // symlink(oldname string newname string) => error
	"temp_dir":       value.NewStaticBuiltinFunction("temp_dir", osTempDir, 0, false),           // temp_dir() => string
	"truncate":       value.NewStaticBuiltinFunction("truncate", osTruncate, 2, false),          // truncate(name string, size int) => error
	"unsetenv":       value.NewStaticBuiltinFunction("unsetenv", osUnsetenv, 1, false),          // unsetenv(key string) => error
	"create":         value.NewStaticBuiltinFunction("create", osCreate, 1, false),              // create(name string) => imap(file)/error
	"open":           value.NewStaticBuiltinFunction("open", osOpen, 1, false),                  // open(name string) => imap(file)/error
	"open_file":      value.NewStaticBuiltinFunction("open_file", osOpenFile, 3, false),         // open_file(name string, flag int, perm int) => imap(file)/error
	"find_process":   value.NewStaticBuiltinFunction("find_process", osFindProcess, 1, false),   // find_process(pid int) => imap(process)/error
	"start_process":  value.NewStaticBuiltinFunction("start_process", osStartProcess, 4, false), // start_process(name string, argv array(string), dir string, env array(string)) => imap(process)/error
	"exec_look_path": value.NewStaticBuiltinFunction("exec_look_path", execLookPath, 1, false),  // exec_look_path(file) => string/error
	"exec":           value.NewStaticBuiltinFunction("exec", osExec, 1, true),                   // exec(name, args...) => command
	"stat":           value.NewStaticBuiltinFunction("stat", osStat, 1, false),                  // stat(name) => imap(fileinfo)/error
	"read_file":      value.NewStaticBuiltinFunction("read_file", osReadFile, 1, false),         // readfile(name) => array(byte)/error
}

func osChmod(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.chmod", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chmod", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chmod", "second", "int(compatible)", args[1])
	}
	return wrapError(vm, os.Chmod(s1, os.FileMode(i2))), nil
}

func osMkdir(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.mkdir", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.mkdir", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.mkdir", "second", "int(compatible)", args[1])
	}
	return wrapError(vm, os.Mkdir(s1, os.FileMode(i2))), nil
}

func osMkdirAll(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.mkdir_all", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.mkdir_all", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.mkdir_all", "second", "int(compatible)", args[1])
	}
	return wrapError(vm, os.MkdirAll(s1, os.FileMode(i2))), nil
}

func osLchown(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, core.NewWrongNumArgumentsError("os.lchown", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.lchown", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.lchown", "second", "int(compatible)", args[1])
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.lchown", "third", "int(compatible)", args[2])
	}
	return wrapError(vm, os.Lchown(s1, int(i2), int(i3))), nil
}

func osChown(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, core.NewWrongNumArgumentsError("os.chown", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chown", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chown", "second", "int(compatible)", args[1])
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chown", "third", "int(compatible)", args[2])
	}
	return wrapError(vm, os.Chown(s1, int(i2), int(i3))), nil
}

func osTruncate(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.truncate", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.truncate", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.truncate", "second", "int(compatible)", args[1])
	}
	return wrapError(vm, os.Truncate(s1, i2)), nil
}

func osSymlink(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.symlink", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.symlink", "first", "string(compatible)", args[0])
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.symlink", "second", "string(compatible)", args[1])
	}
	return wrapError(vm, os.Symlink(s1, s2)), nil
}

func osSetenv(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.setenv", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.setenv", "first", "string(compatible)", args[0])
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.setenv", "second", "string(compatible)", args[1])
	}
	return wrapError(vm, os.Setenv(s1, s2)), nil
}

func osRename(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.rename", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.rename", "first", "string(compatible)", args[0])
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.rename", "second", "string(compatible)", args[1])
	}
	return wrapError(vm, os.Rename(s1, s2)), nil
}

func osLink(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("os.link", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.link", "first", "string(compatible)", args[0])
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.link", "second", "string(compatible)", args[1])
	}
	return wrapError(vm, os.Link(s1, s2)), nil
}

func osUnsetenv(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.unsetenv", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.unsetenv", "first", "string(compatible)", args[0])
	}
	return wrapError(vm, os.Unsetenv(s1)), nil
}

func osRemoveAll(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.remove_all", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.remove_all", "first", "string(compatible)", args[0])
	}
	return wrapError(vm, os.RemoveAll(s1)), nil
}

func osRemove(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.remove", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.remove", "first", "string(compatible)", args[0])
	}
	return wrapError(vm, os.Remove(s1)), nil
}

func osChdir(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.chdir", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.chdir", "first", "string(compatible)", args[0])
	}
	return wrapError(vm, os.Chdir(s1)), nil
}

func execLookPath(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.exec_look_path", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.exec_look_path", "first", "string(compatible)", args[0])
	}
	res, err := exec.LookPath(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.exec_look_path")
	}
	return vm.Allocator().NewString(res), nil
}

func osReadlink(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.readlink", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.readlink", "first", "string(compatible)", args[0])
	}
	res, err := os.Readlink(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.readlink")
	}
	return vm.Allocator().NewString(res), nil
}

func osGetenv(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.getenv", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.getenv", "first", "string(compatible)", args[0])
	}
	s := os.Getenv(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.getenv")
	}
	return vm.Allocator().NewString(s), nil
}

func osExit(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.exit", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.exit", "first", "int(compatible)", args[0])
	}
	os.Exit(int(i1))
	return vm.Allocator().NewUndefined(), nil
}

func osGetgroups(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getgroups", "0", len(args))
	}
	res, err := os.Getgroups()
	if err != nil {
		return wrapError(vm, err), nil
	}
	arr := make([]core.Object, 0, len(res))
	alloc := vm.Allocator()
	for _, v := range res {
		arr = append(arr, alloc.NewInt(int64(v)))
	}
	return alloc.NewArray(arr, false), nil
}

func osEnviron(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.environ", "0", len(args))
	}
	env := os.Environ()
	arr := make([]core.Object, 0, len(env))
	alloc := vm.Allocator()
	for _, elem := range env {
		if len(elem) > core.MaxStringLen {
			return nil, core.NewStringLimitError("os.environ")
		}
		arr = append(arr, alloc.NewString(elem))
	}
	return alloc.NewArray(arr, false), nil
}

func osHostname(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.hostname", "0", len(args))
	}
	res, err := os.Hostname()
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.hostname")
	}
	return vm.Allocator().NewString(res), nil
}

func osGetwd(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getwd", "0", len(args))
	}
	res, err := os.Getwd()
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.getwd")
	}
	return vm.Allocator().NewString(res), nil
}

func osTempDir(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.temp_dir", "0", len(args))
	}
	s := os.TempDir()
	if len(s) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.temp_dir")
	}
	return vm.Allocator().NewString(s), nil
}

func osGetuid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getuid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getuid())), nil
}

func osGetppid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getppid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getppid())), nil
}

func osGetpid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getpid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getpid())), nil
}

func osGetpagesize(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getpagesize", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getpagesize())), nil
}

func osGetgid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getgid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getgid())), nil
}

func osGeteuid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.geteuid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Geteuid())), nil
}

func osGetegid(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.getegid", "0", len(args))
	}
	return vm.Allocator().NewInt(int64(os.Getegid())), nil
}

func osClearenv(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.clearenv", "0", len(args))
	}
	os.Clearenv()
	return vm.Allocator().NewUndefined(), nil
}

func osReadFile(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.read_file", "1", len(args))
	}
	fname, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.read_file", "first", "string(compatible)", args[0])
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(bytes) > core.MaxBytesLen {
		return nil, core.NewBytesLimitError("os.read_file")
	}
	return vm.Allocator().NewBytes(bytes), nil
}

func osStat(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.stat", "1", len(args))
	}
	fname, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.stat", "first", "string(compatible)", args[0])
	}
	stat, err := os.Stat(fname)
	if err != nil {
		return wrapError(vm, err), nil
	}
	alloc := vm.Allocator()
	fstat := alloc.NewRecord(map[string]core.Object{
		"name":  alloc.NewString(stat.Name()),
		"mtime": alloc.NewTime(stat.ModTime()),
		"size":  alloc.NewInt(stat.Size()),
		"mode":  alloc.NewInt(int64(stat.Mode())),
	}, true).(*value.Record)
	fstat.SetKey("directory", alloc.NewBool(stat.IsDir()))
	return fstat, nil
}

func osCreate(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.create", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.create", "first", "string(compatible)", args[0])
	}
	res, err := os.Create(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	return makeOSFile(vm, res), nil
}

func osOpen(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.open", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.open", "first", "string(compatible)", args[0])
	}
	res, err := os.Open(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	return makeOSFile(vm, res), nil
}

func osOpenFile(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, core.NewWrongNumArgumentsError("os.open_file", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.open_file", "first", "string(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.open_file", "second", "int(compatible)", args[1])
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.open_file", "third", "int(compatible)", args[2])
	}
	res, err := os.OpenFile(s1, int(i2), os.FileMode(i3))
	if err != nil {
		return wrapError(vm, err), nil
	}
	return makeOSFile(vm, res), nil
}

func osArgs(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("os.args", "0", len(args))
	}
	arr := make([]core.Object, 0, len(os.Args))
	alloc := vm.Allocator()
	for _, osArg := range os.Args {
		if len(osArg) > core.MaxStringLen {
			return nil, core.NewStringLimitError("os.args")
		}
		arr = append(arr, alloc.NewString(osArg))
	}
	return alloc.NewArray(arr, false), nil
}

func osLookupEnv(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.lookup_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.lookup_env", "first", "string(compatible)", args[0])
	}
	res, ok := os.LookupEnv(s1)
	if !ok {
		return vm.Allocator().NewBool(false), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.lookup_env")
	}
	return vm.Allocator().NewString(res), nil
}

func osExpandEnv(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.expand_env", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.expand_env", "first", "string(compatible)", args[0])
	}
	var vlen int
	var failed bool
	s := os.Expand(s1, func(k string) string {
		if failed {
			return ""
		}
		v := os.Getenv(k)

		// this does not count the other texts that are not being replaced
		// but the code checks the final length at the end
		vlen += len(v)
		if vlen > core.MaxStringLen {
			failed = true
			return ""
		}
		return v
	})
	if failed || len(s) > core.MaxStringLen {
		return nil, core.NewStringLimitError("os.expand_env")
	}
	return vm.Allocator().NewString(s), nil
}

func osExec(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return nil, core.NewWrongNumArgumentsError("os.exec", "at least 1", len(args))
	}
	name, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.exec", "first", "string(compatible)", args[0])
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := arg.AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.exec", fmt.Sprintf("args[%d]", idx), "string(compatible)", arg)
		}
		execArgs = append(execArgs, execArg)
	}
	return makeOSExecCommand(vm, exec.Command(name, execArgs...)), nil
}

func osFindProcess(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("os.find_process", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.find_process", "first", "int(compatible)", args[0])
	}
	proc, err := os.FindProcess(int(i1))
	if err != nil {
		return wrapError(vm, err), nil
	}
	return makeOSProcess(vm, proc), nil
}

func osStartProcess(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 4 {
		return nil, core.NewWrongNumArgumentsError("os.start_process", "4", len(args))
	}
	name, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.start_process", "first", "string(compatible)", args[0])
	}
	var argv []string
	var err error
	switch arg1 := args[1].(type) {
	case *value.Array:
		argv, err = stringArray(arg1.Value(), "second")
		if err != nil {
			return nil, err
		}
	default:
		return nil, core.NewInvalidArgumentTypeError("os.start_process", "second", "array(string)", args[1])
	}

	dir, ok := args[2].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("os.start_process", "third", "string(compatible)", args[2])
	}

	var env []string
	switch arg3 := args[3].(type) {
	case *value.Array:
		env, err = stringArray(arg3.Value(), "fourth")
		if err != nil {
			return nil, err
		}
	default:
		return nil, core.NewInvalidArgumentTypeError("os.start_process", "fourth", "array(string)", args[3])
	}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Dir: dir,
		Env: env,
	})
	if err != nil {
		return wrapError(vm, err), nil
	}
	return makeOSProcess(vm, proc), nil
}

func stringArray(arr []core.Object, argName string) ([]string, error) {
	var ss []string
	for idx, elem := range arr {
		str, ok := elem.(*value.String)
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.start_process", fmt.Sprintf("%s[%d]", argName, idx), "string(compatible)", elem)
		}
		ss = append(ss, str.Value())
	}
	return ss, nil
}

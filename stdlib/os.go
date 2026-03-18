package stdlib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var osModule = map[string]core.Object{
	"platform":            value.NewString(runtime.GOOS),
	"arch":                value.NewString(runtime.GOARCH),
	"o_rdonly":            value.NewInt(int64(os.O_RDONLY)),
	"o_wronly":            value.NewInt(int64(os.O_WRONLY)),
	"o_rdwr":              value.NewInt(int64(os.O_RDWR)),
	"o_append":            value.NewInt(int64(os.O_APPEND)),
	"o_create":            value.NewInt(int64(os.O_CREATE)),
	"o_excl":              value.NewInt(int64(os.O_EXCL)),
	"o_sync":              value.NewInt(int64(os.O_SYNC)),
	"o_trunc":             value.NewInt(int64(os.O_TRUNC)),
	"mode_dir":            value.NewInt(int64(os.ModeDir)),
	"mode_append":         value.NewInt(int64(os.ModeAppend)),
	"mode_exclusive":      value.NewInt(int64(os.ModeExclusive)),
	"mode_temporary":      value.NewInt(int64(os.ModeTemporary)),
	"mode_symlink":        value.NewInt(int64(os.ModeSymlink)),
	"mode_device":         value.NewInt(int64(os.ModeDevice)),
	"mode_named_pipe":     value.NewInt(int64(os.ModeNamedPipe)),
	"mode_socket":         value.NewInt(int64(os.ModeSocket)),
	"mode_setuid":         value.NewInt(int64(os.ModeSetuid)),
	"mode_setgui":         value.NewInt(int64(os.ModeSetgid)),
	"mode_char_device":    value.NewInt(int64(os.ModeCharDevice)),
	"mode_sticky":         value.NewInt(int64(os.ModeSticky)),
	"mode_type":           value.NewInt(int64(os.ModeType)),
	"mode_perm":           value.NewInt(int64(os.ModePerm)),
	"path_separator":      value.NewChar(os.PathSeparator),
	"path_list_separator": value.NewChar(os.PathListSeparator),
	"dev_null":            value.NewString(os.DevNull),
	"seek_set":            value.NewInt(int64(io.SeekStart)),
	"seek_cur":            value.NewInt(int64(io.SeekCurrent)),
	"seek_end":            value.NewInt(int64(io.SeekEnd)),

	"args":           value.NewBuiltinFunction("args", osArgs, 0, false),                  // args() => array(string)
	"chdir":          value.NewBuiltinFunction("chdir", osChdir, 1, false),                // chdir(dir string) => error
	"chmod":          value.NewBuiltinFunction("chmod", osChmod, 2, false),                // chmod(name string, mode int) => error
	"chown":          value.NewBuiltinFunction("chown", osChown, 3, false),                // chown(name string, uid int, gid int) => error
	"clearenv":       value.NewBuiltinFunction("clearenv", osClearenv, 0, false),          // clearenv()
	"environ":        value.NewBuiltinFunction("environ", osEnviron, 0, false),            // environ() => array(string)
	"exit":           value.NewBuiltinFunction("exit", osExit, 1, false),                  // exit(code int)
	"expand_env":     value.NewBuiltinFunction("expand_env", osExpandEnv, 1, false),       // expand_env(s string) => string
	"getegid":        value.NewBuiltinFunction("getegid", osGetegid, 0, false),            // getegid() => int
	"getenv":         value.NewBuiltinFunction("getenv", osGetenv, 1, false),              // getenv(s string) => string
	"geteuid":        value.NewBuiltinFunction("geteuid", osGeteuid, 0, false),            // geteuid() => int
	"getgid":         value.NewBuiltinFunction("getgid", osGetgid, 0, false),              // getgid() => int
	"getgroups":      value.NewBuiltinFunction("getgroups", osGetgroups, 0, false),        // getgroups() => array(string)/error
	"getpagesize":    value.NewBuiltinFunction("getpagesize", osGetpagesize, 0, false),    // getpagesize() => int
	"getpid":         value.NewBuiltinFunction("getpid", osGetpid, 0, false),              // getpid() => int
	"getppid":        value.NewBuiltinFunction("getppid", osGetppid, 0, false),            // getppid() => int
	"getuid":         value.NewBuiltinFunction("getuid", osGetuid, 0, false),              // getuid() => int
	"getwd":          value.NewBuiltinFunction("getwd", osGetwd, 0, false),                // getwd() => string/error
	"hostname":       value.NewBuiltinFunction("hostname", osHostname, 0, false),          // hostname() => string/error
	"lchown":         value.NewBuiltinFunction("lchown", osLchown, 3, false),              // lchown(name string, uid int, gid int) => error
	"link":           value.NewBuiltinFunction("link", osLink, 2, false),                  // link(oldname string, newname string) => error
	"lookup_env":     value.NewBuiltinFunction("lookup_env", osLookupEnv, 1, false),       // lookup_env(key string) => string/false
	"mkdir":          value.NewBuiltinFunction("mkdir", osMkdir, 2, false),                // mkdir(name string, perm int) => error
	"mkdir_all":      value.NewBuiltinFunction("mkdir_all", osMkdirAll, 2, false),         // mkdir_all(name string, perm int) => error
	"readlink":       value.NewBuiltinFunction("readlink", osReadlink, 1, false),          // readlink(name string) => string/error
	"remove":         value.NewBuiltinFunction("remove", osRemove, 1, false),              // remove(name string) => error
	"remove_all":     value.NewBuiltinFunction("remove_all", osRemoveAll, 1, false),       // remove_all(name string) => error
	"rename":         value.NewBuiltinFunction("rename", osRename, 2, false),              // rename(oldpath string, newpath string) => error
	"setenv":         value.NewBuiltinFunction("setenv", osSetenv, 2, false),              // setenv(key string, value string) => error
	"symlink":        value.NewBuiltinFunction("symlink", osSymlink, 2, false),            // symlink(oldname string newname string) => error
	"temp_dir":       value.NewBuiltinFunction("temp_dir", osTempDir, 0, false),           // temp_dir() => string
	"truncate":       value.NewBuiltinFunction("truncate", osTruncate, 2, false),          // truncate(name string, size int) => error
	"unsetenv":       value.NewBuiltinFunction("unsetenv", osUnsetenv, 1, false),          // unsetenv(key string) => error
	"create":         value.NewBuiltinFunction("create", osCreate, 1, false),              // create(name string) => imap(file)/error
	"open":           value.NewBuiltinFunction("open", osOpen, 1, false),                  // open(name string) => imap(file)/error
	"open_file":      value.NewBuiltinFunction("open_file", osOpenFile, 3, false),         // open_file(name string, flag int, perm int) => imap(file)/error
	"find_process":   value.NewBuiltinFunction("find_process", osFindProcess, 1, false),   // find_process(pid int) => imap(process)/error
	"start_process":  value.NewBuiltinFunction("start_process", osStartProcess, 4, false), // start_process(name string, argv array(string), dir string, env array(string)) => imap(process)/error
	"exec_look_path": value.NewBuiltinFunction("exec_look_path", execLookPath, 1, false),  // exec_look_path(file) => string/error
	"exec":           value.NewBuiltinFunction("exec", osExec, 1, true),                   // exec(name, args...) => command
	"stat":           value.NewBuiltinFunction("stat", osStat, 1, false),                  // stat(name) => imap(fileinfo)/error
	"read_file":      value.NewBuiltinFunction("read_file", osReadFile, 1, false),         // readfile(name) => array(byte)/error
}

func osChmod(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Chmod(s1, os.FileMode(i2))), nil
}

func osMkdir(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Mkdir(s1, os.FileMode(i2))), nil
}

func osMkdirAll(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.MkdirAll(s1, os.FileMode(i2))), nil
}

func osLchown(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	return wrapError(os.Lchown(s1, int(i2), int(i3))), nil
}

func osChown(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	return wrapError(os.Chown(s1, int(i2), int(i3))), nil
}

func osTruncate(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Truncate(s1, i2)), nil
}

func osSymlink(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Symlink(s1, s2)), nil
}

func osSetenv(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Setenv(s1, s2)), nil
}

func osRename(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Rename(s1, s2)), nil
}

func osLink(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return wrapError(os.Link(s1, s2)), nil
}

func osUnsetenv(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return wrapError(os.Unsetenv(s1)), nil
}

func osRemoveAll(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return wrapError(os.RemoveAll(s1)), nil
}

func osRemove(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return wrapError(os.Remove(s1)), nil
}

func osChdir(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return wrapError(os.Chdir(s1)), nil
}

func execLookPath(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := exec.LookPath(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("os.exec_look_path")
	}
	return value.NewString(res), nil
}

func osReadlink(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := os.Readlink(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("os.readlink")
	}
	return value.NewString(res), nil
}

func osGetenv(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := os.Getenv(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("os.getenv")
	}
	return value.NewString(s), nil
}

func osExit(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	os.Exit(int(i1))
	return value.UndefinedValue, nil
}

func osGetgroups(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	res, err := os.Getgroups()
	if err != nil {
		return wrapError(err), nil
	}
	arr := make([]core.Object, 0, len(res))
	for _, v := range res {
		arr = append(arr, value.NewInt(int64(v)))
	}
	return value.NewArray(arr, false), nil
}

func osEnviron(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	env := os.Environ()
	arr := make([]core.Object, 0, len(env))
	for _, elem := range env {
		if len(elem) > core.MaxStringLen {
			return nil, core.StringLimit("os.environ")
		}
		arr = append(arr, value.NewString(elem))
	}
	return value.NewArray(arr, false), nil
}

func osHostname(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	res, err := os.Hostname()
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("os.hostname")
	}
	return value.NewString(res), nil
}

func osGetwd(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	res, err := os.Getwd()
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("os.getwd")
	}
	return value.NewString(res), nil
}

func osTempDir(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	s := os.TempDir()
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("os.temp_dir")
	}
	return value.NewString(s), nil
}

func osGetuid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getuid())), nil
}

func osGetppid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getppid())), nil
}

func osGetpid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getpid())), nil
}

func osGetpagesize(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getpagesize())), nil
}

func osGetgid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getgid())), nil
}

func osGeteuid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Geteuid())), nil
}

func osGetegid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewInt(int64(os.Getegid())), nil
}

func osClearenv(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	os.Clearenv()
	return value.UndefinedValue, nil
}

func osReadFile(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	fname, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return wrapError(err), nil
	}
	if len(bytes) > core.MaxBytesLen {
		return nil, core.BytesLimit("os.read_file")
	}
	return value.NewBytes(bytes), nil
}

func osStat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	fname, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	stat, err := os.Stat(fname)
	if err != nil {
		return wrapError(err), nil
	}
	fstat := value.NewMap(map[string]core.Object{
		"name":  value.NewString(stat.Name()),
		"mtime": value.NewTime(stat.ModTime()),
		"size":  value.NewInt(stat.Size()),
		"mode":  value.NewInt(int64(stat.Mode())),
	}, true)
	if stat.IsDir() {
		fstat.SetKey("directory", value.TrueValue)
	} else {
		fstat.SetKey("directory", value.FalseValue)
	}
	return fstat, nil
}

func osCreate(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := os.Create(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpen(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := os.Open(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpenFile(args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	res, err := os.OpenFile(s1, int(i2), os.FileMode(i3))
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osArgs(args ...core.Object) (core.Object, error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	arr := make([]core.Object, 0, len(os.Args))
	for _, osArg := range os.Args {
		if len(osArg) > core.MaxStringLen {
			return nil, core.StringLimit("os.args")
		}
		arr = append(arr, value.NewString(osArg))
	}
	return value.NewArray(arr, false), nil
}

func osLookupEnv(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, ok := os.LookupEnv(s1)
	if !ok {
		return value.FalseValue, nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("os.lookup_env")
	}
	return value.NewString(res), nil
}

func osExpandEnv(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
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
		return nil, core.StringLimit("os.expand_env")
	}
	return value.NewString(s), nil
}

func osExec(args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return nil, gse.ErrWrongNumArguments
	}
	name, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := arg.AsString()
		if !ok {
			return nil, &gse.InvalidArgumentTypeError{
				Name:     fmt.Sprintf("args[%d]", idx),
				Expected: "string(compatible)",
				Found:    args[1+idx].TypeName(),
			}
		}
		execArgs = append(execArgs, execArg)
	}
	return makeOSExecCommand(exec.Command(name, execArgs...)), nil
}

func osFindProcess(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	proc, err := os.FindProcess(int(i1))
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSProcess(proc), nil
}

func osStartProcess(args ...core.Object) (core.Object, error) {
	if len(args) != 4 {
		return nil, gse.ErrWrongNumArguments
	}
	name, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
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
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "second",
			Expected: "array",
			Found:    arg1.TypeName(),
		}
	}

	dir, ok := args[2].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
	}

	var env []string
	switch arg3 := args[3].(type) {
	case *value.Array:
		env, err = stringArray(arg3.Value(), "fourth")
		if err != nil {
			return nil, err
		}
	default:
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "fourth",
			Expected: "array",
			Found:    arg3.TypeName(),
		}
	}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Dir: dir,
		Env: env,
	})
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSProcess(proc), nil
}

func stringArray(arr []core.Object, argName string) ([]string, error) {
	var ss []string
	for idx, elem := range arr {
		str, ok := elem.(*value.String)
		if !ok {
			return nil, &gse.InvalidArgumentTypeError{Name: fmt.Sprintf("%s[%d]", argName, idx), Expected: "string", Found: elem.TypeName()}
		}
		ss = append(ss, str.Value())
	}
	return ss, nil
}

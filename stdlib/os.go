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
	"platform":            &value.String{Value: runtime.GOOS},
	"arch":                &value.String{Value: runtime.GOARCH},
	"o_rdonly":            &value.Int{Value: int64(os.O_RDONLY)},
	"o_wronly":            &value.Int{Value: int64(os.O_WRONLY)},
	"o_rdwr":              &value.Int{Value: int64(os.O_RDWR)},
	"o_append":            &value.Int{Value: int64(os.O_APPEND)},
	"o_create":            &value.Int{Value: int64(os.O_CREATE)},
	"o_excl":              &value.Int{Value: int64(os.O_EXCL)},
	"o_sync":              &value.Int{Value: int64(os.O_SYNC)},
	"o_trunc":             &value.Int{Value: int64(os.O_TRUNC)},
	"mode_dir":            &value.Int{Value: int64(os.ModeDir)},
	"mode_append":         &value.Int{Value: int64(os.ModeAppend)},
	"mode_exclusive":      &value.Int{Value: int64(os.ModeExclusive)},
	"mode_temporary":      &value.Int{Value: int64(os.ModeTemporary)},
	"mode_symlink":        &value.Int{Value: int64(os.ModeSymlink)},
	"mode_device":         &value.Int{Value: int64(os.ModeDevice)},
	"mode_named_pipe":     &value.Int{Value: int64(os.ModeNamedPipe)},
	"mode_socket":         &value.Int{Value: int64(os.ModeSocket)},
	"mode_setuid":         &value.Int{Value: int64(os.ModeSetuid)},
	"mode_setgui":         &value.Int{Value: int64(os.ModeSetgid)},
	"mode_char_device":    &value.Int{Value: int64(os.ModeCharDevice)},
	"mode_sticky":         &value.Int{Value: int64(os.ModeSticky)},
	"mode_type":           &value.Int{Value: int64(os.ModeType)},
	"mode_perm":           &value.Int{Value: int64(os.ModePerm)},
	"path_separator":      &value.Char{Value: os.PathSeparator},
	"path_list_separator": &value.Char{Value: os.PathListSeparator},
	"dev_null":            &value.String{Value: os.DevNull},
	"seek_set":            &value.Int{Value: int64(io.SeekStart)},
	"seek_cur":            &value.Int{Value: int64(io.SeekCurrent)},
	"seek_end":            &value.Int{Value: int64(io.SeekEnd)},
	"args": &value.BuiltinFunction{
		Name:  "args",
		Value: osArgs,
	}, // args() => array(string)
	"chdir": &value.BuiltinFunction{
		Name:  "chdir",
		Value: osChdir,
	}, // chdir(dir string) => error
	"chmod": osFuncASFmRE("chmod", os.Chmod), // chmod(name string, mode int) => error
	"chown": &value.BuiltinFunction{
		Name:  "chown",
		Value: FuncASIIRE(os.Chown),
	}, // chown(name string, uid int, gid int) => error
	"clearenv": &value.BuiltinFunction{
		Name:  "clearenv",
		Value: osClearenv,
	}, // clearenv()
	"environ": &value.BuiltinFunction{
		Name:  "environ",
		Value: osEnviron,
	}, // environ() => array(string)
	"exit": &value.BuiltinFunction{
		Name:  "exit",
		Value: osExit,
	}, // exit(code int)
	"expand_env": &value.BuiltinFunction{
		Name:  "expand_env",
		Value: osExpandEnv,
	}, // expand_env(s string) => string
	"getegid": &value.BuiltinFunction{
		Name:  "getegid",
		Value: osGetegid,
	}, // getegid() => int
	"getenv": &value.BuiltinFunction{
		Name:  "getenv",
		Value: osGetenv,
	}, // getenv(s string) => string
	"geteuid": &value.BuiltinFunction{
		Name:  "geteuid",
		Value: osGeteuid,
	}, // geteuid() => int
	"getgid": &value.BuiltinFunction{
		Name:  "getgid",
		Value: osGetgid,
	}, // getgid() => int
	"getgroups": &value.BuiltinFunction{
		Name:  "getgroups",
		Value: osGetgroups,
	}, // getgroups() => array(string)/error
	"getpagesize": &value.BuiltinFunction{
		Name:  "getpagesize",
		Value: osGetpagesize,
	}, // getpagesize() => int
	"getpid": &value.BuiltinFunction{
		Name:  "getpid",
		Value: osGetpid,
	}, // getpid() => int
	"getppid": &value.BuiltinFunction{
		Name:  "getppid",
		Value: osGetppid,
	}, // getppid() => int
	"getuid": &value.BuiltinFunction{
		Name:  "getuid",
		Value: osGetuid,
	}, // getuid() => int
	"getwd": &value.BuiltinFunction{
		Name:  "getwd",
		Value: osGetwd,
	}, // getwd() => string/error
	"hostname": &value.BuiltinFunction{
		Name:  "hostname",
		Value: osHostname,
	}, // hostname() => string/error
	"lchown": &value.BuiltinFunction{
		Name:  "lchown",
		Value: FuncASIIRE(os.Lchown),
	}, // lchown(name string, uid int, gid int) => error
	"link": &value.BuiltinFunction{
		Name:  "link",
		Value: osLink,
	}, // link(oldname string, newname string) => error
	"lookup_env": &value.BuiltinFunction{
		Name:  "lookup_env",
		Value: osLookupEnv,
	}, // lookup_env(key string) => string/false
	"mkdir":     osFuncASFmRE("mkdir", os.Mkdir),        // mkdir(name string, perm int) => error
	"mkdir_all": osFuncASFmRE("mkdir_all", os.MkdirAll), // mkdir_all(name string, perm int) => error
	"readlink": &value.BuiltinFunction{
		Name:  "readlink",
		Value: osReadlink,
	}, // readlink(name string) => string/error
	"remove": &value.BuiltinFunction{
		Name:  "remove",
		Value: osRemove,
	}, // remove(name string) => error
	"remove_all": &value.BuiltinFunction{
		Name:  "remove_all",
		Value: osRemoveAll,
	}, // remove_all(name string) => error
	"rename": &value.BuiltinFunction{
		Name:  "rename",
		Value: osRename,
	}, // rename(oldpath string, newpath string) => error
	"setenv": &value.BuiltinFunction{
		Name:  "setenv",
		Value: osSetenv,
	}, // setenv(key string, value string) => error
	"symlink": &value.BuiltinFunction{
		Name:  "symlink",
		Value: osSymlink,
	}, // symlink(oldname string newname string) => error
	"temp_dir": &value.BuiltinFunction{
		Name:  "temp_dir",
		Value: osTempDir,
	}, // temp_dir() => string
	"truncate": &value.BuiltinFunction{
		Name:  "truncate",
		Value: FuncASI64RE(os.Truncate),
	}, // truncate(name string, size int) => error
	"unsetenv": &value.BuiltinFunction{
		Name:  "unsetenv",
		Value: osUnsetenv,
	}, // unsetenv(key string) => error
	"create": &value.BuiltinFunction{
		Name:  "create",
		Value: osCreate,
	}, // create(name string) => imap(file)/error
	"open": &value.BuiltinFunction{
		Name:  "open",
		Value: osOpen,
	}, // open(name string) => imap(file)/error
	"open_file": &value.BuiltinFunction{
		Name:  "open_file",
		Value: osOpenFile,
	}, // open_file(name string, flag int, perm int) => imap(file)/error
	"find_process": &value.BuiltinFunction{
		Name:  "find_process",
		Value: osFindProcess,
	}, // find_process(pid int) => imap(process)/error
	"start_process": &value.BuiltinFunction{
		Name:  "start_process",
		Value: osStartProcess,
	}, // start_process(name string, argv array(string), dir string, env array(string)) => imap(process)/error
	"exec_look_path": &value.BuiltinFunction{
		Name:  "exec_look_path",
		Value: execLookPath,
	}, // exec_look_path(file) => string/error
	"exec": &value.BuiltinFunction{
		Name:  "exec",
		Value: osExec,
	}, // exec(name, args...) => command
	"stat": &value.BuiltinFunction{
		Name:  "stat",
		Value: osStat,
	}, // stat(name) => imap(fileinfo)/error
	"read_file": &value.BuiltinFunction{
		Name:  "read_file",
		Value: osReadFile,
	}, // readfile(name) => array(byte)/error
}

func osSymlink(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
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
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: res}, nil
}

func osReadlink(args ...core.Object) (core.Object, error) {
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
	res, err := os.Readlink(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: res}, nil
}

func osGetenv(args ...core.Object) (core.Object, error) {
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
	s := os.Getenv(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: s}, nil
}

func osExit(args ...core.Object) (ret core.Object, err error) {
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
	arr := &value.Array{}
	for _, v := range res {
		arr.Value = append(arr.Value, &value.Int{Value: int64(v)})
	}
	return arr, nil
}

func osEnviron(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	arr := &value.Array{}
	for _, elem := range os.Environ() {
		if len(elem) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, &value.String{Value: elem})
	}
	return arr, nil
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
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: res}, nil
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
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: res}, nil
}

func osTempDir(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	s := os.TempDir()
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: s}, nil
}

func osGetuid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getuid())}, nil
}

func osGetppid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getppid())}, nil
}

func osGetpid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getpid())}, nil
}

func osGetpagesize(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getpagesize())}, nil
}

func osGetgid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getgid())}, nil
}

func osGeteuid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Geteuid())}, nil
}

func osGetegid(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: int64(os.Getegid())}, nil
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrBytesLimit
	}
	return &value.Bytes{Value: bytes}, nil
}

func osStat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	fname, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	stat, err := os.Stat(fname)
	if err != nil {
		return wrapError(err), nil
	}
	fstat := &value.ImmutableMap{
		Value: map[string]core.Object{
			"name":  &value.String{Value: stat.Name()},
			"mtime": &value.Time{Value: stat.ModTime()},
			"size":  &value.Int{Value: stat.Size()},
			"mode":  &value.Int{Value: int64(stat.Mode())},
		},
	}
	if stat.IsDir() {
		fstat.Value["directory"] = value.TrueValue
	} else {
		fstat.Value["directory"] = value.FalseValue
	}
	return fstat, nil
}

func osCreate(args ...core.Object) (core.Object, error) {
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
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
	arr := &value.Array{}
	for _, osArg := range os.Args {
		if len(osArg) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, &value.String{Value: osArg})
	}
	return arr, nil
}

func osFuncASFmRE(
	name string,
	fn func(string, os.FileMode) error,
) *value.BuiltinFunction {
	return &value.BuiltinFunction{
		Name: name,
		Value: func(args ...core.Object) (core.Object, error) {
			if len(args) != 2 {
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
			i2, ok := args[1].AsInt()
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     "second",
					Expected: "int(compatible)",
					Found:    args[1].TypeName(),
				}
			}
			return wrapError(fn(s1, os.FileMode(i2))), nil
		},
	}
}

func osLookupEnv(args ...core.Object) (core.Object, error) {
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
	res, ok := os.LookupEnv(s1)
	if !ok {
		return value.FalseValue, nil
	}
	if len(res) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: res}, nil
}

func osExpandEnv(args ...core.Object) (core.Object, error) {
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
		return nil, gse.ErrStringLimit
	}
	return &value.String{Value: s}, nil
}

func osExec(args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return nil, gse.ErrWrongNumArguments
	}
	name, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := arg.AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
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
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	var argv []string
	var err error
	switch arg1 := args[1].(type) {
	case *value.Array:
		argv, err = stringArray(arg1.Value, "second")
		if err != nil {
			return nil, err
		}
	case *value.ImmutableArray:
		argv, err = stringArray(arg1.Value, "second")
		if err != nil {
			return nil, err
		}
	default:
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "array",
			Found:    arg1.TypeName(),
		}
	}

	dir, ok := args[2].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
	}

	var env []string
	switch arg3 := args[3].(type) {
	case *value.Array:
		env, err = stringArray(arg3.Value, "fourth")
		if err != nil {
			return nil, err
		}
	case *value.ImmutableArray:
		env, err = stringArray(arg3.Value, "fourth")
		if err != nil {
			return nil, err
		}
	default:
		return nil, gse.ErrInvalidArgumentType{
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
	var sarr []string
	for idx, elem := range arr {
		str, ok := elem.(*value.String)
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     fmt.Sprintf("%s[%d]", argName, idx),
				Expected: "string",
				Found:    elem.TypeName(),
			}
		}
		sarr = append(sarr, str.Value)
	}
	return sarr, nil
}

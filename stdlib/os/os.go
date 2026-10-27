package os

import (
	"os"
	"runtime"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
	"github.com/jokruger/kavun/stdlib/rand"
)

// The os module is the process and its environment: arguments, exit, environment variables, identity, the
// working directory. Files are the fs module's, processes the exec module's, path text the path module's.

// The process and its environment: arguments, exit, environment variables, identity, the working directory.
//
// Files are [`fs`](fs.md)'s, other processes [`exec`](exec.md)'s, path text [`path`](path.md)'s.
//
// ```go
// os := import("os")
// os.get_env("HOME")
// os.platform              // "linux", "darwin", …
// ```
//
// Constants: `platform`, `arch`, `dev_null`, `path_separator`, `path_list_separator`.
//
// **Failures.** Everything the world can refuse raises kind `io`, with the operation named in the message:
// `(os.chdir) chdir /nope: no such file or directory`. A function whose success has no result answers `undefined`.
//
//kavun:module os impure

// moduleExtras declares what the //kavun:fn directives do not: the module's constants and owned types.
func moduleExtras(m *core.ModuleBuilder) {
	m.Const(members.Platform, core.NewStringValue(runtime.GOOS))
	m.Const(members.Arch, core.NewStringValue(runtime.GOARCH))
	m.Const(members.PathSeparator, core.RuneValue(os.PathSeparator))
	m.Const(members.PathListSeparator, core.RuneValue(os.PathListSeparator))
	m.Const(members.DevNull, core.NewStringValue(os.DevNull))
}

// Answer the command-line arguments.
//
//kavun:fn args slot=0 impure () -> [string]
func osArgs(vm core.VM, args []core.Value) (core.Value, error) {
	arr := make([]core.Value, 0, len(os.Args))
	for _, osArg := range os.Args {
		arr = append(arr, core.NewStringValue(osArg))
	}
	return core.NewArrayValue(arr, false), nil
}

// Exit the process with `code`.
//
// `code` is an `int`.
//
//kavun:fn exit slot=1 impure (code) -> undefined
func osExit(vm core.VM, args []core.Value) (ret core.Value, err error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.exit", "first", "int(compatible)", args[0].TypeName())
	}
	os.Exit(int(i1))
	return core.Undefined, nil
}

// Answer the environment as `KEY=VALUE` strings.
//
//kavun:fn environ slot=2 impure () -> [string]
func osEnviron(vm core.VM, args []core.Value) (ret core.Value, err error) {
	env := os.Environ()
	arr := make([]core.Value, 0, len(env))
	for _, elem := range env {
		arr = append(arr, core.NewStringValue(elem))
	}
	return core.NewArrayValue(arr, false), nil
}

// Answer an environment variable's value, `""` when it is missing.
//
// `key` is a `string`.
//
//kavun:fn get_env slot=3 impure (key) -> string
func osGetenv(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.get_env", "first", "string(compatible)", args[0].TypeName())
	}
	s := os.Getenv(s1)
	return core.NewStringValue(s), nil
}

// Set an environment variable.
//
// `key` and `value` are `string`s.
//
//kavun:fn set_env slot=4 impure (key, value) -> undefined
func osSetenv(vm core.VM, args []core.Value) (core.Value, error) {
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

// Unset an environment variable.
//
// `key` is a `string`.
//
//kavun:fn unset_env slot=5 impure (key) -> undefined
func osUnsetenv(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.unset_env", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("os.unset_env", os.Unsetenv(s1))
}

// Answer an environment variable's value, or `false` when it is missing.
//
// `key` is a `string`.
//
//kavun:fn lookup_env slot=6 impure (key) -> string | false
func osLookupEnv(vm core.VM, args []core.Value) (core.Value, error) {
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

// Expand the `$VAR` references in `s`.
//
// `s` is a `string`.
//
//kavun:fn expand_env slot=7 impure (s) -> string
func osExpandEnv(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.expand_env", "first", "string(compatible)", args[0].TypeName())
	}
	s := os.Expand(s1, func(k string) string {
		return os.Getenv(k)
	})
	return core.NewStringValue(s), nil
}

// Clear every environment variable.
//
//kavun:fn clear_env slot=8 impure () -> undefined
func osClearenv(vm core.VM, args []core.Value) (ret core.Value, err error) {
	os.Clearenv()
	return core.Undefined, nil
}

// Answer the host name.
//
//kavun:fn hostname slot=9 impure () -> string
func osHostname(vm core.VM, args []core.Value) (ret core.Value, err error) {
	res, err := os.Hostname()
	if err != nil {
		return modkit.RaiseIO("os.hostname", err)
	}
	return core.NewStringValue(res), nil
}

// Answer the current working directory.
//
//kavun:fn get_wd slot=10 impure () -> string
func osGetwd(vm core.VM, args []core.Value) (ret core.Value, err error) {
	res, err := os.Getwd()
	if err != nil {
		return modkit.RaiseIO("os.get_wd", err)
	}
	return core.NewStringValue(res), nil
}

// Change the current working directory.
//
// `dir` is a `string`.
//
//kavun:fn chdir slot=11 impure (dir) -> undefined
func osChdir(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("os.chdir", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("os.chdir", os.Chdir(s1))
}

// Answer the current process id.
//
//kavun:fn get_pid slot=12 impure () -> int
func osGetpid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getpid())), nil
}

// Answer the parent process id; see [`get_pid`](#get_pid).
//
//kavun:fn get_ppid slot=13 impure () -> int
func osGetppid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getppid())), nil
}

// Answer the real user id.
//
//kavun:fn get_uid slot=14 impure () -> int
func osGetuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getuid())), nil
}

// Answer the effective user id; see [`get_uid`](#get_uid).
//
//kavun:fn get_euid slot=15 impure () -> int
func osGeteuid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Geteuid())), nil
}

// Answer the real group id; see [`get_uid`](#get_uid).
//
//kavun:fn get_gid slot=16 impure () -> int
func osGetgid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getgid())), nil
}

// Answer the effective group id; see [`get_uid`](#get_uid).
//
//kavun:fn get_egid slot=17 impure () -> int
func osGetegid(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getegid())), nil
}

// Answer the supplementary group ids.
//
//kavun:fn get_groups slot=18 impure () -> [int]
func osGetgroups(vm core.VM, args []core.Value) (ret core.Value, err error) {
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

// Answer the memory page size.
//
//kavun:fn get_page_size slot=19 impure () -> int
func osGetpagesize(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.IntValue(int64(os.Getpagesize())), nil
}

// Answer a [`rand.generator`](../types/rand_generator.md) over `crypto/rand` — the one entropy source.
//
// `os.rand_generator().int(1, 7)` rolls an unpredictable die.
//
// It is a world read, so it lives here and not in the pure `rand` module; a reproducible run seeds `rand.generator(…)`
// instead.
//
//kavun:fn rand_generator slot=20 impure () -> rand.generator
func osRandGenerator(_ core.VM, args []core.Value) (core.Value, error) {
	return rand.NewEntropyValue(), nil
}

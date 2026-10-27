package fs

import (
	"io"
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The fs module is the file system: files (the fs.file type), directories, links, permissions.

// The file system: files, directories, links, permissions.
//
// ```go
// fs := import("fs")
// f := fs.create("./out.txt")
// f.write_string("hello")
// f.close()
// string(fs.read_file("./out.txt"))   // => "hello" — read_file answers bytes
// fs.stat("./out.txt").size           // => 5
// ```
//
// Constants:
//
// - Open flags: `o_rd`, `o_wr`, `o_rdwr`, `o_append`, `o_create`, `o_excl`, `o_sync`, `o_trunc`.
// - Seek modes: `seek_set`, `seek_cur`, `seek_end`.
// - File mode bits: `mode_dir`, `mode_append`, `mode_exclusive`, `mode_temporary`, `mode_symlink`, `mode_device`,
// `mode_named_pipe`, `mode_socket`, `mode_set_uid`, `mode_set_gui`, `mode_char_device`, `mode_sticky`,
// `mode_type`, `mode_perm`.
//
// An open file is an [`fs.file`](../types/fs_file.md): `name`, `read(buffer)`, `write(bytes)`, `write_string(s)`,
// `seek(offset, whence)`, `stat`, `sync`, `chmod`, `chown`, `chdir`, `read_dir_names(n)`, `close`.
//
// **Failures.** Everything the world can refuse raises kind `io`, with the operation named in the message:
// `(fs.remove) remove /nope: no such file or directory`. A function whose success has no result answers `undefined`.
//
//kavun:module fs impure

// moduleExtras declares what the //kavun:fn directives do not: the module's constants and owned types.
func moduleExtras(m *core.ModuleBuilder) {
	m.Type(value.FsFile, TypeFile)
	m.Const(members.ORd, core.IntValue(int64(os.O_RDONLY)))
	m.Const(members.OWr, core.IntValue(int64(os.O_WRONLY)))
	m.Const(members.ORdwr, core.IntValue(int64(os.O_RDWR)))
	m.Const(members.OAppend, core.IntValue(int64(os.O_APPEND)))
	m.Const(members.OCreate, core.IntValue(int64(os.O_CREATE)))
	m.Const(members.OExcl, core.IntValue(int64(os.O_EXCL)))
	m.Const(members.OSync, core.IntValue(int64(os.O_SYNC)))
	m.Const(members.OTrunc, core.IntValue(int64(os.O_TRUNC)))
	m.Const(members.SeekSet, core.IntValue(int64(io.SeekStart)))
	m.Const(members.SeekCur, core.IntValue(int64(io.SeekCurrent)))
	m.Const(members.SeekEnd, core.IntValue(int64(io.SeekEnd)))
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
}

// Open a file for reading.
//
// `path` is a `string`.
//
//kavun:fn open slot=0 impure (path) -> fs.file
func fsOpen(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.open", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Open(s1)
	if err != nil {
		return modkit.RaiseIO("fs.open", err)
	}
	return NewFileValue(res), nil
}

// Open a file with flags and permission bits.
//
// `path` is a `string` and `flag` and `perm` are `int`s.
//
//kavun:fn open_file slot=1 impure (path, flag, perm) -> fs.file
func fsOpenFile(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.open_file", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.open_file", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.open_file", "third", "int(compatible)", args[2].TypeName())
	}
	res, err := os.OpenFile(s1, int(i2), os.FileMode(i3))
	if err != nil {
		return modkit.RaiseIO("fs.open_file", err)
	}
	return NewFileValue(res), nil
}

// Create (or truncate) a file.
//
// `path` is a `string`.
//
//kavun:fn create slot=2 impure (path) -> fs.file
func fsCreate(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.create", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Create(s1)
	if err != nil {
		return modkit.RaiseIO("fs.create", err)
	}
	return NewFileValue(res), nil
}

// Read a whole file.
//
// `path` is a `string`.
//
//kavun:fn read_file slot=3 impure (path) -> bytes
func fsReadFile(vm core.VM, args []core.Value) (ret core.Value, err error) {
	fname, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.read_file", "first", "string(compatible)", args[0].TypeName())
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return modkit.RaiseIO("fs.read_file", err)
	}
	return core.NewBytesValue(bytes, false), nil
}

// Answer a frozen record `{name, mtime, size, mode, directory}` of facts about a file.
//
// `path` is a `string`.
//
//kavun:fn stat slot=4 impure (path) -> record
func fsStat(vm core.VM, args []core.Value) (ret core.Value, err error) {

	fname, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.stat", "first", "string(compatible)", args[0].TypeName())
	}

	stat, err := os.Stat(fname)
	if err != nil {
		return modkit.RaiseIO("fs.stat", err)
	}

	return NewStatValue(stat), nil
}

// Create a directory.
//
// `path` is a `string` and `perm` is an `int`.
//
//kavun:fn mkdir slot=5 impure (path, perm) -> undefined
func fsMkdir(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.mkdir", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.mkdir", "second", "int(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.mkdir", os.Mkdir(s1, os.FileMode(i2)))
}

// Create a directory and every missing parent; see [`mkdir`](#mkdir).
//
// `path` is a `string` and `perm` is an `int`.
//
//kavun:fn mkdir_all slot=6 impure (path, perm) -> undefined
func fsMkdirAll(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.mkdir_all", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.mkdir_all", "second", "int(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.mkdir_all", os.MkdirAll(s1, os.FileMode(i2)))
}

// Remove a file or an empty directory.
//
// `path` is a `string`.
//
//kavun:fn remove slot=7 impure (path) -> undefined
func fsRemove(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.remove", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("fs.remove", os.Remove(s1))
}

// Remove a file or a whole tree; see [`remove`](#remove).
//
// `path` is a `string`.
//
//kavun:fn remove_all slot=8 impure (path) -> undefined
func fsRemoveAll(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.remove_all", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("fs.remove_all", os.RemoveAll(s1))
}

// Rename or move a file.
//
// `old_path` and `new_path` are `string`s.
//
//kavun:fn rename slot=9 impure (old_path, new_path) -> undefined
func fsRename(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.rename", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.rename", "second", "string(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.rename", os.Rename(s1, s2))
}

// Truncate a file to `size` bytes.
//
// `path` is a `string` and `size` is an `int`.
//
//kavun:fn truncate slot=10 impure (path, size) -> undefined
func fsTruncate(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.truncate", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.truncate", "second", "int(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.truncate", os.Truncate(s1, i2))
}

// Set a file's permission bits.
//
// `path` is a `string` and `mode` is an `int`.
//
//kavun:fn chmod slot=11 impure (path, mode) -> undefined
func fsChmod(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.chmod", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.chmod", "second", "int(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.chmod", os.Chmod(s1, os.FileMode(i2)))
}

// Set a file's owner and group; see [`chmod`](#chmod).
//
// `path` is a `string` and `uid` and `gid` are `int`s.
//
//kavun:fn chown slot=12 impure (path, uid, gid) -> undefined
func fsChown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.chown", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.chown", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.chown", "third", "int(compatible)", args[2].TypeName())
	}
	return modkit.Done("fs.chown", os.Chown(s1, int(i2), int(i3)))
}

// Set a symbolic link's own owner and group; see [`chmod`](#chmod).
//
// `path` is a `string` and `uid` and `gid` are `int`s.
//
//kavun:fn lchown slot=13 impure (path, uid, gid) -> undefined
func fsLchown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.lchown", "first", "string(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.lchown", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.lchown", "third", "int(compatible)", args[2].TypeName())
	}
	return modkit.Done("fs.lchown", os.Lchown(s1, int(i2), int(i3)))
}

// Create a hard link.
//
// `old_path` and `new_path` are `string`s.
//
//kavun:fn link slot=14 impure (old_path, new_path) -> undefined
func fsLink(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.link", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.link", "second", "string(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.link", os.Link(s1, s2))
}

// Create a symbolic link; see [`link`](#link).
//
// `old_path` and `new_path` are `string`s.
//
//kavun:fn symlink slot=15 impure (old_path, new_path) -> undefined
func fsSymlink(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.symlink", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.symlink", "second", "string(compatible)", args[1].TypeName())
	}
	return modkit.Done("fs.symlink", os.Symlink(s1, s2))
}

// Answer the target of a symbolic link; see [`link`](#link).
//
// `path` is a `string`.
//
//kavun:fn read_link slot=16 impure (path) -> string
func fsReadlink(vm core.VM, args []core.Value) (core.Value, error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.read_link", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := os.Readlink(s1)
	if err != nil {
		return modkit.RaiseIO("fs.read_link", err)
	}
	return core.NewStringValue(res), nil
}

// Answer the system temporary directory.
//
//kavun:fn temp_dir slot=17 impure () -> string
func fsTempDir(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s := os.TempDir()
	return core.NewStringValue(s), nil
}

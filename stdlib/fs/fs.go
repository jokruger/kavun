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

func init() {
	m := core.NewModule(value.ModuleFs, "fs", false)
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
	m.Fn(members.Open, 0, fsOpen, 1, core.Fixed)
	m.Fn(members.OpenFile, 1, fsOpenFile, 3, core.Fixed)
	m.Fn(members.Create, 2, fsCreate, 1, core.Fixed)
	m.Fn(members.ReadFile, 3, fsReadFile, 1, core.Fixed)
	m.Fn(members.Stat, 4, fsStat, 1, core.Fixed)
	m.Fn(members.Mkdir, 5, fsMkdir, 2, core.Fixed)
	m.Fn(members.MkdirAll, 6, fsMkdirAll, 2, core.Fixed)
	m.Fn(members.Remove, 7, fsRemove, 1, core.Fixed)
	m.Fn(members.RemoveAll, 8, fsRemoveAll, 1, core.Fixed)
	m.Fn(members.Rename, 9, fsRename, 2, core.Fixed)
	m.Fn(members.Truncate, 10, fsTruncate, 2, core.Fixed)
	m.Fn(members.Chmod, 11, fsChmod, 2, core.Fixed)
	m.Fn(members.Chown, 12, fsChown, 3, core.Fixed)
	m.Fn(members.Lchown, 13, fsLchown, 3, core.Fixed)
	m.Fn(members.Link, 14, fsLink, 2, core.Fixed)
	m.Fn(members.Symlink, 15, fsSymlink, 2, core.Fixed)
	m.Fn(members.ReadLink, 16, fsReadlink, 1, core.Fixed)
	m.Fn(members.TempDir, 17, fsTempDir, 0, core.Fixed)
	// 18..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func fsOpen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.open", "1", len(args))
	}
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

func fsOpenFile(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.open_file", "3", len(args))
	}
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

func fsCreate(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.create", "1", len(args))
	}
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

func fsReadFile(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.read_file", "1", len(args))
	}
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

func fsStat(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.stat", "1", len(args))
	}

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

func fsMkdir(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.mkdir", "2", len(args))
	}
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

func fsMkdirAll(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.mkdir_all", "2", len(args))
	}
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

func fsRemove(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.remove", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.remove", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("fs.remove", os.Remove(s1))
}

func fsRemoveAll(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.remove_all", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("fs.remove_all", "first", "string(compatible)", args[0].TypeName())
	}
	return modkit.Done("fs.remove_all", os.RemoveAll(s1))
}

func fsRename(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.rename", "2", len(args))
	}
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

func fsTruncate(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.truncate", "2", len(args))
	}
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

func fsChmod(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.chmod", "2", len(args))
	}
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

func fsChown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.chown", "3", len(args))
	}
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

func fsLchown(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.lchown", "3", len(args))
	}
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

func fsLink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.link", "2", len(args))
	}
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

func fsSymlink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.symlink", "2", len(args))
	}
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

func fsReadlink(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.read_link", "1", len(args))
	}
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

func fsTempDir(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fs.temp_dir", "0", len(args))
	}
	s := os.TempDir()
	return core.NewStringValue(s), nil
}

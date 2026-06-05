package stdlib

import (
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSFile(a *core.Arena, vm core.VM, file *os.File) (core.Value, error) {
	// chdir() => true/error
	fileChdir := a.NewBuiltinClosureValue("chdir", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(a, file.Chdir())
	}, 0, false)

	// chown(uid int, gid int) => true/error
	fileChown := a.NewBuiltinClosureValue("chown", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chown", "2", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chown", "first", "int(compatible)", args[0].TypeName(a))
		}
		i2, ok := args[1].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chown", "second", "int(compatible)", args[1].TypeName(a))
		}
		return wrapError(a, file.Chown(int(i1), int(i2)))
	}, 2, false)

	// close() => error
	fileClose := a.NewBuiltinClosureValue("close", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(a, file.Close())
	}, 0, false)

	// name() => string
	fileName := a.NewBuiltinClosureValue("name", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		return a.NewStringValue(s), nil
	}, 0, false)

	// read_dir_names(n int) => array(string)/error
	fileReadDirNames := a.NewBuiltinClosureValue("read_dir_names", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read_dir_names", "1", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read_dir_names", "first", "int(compatible)", args[0].TypeName(a))
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(a, err)
		}
		arr := a.NewArray(len(res), false)
		for _, r := range res {
			t := a.NewStringValue(r)
			arr = append(arr, t)
		}
		return a.NewArrayValue(arr, false), nil
	}, 1, false)

	// sync() => error
	fileSync := a.NewBuiltinClosureValue("sync", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(a, file.Sync())
	}, 0, false)

	// write(bytes) => int/error
	fileWrite := a.NewBuiltinClosureValue("write", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsBytes(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0].TypeName(a))
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(a, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// write(string) => int/error
	fileWriteString := a.NewBuiltinClosureValue("write_string", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0].TypeName(a))
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(a, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// read(bytes) => int/error
	fileRead := a.NewBuiltinClosureValue("read", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0].TypeName(a))
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(a, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// chmod(mode int) => error
	fileChmod := a.NewBuiltinClosureValue("chmod", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0].TypeName(a))
		}
		return wrapError(a, file.Chmod(os.FileMode(i1)))
	}, 1, false)

	// seek(offset int, whence int) => int/error
	fileSeek := a.NewBuiltinClosureValue("seek", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.seek", "2", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.seek", "first", "int(compatible)", args[0].TypeName(a))
		}
		i2, ok := args[1].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.seek", "second", "int(compatible)", args[1].TypeName(a))
		}
		res, err := file.Seek(i1, int(i2))
		if err != nil {
			return wrapError(a, err)
		}
		return core.IntValue(res), nil
	}, 2, false)

	// stat() => idict(fileinfo)/error
	fileStat := a.NewBuiltinClosureValue("stat", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		t := a.NewStringValue(file.Name())
		return osStat(a, vm, []core.Value{t})
	}, 0, false)

	m := a.NewRecordValue(map[string]core.Value{
		"chdir":          fileChdir,
		"chown":          fileChown,
		"close":          fileClose,
		"name":           fileName,
		"read_dir_names": fileReadDirNames,
		"sync":           fileSync,
		"write":          fileWrite,
		"write_string":   fileWriteString,
		"read":           fileRead,
		"chmod":          fileChmod,
		"seek":           fileSeek,
		"stat":           fileStat,
	}, true)

	return m, nil
}

package stdlib

import (
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSFile(vm core.VM, file *os.File) (core.Value, error) {
	// chdir() => true/error
	fileChdir := core.NewBuiltinClosureValue("chdir", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(file.Chdir())
	}, 0, false)

	// chown(uid int, gid int) => true/error
	fileChown := core.NewBuiltinClosureValue("chown", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chown", "2", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chown", "first", "int(compatible)", args[0].TypeName())
		}
		i2, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chown", "second", "int(compatible)", args[1].TypeName())
		}
		return wrapError(file.Chown(int(i1), int(i2)))
	}, 2, false)

	// close() => error
	fileClose := core.NewBuiltinClosureValue("close", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(file.Close())
	}, 0, false)

	// name() => string
	fileName := core.NewBuiltinClosureValue("name", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		return core.NewStringValue(s), nil
	}, 0, false)

	// read_dir_names(n int) => array(string)/error
	fileReadDirNames := core.NewBuiltinClosureValue("read_dir_names", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read_dir_names", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read_dir_names", "first", "int(compatible)", args[0].TypeName())
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(err)
		}
		arr := make([]core.Value, 0, len(res))
		for _, r := range res {
			arr = append(arr, core.NewStringValue(r))
		}
		return core.NewArrayValue(arr, false), nil
	}, 1, false)

	// sync() => error
	fileSync := core.NewBuiltinClosureValue("sync", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(file.Sync())
	}, 0, false)

	// write(bytes) => int/error
	fileWrite := core.NewBuiltinClosureValue("write", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// write(string) => int/error
	fileWriteString := core.NewBuiltinClosureValue("write_string", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0].TypeName())
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// read(bytes) => int/error
	fileRead := core.NewBuiltinClosureValue("read", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)

	// chmod(mode int) => error
	fileChmod := core.NewBuiltinClosureValue("chmod", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(file.Chmod(os.FileMode(i1)))
	}, 1, false)

	// seek(offset int, whence int) => int/error
	fileSeek := core.NewBuiltinClosureValue("seek", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.seek", "2", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.seek", "first", "int(compatible)", args[0].TypeName())
		}
		i2, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.seek", "second", "int(compatible)", args[1].TypeName())
		}
		res, err := file.Seek(i1, int(i2))
		if err != nil {
			return wrapError(err)
		}
		return core.IntValue(res), nil
	}, 2, false)

	// stat() => idict(fileinfo)/error
	fileStat := core.NewBuiltinClosureValue("stat", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		return osStat(vm, []core.Value{core.NewStringValue(file.Name())})
	}, 0, false)

	m := core.NewRecordValue(map[string]core.Value{
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

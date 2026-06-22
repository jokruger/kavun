package stdlib

import (
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeOSFile(vm core.VM, file *os.File) (core.Value, error) {
	// chdir() => true/error
	fileChdir, err := a.NewBuiltinClosureValue("chdir", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(a, file.Chdir())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// chown(uid int, gid int) => true/error
	fileChown, err := a.NewBuiltinClosureValue("chown", func(vm core.VM, args []core.Value) (core.Value, error) {
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
		return wrapError(a, file.Chown(int(i1), int(i2)))
	}, 2, false)
	if err != nil {
		return core.Undefined, err
	}

	// close() => error
	fileClose, err := a.NewBuiltinClosureValue("close", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(a, file.Close())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// name() => string
	fileName, err := a.NewBuiltinClosureValue("name", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		return a.NewStringValue(s)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// read_dir_names(n int) => array(string)/error
	fileReadDirNames, err := a.NewBuiltinClosureValue("read_dir_names", func(vm core.VM, args []core.Value) (core.Value, error) {
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
		arr := a.NewArray(len(res), false)
		for _, r := range res {
			t, err := a.NewStringValue(r)
			if err != nil {
				return core.Undefined, err
			}
			a.PinAllocated(t)
			arr = append(arr, t)
		}
		return a.NewArrayValue(arr, false)
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// sync() => error
	fileSync, err := a.NewBuiltinClosureValue("sync", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(a, file.Sync())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// write(bytes) => int/error
	fileWrite, err := a.NewBuiltinClosureValue("write", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	if err != nil {
		return core.Undefined, err
	}

	// write(string) => int/error
	fileWriteString, err := a.NewBuiltinClosureValue("write_string", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	if err != nil {
		return core.Undefined, err
	}

	// read(bytes) => int/error
	fileRead, err := a.NewBuiltinClosureValue("read", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	if err != nil {
		return core.Undefined, err
	}

	// chmod(mode int) => error
	fileChmod, err := a.NewBuiltinClosureValue("chmod", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(a, file.Chmod(os.FileMode(i1)))
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// seek(offset int, whence int) => int/error
	fileSeek, err := a.NewBuiltinClosureValue("seek", func(vm core.VM, args []core.Value) (core.Value, error) {
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
	if err != nil {
		return core.Undefined, err
	}

	// stat() => idict(fileinfo)/error
	fileStat, err := a.NewBuiltinClosureValue("stat", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		t, err := a.NewStringValue(file.Name())
		if err != nil {
			return core.Undefined, err
		}
		return osStat(a, vm, []core.Value{t})
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
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
	if err != nil {
		return core.Undefined, err
	}

	return m, nil
}

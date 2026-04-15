package stdlib

import (
	"os"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeOSFile(vm core.VM, file *os.File) (core.Value, error) {
	alloc := vm.Allocator()

	// chdir() => true/error
	fileChdir, err := alloc.NewBuiltinFunctionValue("chdir", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(vm, file.Chdir())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// chown(uid int, gid int) => true/error
	fileChown, err := alloc.NewBuiltinFunctionValue("chown", func(vm core.VM, args []core.Value) (core.Value, error) {
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
		return wrapError(vm, file.Chown(int(i1), int(i2)))
	}, 2, false)
	if err != nil {
		return core.Undefined, err
	}

	// close() => error
	fileClose, err := alloc.NewBuiltinFunctionValue("close", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(vm, file.Close())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// name() => string
	fileName, err := alloc.NewBuiltinFunctionValue("name", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		if len(s) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("os.file.name")
		}
		return vm.Allocator().NewStringValue(s)
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// read_dir_names(n int) => array(string)/error
	fileReadDirNames, err := alloc.NewBuiltinFunctionValue("read_dir_names", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read_dir_names", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read_dir_names", "first", "int(compatible)", args[0].TypeName())
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(vm, err)
		}
		arr := make([]core.Value, 0, len(res))
		alloc := vm.Allocator()
		for _, r := range res {
			if len(r) > core.MaxStringLen {
				return core.Undefined, errs.NewStringLimitError("os.file.read_dir_names")
			}
			t, err := alloc.NewStringValue(r)
			if err != nil {
				return core.Undefined, err
			}
			arr = append(arr, t)
		}
		return alloc.NewArrayValue(arr, false)
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// sync() => error
	fileSync, err := alloc.NewBuiltinFunctionValue("sync", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(vm, file.Sync())
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	// write(bytes) => int/error
	fileWrite, err := alloc.NewBuiltinFunctionValue("write", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(vm, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// write(string) => int/error
	fileWriteString, err := alloc.NewBuiltinFunctionValue("write_string", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0].TypeName())
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(vm, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// read(bytes) => int/error
	fileRead, err := alloc.NewBuiltinFunctionValue("read", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(vm, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// chmod(mode int) => error
	fileChmod, err := alloc.NewBuiltinFunctionValue("chmod", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(vm, file.Chmod(os.FileMode(i1)))
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// seek(offset int, whence int) => int/error
	fileSeek, err := alloc.NewBuiltinFunctionValue("seek", func(vm core.VM, args []core.Value) (core.Value, error) {
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
			return wrapError(vm, err)
		}
		return core.IntValue(res), nil
	}, 2, false)
	if err != nil {
		return core.Undefined, err
	}

	// stat() => imap(fileinfo)/error
	fileStat, err := alloc.NewBuiltinFunctionValue("stat", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		t, err := vm.Allocator().NewStringValue(file.Name())
		if err != nil {
			return core.Undefined, err
		}
		return osStat(vm, []core.Value{t})
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := vm.Allocator().NewRecordValue(map[string]core.Value{
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

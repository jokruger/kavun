package stdlib

import (
	"os"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeOSFile(vm core.VM, file *os.File) core.Value {
	fileChdir := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(vm, file.Chdir()), nil
	}

	fileClose := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(vm, file.Close()), nil
	}

	fileSync := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(vm, file.Sync()), nil
	}

	fileName := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		if len(s) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("os.file.name")
		}
		return vm.Allocator().NewStringValue(s), nil
	}

	fileChown := func(vm core.VM, args []core.Value) (core.Value, error) {
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
		return wrapError(vm, file.Chown(int(i1), int(i2))), nil
	}

	fileWrite := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return core.IntValue(int64(res)), nil
	}

	fileRead := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0].TypeName())
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return core.IntValue(int64(res)), nil
	}

	fileWriteString := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0].TypeName())
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return core.IntValue(int64(res)), nil
	}

	fileReaddirnames := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.read_dir_names", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.read_dir_names", "first", "int(compatible)", args[0].TypeName())
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(vm, err), nil
		}
		arr := make([]core.Value, 0, len(res))
		alloc := vm.Allocator()
		for _, r := range res {
			if len(r) > core.MaxStringLen {
				return core.Undefined, errs.NewStringLimitError("os.file.read_dir_names")
			}
			arr = append(arr, alloc.NewStringValue(r))
		}
		return alloc.NewArrayValue(arr, false), nil
	}

	fileChmod := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0].TypeName())
		}
		return wrapError(vm, file.Chmod(os.FileMode(i1))), nil
	}

	fileSeek := func(vm core.VM, args []core.Value) (core.Value, error) {
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
			return wrapError(vm, err), nil
		}
		return core.IntValue(res), nil
	}

	fileStat := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		return osStat(vm, []core.Value{vm.Allocator().NewStringValue(file.Name())})
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecordValue(map[string]core.Value{
		"chdir":          alloc.NewBuiltinFunctionValue("chdir", fileChdir, 0, false),                 // chdir() => true/error
		"chown":          alloc.NewBuiltinFunctionValue("chown", fileChown, 2, false),                 // chown(uid int, gid int) => true/error
		"close":          alloc.NewBuiltinFunctionValue("close", fileClose, 0, false),                 // close() => error
		"name":           alloc.NewBuiltinFunctionValue("name", fileName, 0, false),                   // name() => string
		"read_dir_names": alloc.NewBuiltinFunctionValue("read_dir_names", fileReaddirnames, 1, false), // read_dir_names(n int) => array(string)/error
		"sync":           alloc.NewBuiltinFunctionValue("sync", fileSync, 0, false),                   // sync() => error
		"write":          alloc.NewBuiltinFunctionValue("write", fileWrite, 1, false),                 // write(bytes) => int/error
		"write_string":   alloc.NewBuiltinFunctionValue("write_string", fileWriteString, 1, false),    // write(string) => int/error
		"read":           alloc.NewBuiltinFunctionValue("read", fileRead, 1, false),                   // read(bytes) => int/error
		"chmod":          alloc.NewBuiltinFunctionValue("chmod", fileChmod, 1, false),                 // chmod(mode int) => error
		"seek":           alloc.NewBuiltinFunctionValue("seek", fileSeek, 2, false),                   // seek(offset int, whence int) => int/error
		"stat":           alloc.NewBuiltinFunctionValue("stat", fileStat, 0, false),                   // stat() => imap(fileinfo)/error
	}, true)
}

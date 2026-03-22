package stdlib

import (
	"os"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeOSFile(vm core.VM, file *os.File) *value.Record {
	fileChdir := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(vm, file.Chdir()), nil
	}

	fileClose := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(vm, file.Close()), nil
	}

	fileSync := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(vm, file.Sync()), nil
	}

	fileName := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		if len(s) > core.MaxStringLen {
			return nil, core.NewStringLimitError("os.file.name")
		}
		return vm.Allocator().NewString(s), nil
	}

	fileChown := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 2 {
			return nil, core.NewWrongNumArgumentsError("os.file.chown", "2", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.chown", "first", "int(compatible)", args[0])
		}
		i2, ok := args[1].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.chown", "second", "int(compatible)", args[1])
		}
		return wrapError(vm, file.Chown(int(i1), int(i2))), nil
	}

	fileWrite := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0])
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return vm.Allocator().NewInt(int64(res)), nil
	}

	fileRead := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0])
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return vm.Allocator().NewInt(int64(res)), nil
	}

	fileWriteString := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0])
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(vm, err), nil
		}
		return vm.Allocator().NewInt(int64(res)), nil
	}

	fileReaddirnames := func(vm core.VM, args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.readdirnames", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.readdirnames", "first", "int(compatible)", args[0])
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(vm, err), nil
		}
		arr := make([]core.Object, 0, len(res))
		alloc := vm.Allocator()
		for _, r := range res {
			if len(r) > core.MaxStringLen {
				return nil, core.NewStringLimitError("os.file.readdirnames")
			}
			arr = append(arr, alloc.NewString(r))
		}
		return alloc.NewArray(arr, false), nil
	}

	fileChmod := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0])
		}
		return wrapError(vm, file.Chmod(os.FileMode(i1))), nil
	}

	fileSeek := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, core.NewWrongNumArgumentsError("os.file.seek", "2", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.seek", "first", "int(compatible)", args[0])
		}
		i2, ok := args[1].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.seek", "second", "int(compatible)", args[1])
		}
		res, err := file.Seek(i1, int(i2))
		if err != nil {
			return wrapError(vm, err), nil
		}
		return vm.Allocator().NewInt(res), nil
	}

	fileStat := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		return osStat(vm, vm.Allocator().NewString(file.Name()))
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecord(map[string]core.Object{
		"chdir":        alloc.NewBuiltinFunction("chdir", fileChdir, 0, false),               // chdir() => true/error
		"chown":        alloc.NewBuiltinFunction("chown", fileChown, 2, false),               // chown(uid int, gid int) => true/error
		"close":        alloc.NewBuiltinFunction("close", fileClose, 0, false),               // close() => error
		"name":         alloc.NewBuiltinFunction("name", fileName, 0, false),                 // name() => string
		"readdirnames": alloc.NewBuiltinFunction("readdirnames", fileReaddirnames, 1, false), // readdirnames(n int) => array(string)/error
		"sync":         alloc.NewBuiltinFunction("sync", fileSync, 0, false),                 // sync() => error
		"write":        alloc.NewBuiltinFunction("write", fileWrite, 1, false),               // write(bytes) => int/error
		"write_string": alloc.NewBuiltinFunction("write_string", fileWriteString, 1, false),  // write(string) => int/error
		"read":         alloc.NewBuiltinFunction("read", fileRead, 1, false),                 // read(bytes) => int/error
		"chmod":        alloc.NewBuiltinFunction("chmod", fileChmod, 1, false),               // chmod(mode int) => error
		"seek":         alloc.NewBuiltinFunction("seek", fileSeek, 2, false),                 // seek(offset int, whence int) => int/error
		"stat":         alloc.NewBuiltinFunction("stat", fileStat, 0, false),                 // stat() => imap(fileinfo)/error
	}, true).(*value.Record)
}

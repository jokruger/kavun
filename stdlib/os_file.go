package stdlib

import (
	"os"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeOSFile(file *os.File) *value.Record {
	fileChdir := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.chdir", "0", len(args))
		}
		return wrapError(file.Chdir()), nil
	}

	fileClose := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.close", "0", len(args))
		}
		return wrapError(file.Close()), nil
	}

	fileSync := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.sync", "0", len(args))
		}
		return wrapError(file.Sync()), nil
	}

	fileName := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.name", "0", len(args))
		}
		s := file.Name()
		if len(s) > core.MaxStringLen {
			return nil, core.NewStringLimitError("os.file.name")
		}
		return value.NewString(s), nil
	}

	fileChown := func(args ...core.Object) (ret core.Object, err error) {
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
		return wrapError(file.Chown(int(i1), int(i2))), nil
	}

	fileWrite := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.write", "1", len(args))
		}
		y1, ok := args[0].AsByteSlice()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.write", "first", "bytes(compatible)", args[0])
		}
		res, err := file.Write(y1)
		if err != nil {
			return wrapError(err), nil
		}
		return value.NewInt(int64(res)), nil
	}

	fileRead := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.read", "1", len(args))
		}
		y1, ok := args[0].AsByteSlice()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.read", "first", "bytes(compatible)", args[0])
		}
		res, err := file.Read(y1)
		if err != nil {
			return wrapError(err), nil
		}
		return value.NewInt(int64(res)), nil
	}

	fileWriteString := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.write_string", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.write_string", "first", "string(compatible)", args[0])
		}
		res, err := file.WriteString(s1)
		if err != nil {
			return wrapError(err), nil
		}
		return value.NewInt(int64(res)), nil
	}

	fileReaddirnames := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.readdirnames", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.readdirnames", "first", "int(compatible)", args[0])
		}
		res, err := file.Readdirnames(int(i1))
		if err != nil {
			return wrapError(err), nil
		}
		arr := make([]core.Object, 0, len(res))
		for _, r := range res {
			if len(r) > core.MaxStringLen {
				return nil, core.NewStringLimitError("os.file.readdirnames")
			}
			arr = append(arr, value.NewString(r))
		}
		return value.NewArray(arr, false), nil
	}

	fileChmod := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("os.file.chmod", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("os.file.chmod", "first", "int(compatible)", args[0])
		}
		return wrapError(file.Chmod(os.FileMode(i1))), nil
	}

	fileSeek := func(args ...core.Object) (core.Object, error) {
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
			return wrapError(err), nil
		}
		return value.NewInt(res), nil
	}

	fileStat := func(args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError("os.file.stat", "0", len(args))
		}
		return osStat(value.NewString(file.Name()))
	}

	return value.NewRecord(map[string]core.Object{
		"chdir":        value.NewBuiltinFunction("chdir", fileChdir, 0, false),               // chdir() => true/error
		"chown":        value.NewBuiltinFunction("chown", fileChown, 2, false),               // chown(uid int, gid int) => true/error
		"close":        value.NewBuiltinFunction("close", fileClose, 0, false),               // close() => error
		"name":         value.NewBuiltinFunction("name", fileName, 0, false),                 // name() => string
		"readdirnames": value.NewBuiltinFunction("readdirnames", fileReaddirnames, 1, false), // readdirnames(n int) => array(string)/error
		"sync":         value.NewBuiltinFunction("sync", fileSync, 0, false),                 // sync() => error
		"write":        value.NewBuiltinFunction("write", fileWrite, 1, false),               // write(bytes) => int/error
		"write_string": value.NewBuiltinFunction("write_string", fileWriteString, 1, false),  // write(string) => int/error
		"read":         value.NewBuiltinFunction("read", fileRead, 1, false),                 // read(bytes) => int/error
		"chmod":        value.NewBuiltinFunction("chmod", fileChmod, 1, false),               // chmod(mode int) => error
		"seek":         value.NewBuiltinFunction("seek", fileSeek, 2, false),                 // seek(offset int, whence int) => int/error
		"stat":         value.NewBuiltinFunction("stat", fileStat, 0, false),                 // stat() => imap(fileinfo)/error
	}, true)
}

package stdlib

import (
	"os"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeOSFile(file *os.File) *value.ImmutableMap {
	fileChdir := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(file.Chdir()), nil
	}

	fileClose := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(file.Close()), nil
	}

	fileSync := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, gse.ErrWrongNumArguments
		}
		return wrapError(file.Sync()), nil
	}

	return &value.ImmutableMap{
		Value: map[string]core.Object{
			// chdir() => true/error
			"chdir": &value.BuiltinFunction{
				Name:  "chdir",
				Value: fileChdir,
			}, //
			// chown(uid int, gid int) => true/error
			"chown": &value.BuiltinFunction{
				Name:  "chown",
				Value: FuncAIIRE(file.Chown),
			}, //
			// close() => error
			"close": &value.BuiltinFunction{
				Name:  "close",
				Value: fileClose,
			}, //
			// name() => string
			"name": &value.BuiltinFunction{
				Name:  "name",
				Value: FuncARS(file.Name),
			}, //
			// readdirnames(n int) => array(string)/error
			"readdirnames": &value.BuiltinFunction{
				Name:  "readdirnames",
				Value: FuncAIRSsE(file.Readdirnames),
			}, //
			// sync() => error
			"sync": &value.BuiltinFunction{
				Name:  "sync",
				Value: fileSync,
			}, //
			// write(bytes) => int/error
			"write": &value.BuiltinFunction{
				Name:  "write",
				Value: FuncAYRIE(file.Write),
			}, //
			// write(string) => int/error
			"write_string": &value.BuiltinFunction{
				Name:  "write_string",
				Value: FuncASRIE(file.WriteString),
			}, //
			// read(bytes) => int/error
			"read": &value.BuiltinFunction{
				Name:  "read",
				Value: FuncAYRIE(file.Read),
			}, //
			// chmod(mode int) => error
			"chmod": &value.BuiltinFunction{
				Name: "chmod",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 1 {
						return nil, gse.ErrWrongNumArguments
					}
					i1, ok := args[0].AsInt()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return wrapError(file.Chmod(os.FileMode(i1))), nil
				},
			},
			// seek(offset int, whence int) => int/error
			"seek": &value.BuiltinFunction{
				Name: "seek",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 2 {
						return nil, gse.ErrWrongNumArguments
					}
					i1, ok := args[0].AsInt()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					i2, ok := args[1].AsInt()
					if !ok {
						return nil, gse.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "int(compatible)",
							Found:    args[1].TypeName(),
						}
					}
					res, err := file.Seek(i1, int(i2))
					if err != nil {
						return wrapError(err), nil
					}
					return &value.Int{Value: res}, nil
				},
			},
			// stat() => imap(fileinfo)/error
			"stat": &value.BuiltinFunction{
				Name: "stat",
				Value: func(args ...core.Object) (core.Object, error) {
					if len(args) != 0 {
						return nil, gse.ErrWrongNumArguments
					}
					return osStat(&value.String{Value: file.Name()})
				},
			},
		},
	}
}

package fs

import (
	"os"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fs.file's members: one function per entry of TypeFile.Methods that is not a shared member. Every one touches the
// world (IMPURE); a refusal from the world is an io error named fs.file.<member>.

func fileContext(id member.ID) string {
	return "fs.file." + id.String()
}

// fileDone answers undefined for a statement-like member, or the io error.
func fileDone(id member.ID, err error) (core.Value, error) {
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return core.Undefined, nil
}

// IMPURE by contract
func fileName(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "0", len(args))
	}
	return core.NewStringValue(FileOf(v).Name()), nil
}

// IMPURE by contract — read(buffer) fills a bytes buffer and answers the count read
func fileRead(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "1", len(args))
	}
	buf, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "bytes(compatible)", args[0].TypeName())
	}
	n, err := FileOf(v).Read(buf)
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return core.IntValue(int64(n)), nil
}

// IMPURE by contract
func fileWrite(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "1", len(args))
	}
	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "bytes(compatible)", args[0].TypeName())
	}
	n, err := FileOf(v).Write(b)
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return core.IntValue(int64(n)), nil
}

// IMPURE by contract
func fileWriteString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "1", len(args))
	}
	s, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "string(compatible)", args[0].TypeName())
	}
	n, err := FileOf(v).WriteString(s)
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return core.IntValue(int64(n)), nil
}

// IMPURE by contract
func fileClose(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "0", len(args))
	}
	return fileDone(id, FileOf(v).Close())
}

// IMPURE by contract
func fileSync(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "0", len(args))
	}
	return fileDone(id, FileOf(v).Sync())
}

// IMPURE by contract — seek(offset, whence) answers the new offset
func fileSeek(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "2", len(args))
	}
	off, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "int(compatible)", args[0].TypeName())
	}
	whence, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "second", "int(compatible)", args[1].TypeName())
	}
	pos, err := FileOf(v).Seek(off, int(whence))
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return core.IntValue(pos), nil
}

// IMPURE by contract
func fileStat(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "0", len(args))
	}
	fi, err := FileOf(v).Stat()
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	return NewStatValue(fi), nil
}

// IMPURE by contract
func fileChmod(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "1", len(args))
	}
	mode, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "int(compatible)", args[0].TypeName())
	}
	return fileDone(id, FileOf(v).Chmod(os.FileMode(mode)))
}

// IMPURE by contract
func fileChown(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "2", len(args))
	}
	uid, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "int(compatible)", args[0].TypeName())
	}
	gid, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "second", "int(compatible)", args[1].TypeName())
	}
	return fileDone(id, FileOf(v).Chown(int(uid), int(gid)))
}

// IMPURE by contract
func fileChdir(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "0", len(args))
	}
	return fileDone(id, FileOf(v).Chdir())
}

// IMPURE by contract — read_dir_names(n): up to n names of a directory's entries (n <= 0: all of them)
func fileReadDirNames(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(fileContext(id), "1", len(args))
	}
	n, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError(fileContext(id), "first", "int(compatible)", args[0].TypeName())
	}
	names, err := FileOf(v).Readdirnames(int(n))
	if err != nil {
		return core.Undefined, errs.NewIOError(fileContext(id), err)
	}
	out := make([]core.Value, 0, len(names))
	for _, name := range names {
		out = append(out, core.NewStringValue(name))
	}
	return core.NewArrayValue(out, false), nil
}

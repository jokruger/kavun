package fs

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
)

// fs.file is an open file. It is owned by the fs module — answered by fs.open, fs.open_file and fs.create — and
// registered by it (core.ModuleBuilder.Type). It is a handle on the world: every member is impure, and
// two values are equal only when they are the same handle.

const fileTypeName = "fs.file"

// NewFileValue wraps an open file.
func NewFileValue(f *os.File) core.Value {
	return core.Value{Type: value.FsFile, Immutable: true, Ptr: unsafe.Pointer(f)}
}

// FileOf unwraps an fs.file value.
func FileOf(v core.Value) *os.File {
	return (*os.File)(v.Ptr)
}

// TypeFile is the fs.file type descriptor.
var TypeFile = core.ValueTypeDescr{
	Name:   core.ConstHook(fileTypeName),            // PURE by contract
	String: fileTypeString,                          // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  core.SamePtrEqual,                       // PURE by contract

	Methods: []core.Method{
		members.IsTrue:       core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.Format:       core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:         core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:       core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Name:         core.NewMethod(fileName, 0, core.Fixed),
		members.Read:         core.NewMethod(fileRead, 1, core.Fixed),
		members.Write:        core.NewMethod(fileWrite, 1, core.Fixed),
		members.WriteString:  core.NewMethod(fileWriteString, 1, core.Fixed),
		members.Close:        core.NewMethod(fileClose, 0, core.Fixed),
		members.Sync:         core.NewMethod(fileSync, 0, core.Fixed),
		members.Seek:         core.NewMethod(fileSeek, 2, core.Fixed),
		members.Stat:         core.NewMethod(fileStat, 0, core.Fixed),
		members.Chmod:        core.NewMethod(fileChmod, 1, core.Fixed),
		members.Chown:        core.NewMethod(fileChown, 2, core.Fixed),
		members.Chdir:        core.NewMethod(fileChdir, 0, core.Fixed),
		members.ReadDirNames: core.NewMethod(fileReadDirNames, 1, core.Fixed),
	},
}

// A file renders as <fs.file:NAME>; it has no text content, so no string() member.
func fileTypeString(v core.Value) string {
	return fmt.Sprintf("<%s:%s>", fileTypeName, FileOf(v).Name())
}

// NewStatValue is the frozen record fs.stat and file.stat() answer: facts about a file at one moment.
func NewStatValue(fi os.FileInfo) core.Value {
	return core.NewRecordValue(map[string]core.Value{
		"name":      core.NewStringValue(fi.Name()),
		"mtime":     core.NewTimeValue(fi.ModTime()),
		"size":      core.IntValue(fi.Size()),
		"mode":      core.IntValue(int64(fi.Mode())),
		"directory": core.BoolValue(fi.IsDir()),
	}, true)
}

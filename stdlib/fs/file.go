package fs

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/jokruger/kavun/core"
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

// An open file, as `fs.open`, `fs.open_file` and `fs.create` answer it.
//
// ## Overview
//
// `fs.file` is a handle on the world: it is owned by the [`fs`](../stdlib/fs.md) module, every member touches the file
// system, and a refusal raises `io` naming the member (`(fs.file.read) read …: bad file descriptor`). It has no text
// content, so it renders instead (`format(f)` → `<fs.file:./out.txt>`) and has no `string()`.
//
// ```go
// fs := import("fs")
// f := fs.create("./out.txt")
// f.write_string("hello")      // => 5 — the number of bytes written
// f.seek(0, fs.seek_set)       // => 0
// buf := bytes(b"\x00", 5)
// f.read(buf)                  // => 5 — buf now holds "hello"
// f.stat().size                // => 5
// f.close()
// ```
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `==` / `!=` | the same open file (identity: two opens of one path are different files) |
// | `!` | a file is always truthy |
// | everything else | raises |
//
//kavun:type fs.file

// TypeFile is the fs.file type descriptor.
var TypeFile = core.ValueTypeDescr{
	Name:   core.ConstHook(fileTypeName),            // PURE by contract
	String: fileTypeString,                          // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  core.SamePtrEqual,                       // PURE by contract

	Methods: fileMethods,
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

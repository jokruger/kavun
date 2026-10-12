# fs.file

An open file, as `fs.open`, `fs.open_file` and `fs.create` answer it.

## Overview

`fs.file` is a handle on the world: it is owned by the [`fs`](../stdlib.md#fs) module, every member touches the file
system, and a refusal raises `io` naming the member (`(fs.file.read) read …: bad file descriptor`). It has no text
content, so it renders instead (`format(f)` → `<fs.file:./out.txt>`) and has no `string()`.

```go
fs := import("fs")
f := fs.create("./out.txt")
f.write_string("hello")      // 5 — the number of bytes written
f.seek(0, fs.seek_set)       // 0
buf := bytes(b"\x00", 5)
f.read(buf)                  // 5 — buf now holds "hello"
f.stat().size                // 5
f.close()
```

## Operators

| operator | reading |
| --- | --- |
| `==` / `!=` | the same open file (identity: two opens of one path are different files) |
| `!` | a file is always truthy |
| everything else | raises |

## Members

| member | answers |
| --- | --- |
| `name()` | the path it was opened with |
| `read(buffer)` | fills a `bytes` buffer, answers the count read |
| `write(bytes)` / `write_string(s)` | the count written |
| `seek(offset, whence)` | the new offset; `whence` is `fs.seek_set`, `fs.seek_cur` or `fs.seek_end` |
| `stat()` | the frozen record `{name, mtime, size, mode, directory}` (`fs.stat`'s) |
| `read_dir_names(n)` | up to `n` entry names of a directory (`n <= 0`: all) |
| `sync()`, `close()`, `chmod(mode)`, `chown(uid, gid)`, `chdir()` | `undefined` |
| `is_true()`, `format([spec])`, `copy()`, `freeze()` | `true`; the render; the file itself |

## Migration notes

- The file was a frozen record of closures (`type_name` `immutable-record`, `f.close` a function value); it is a
  type now: `type_name(f)` is `fs.file`, and the members are called the same way. Error contexts say
  `fs.file.<member>` (were `os.file.<member>`).

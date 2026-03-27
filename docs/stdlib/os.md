# Module `os`

```text
os := import("os")
```

## Constants

`platform`, `arch`, `o_rd`, `o_wr`, `o_rdwr`, `o_append`, `o_create`, `o_excl`, `o_sync`, `o_trunc`, `mode_dir`, `mode_append`, `mode_exclusive`, `mode_temporary`, `mode_symlink`, `mode_device`, `mode_named_pipe`, `mode_socket`, `mode_set_uid`, `mode_set_gui`, `mode_char_device`, `mode_sticky`, `mode_type`, `mode_perm`, `path_separator`, `path_list_separator`, `dev_null`, `seek_set`, `seek_cur`, `seek_end`.

## Functions

All functions return either the requested value or an `error(...)` object.

- `args() => [string]`
- `chdir(dir string)`
- `chmod(path string, mode int)`
- `chown(path string, uid int, gid int)`
- `clear_env()`
- `environ() => [string]`
- `exit(code int)`
- `expand_env(text string) => string`
- `get_egid() => int`
- `get_env(key string) => string`
- `get_euid() => int`
- `get_gid() => int`
- `get_groups() => [int]`
- `get_page_size() => int`
- `get_pid() => int`
- `get_ppid() => int`
- `get_uid() => int`
- `get_wd() => string`
- `hostname() => string`
- `lchown(path string, uid int, gid int)`
- `link(old string, new string)`
- `lookup_env(key string) => string/false`
- `mkdir(path string, perm int)`
- `mkdir_all(path string, perm int)`
- `read_link(path string) => string`
- `remove(path string)`
- `remove_all(path string)`
- `rename(old string, new string)`
- `set_env(key string, value string)`
- `symlink(old string, new string)`
- `temp_dir() => string`
- `truncate(path string, size int)`
- `unset_env(key string)`
- `create(path string) => file`
- `open(path string) => file`
- `open_file(path string, flag int, perm int) => file`
- `find_process(pid int) => process`
- `start_process(name string, argv [string], dir string, env [string]) => process`
- `exec_look_path(file string) => string`
- `exec(name string, args...) => command`
- `stat(path string) => record{name, mtime, size, mode, directory}`
- `read_file(path string) => bytes`

## File records

`create`, `open`, and `open_file` return immutable records with methods:

`chdir()`, `chown(uid, gid)`, `close()`, `name()`, `read_dir_names(n)`, `sync()`, `write(bytes)`, `write_string(string)`, `read(bytes)`, `chmod(mode)`, `seek(offset, whence)`, `stat()`.

## Process records

`find_process` and `start_process` return process records with methods:

`kill()`, `release()`, `signal(signal int)`, `wait() => process_state`.

Process states expose: `pid()`, `exited()`, `success()`, `string()`.

## Command records

`exec` returns a command record that mirrors Go's `exec.Cmd`:

`combined_output() => bytes`, `output() => bytes`, `run()`, `start()`, `wait()`, `set_path(path)`, `set_dir(dir)`, `set_env(env [string])`, `process() => process`.

These helpers are used heavily in `tests/unit` to validate interop and mirror Go's own APIs as closely as possible.

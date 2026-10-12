# exec.command

A prepared external command, as `exec.exec(name, args...)` answers it.

## Overview

`exec.command` is owned by the [`exec`](../stdlib.md#exec) module. Configure it (`set_dir`, `set_env`, `set_path`),
then run it once — `run()`, `output()`, `combined_output()`, or `start()` followed by `wait()`. A refusal or a failed
run raises `io` naming the member. It renders as `<exec.command:NAME>`; no `string()`.

```go
exec := import("exec")
string(exec.exec("echo", "hi").output())   // "hi\n" — output() answers bytes
c := exec.exec("sh", "-c", "exit 3")
c.start()
c.process().wait().success            // false
```

## Operators

| operator | reading |
| --- | --- |
| `==` / `!=` | the same command (identity) |
| `!` | a command is always truthy |
| everything else | raises |

## Members

| member | answers |
| --- | --- |
| `run()` | `undefined`; raises `io` if it cannot start or ends with a failure status |
| `output()` / `combined_output()` | its stdout / stdout and stderr together, as `bytes` |
| `start()`, `wait()` | `undefined` — start it, then wait for it to end |
| `process()` | the started command's [`exec.process`](exec-process.md); before `start()` raises `invalid_value` |
| `set_path(p)`, `set_dir(d)`, `set_env(["K=V", …])` | `undefined` — configure before running |
| `is_true()`, `format([spec])`, `copy()`, `freeze()` | `true`; the render; the command itself |

## Migration notes

- Was `os.exec(...)`, a frozen record of closures; it is a type now (`type_name` `exec.command`). Error contexts say
  `exec.command.<member>` (were `os.exec.<member>`). `process()` before `start()` raises instead of answering a
  process that fails on first use.

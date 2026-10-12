# exec.process

A process, as `exec.start_process`, `exec.find_process` and `command.process()` answer it.

## Overview

`exec.process` is owned by the [`exec`](../stdlib.md#exec) module. Every member reaches the operating system; a
refusal raises `io` naming the member. It renders as `<exec.process:PID>`; no `string()`.

```go
exec := import("exec")
p := exec.start_process("/bin/sleep", ["sleep", "1"], "", [])
p.pid() > 0                // true
st := p.wait()             // {pid: …, exited: true, success: true, status: "exit status 0"}
st.success                 // true
```

## Operators

| operator | reading |
| --- | --- |
| `==` / `!=` | the same process (by pid) |
| `!` | a process is always truthy |
| everything else | raises |

## Members

| member | answers |
| --- | --- |
| `pid()` | its process id |
| `wait()` | blocks until it ends; answers the frozen record `{pid, exited, success, status}` — facts, not behaviour (`status` is the system's text: `"exit status 3"`, `"signal: killed"`) |
| `kill()`, `signal(sig)`, `release()` | `undefined` |
| `is_true()`, `format([spec])`, `copy()`, `freeze()` | `true`; the render; the process itself |

## Migration notes

- Was a frozen record of closures (`os.start_process`, `os.find_process`, `cmd.process()`); it is a type now
  (`type_name` `exec.process`), and gained `pid()`. Error contexts say `exec.process.<member>` (were
  `os.process.<member>`).
- **`wait()` answers values, not functions:** `st.success` (was `st.success()`), `st.exited`, `st.pid`, and the text
  is `st.status` (was `st.string()`).

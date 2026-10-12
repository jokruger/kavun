# Embedding Kavun In Go

The recommended embedding API is `kavun.Script`. It wraps parsing, compilation, bindings setup, and VM execution into a
higher-level workflow that is easier to integrate and maintain in Go applications.

For lower-level control over compilation and execution, direct use of the compiler and VM is still available; this
document focuses on the Script-first approach.

## Quick Start

The primary pattern: create a script, compile once, then run multiple times.

```go
package main

import (
	"fmt"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/vm"
)

func main() {
	src := []byte(`
fib := func(x) {
    if x < 2 {
        return x
    }
    return fib(x-1) + fib(x-2)
}
out = fib(10)
`)

	// Create and configure script
	script := kavun.NewScript(src, "out")

	// Create VM
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// Compile once
	compiled, err := script.Compile()
	if err != nil {
		panic(err)
	}

	// Run repeatedly with the same compiled code
	for i := 0; i < 100; i++ {
		if err := compiled.Run(machine); err != nil {
			panic(err)
		}
	}

	fmt.Println("result:", compiled.Get("out").String())
}
```

## Inputs and Outputs

Setup input/output bindings before compilation:

```go
script := kavun.NewScript(src, "x", "y", "out")
compiled, err := script.Compile()
```

Before each run, set input values with `compiled.Set(...)`:

```go
if err := compiled.Set("x", core.IntValue(50)); err != nil {
    panic(err)
}
if err := compiled.Set("y", core.IntValue(7)); err != nil {
    panic(err)
}
if err := compiled.Run(machine); err != nil {
    panic(err)
}
```

After execution, retrieve output values with `Get`:

```go
out := compiled.Get("out")
sum, _ := out.AsInt()
fmt.Println(sum)
```

## Modules and Imports

### The module grant

A builtin module either only computes (`math`, `json`, `fin`, … — *pure*) or touches the world (`clock`,
`exec`, `fs`, `io`, `os`). By default a script may import **only the pure modules** (`core.PureModuleNames()`); anything
else must be granted. The grant is checked twice — by the compiler, and again by the VM that runs the bytecode —
so bytecode compiled elsewhere or decoded from a file cannot bypass it:

```go
script.SetAllowedModules("math", "os") // compile-time gate: exactly these
compiled, err := script.Compile()

machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
if err := machine.SetAllowedModules("math", "os"); err != nil { // run-time gate: an unknown name is a host error
    return err
}
err = compiled.Run(machine)
```

`Script.AllowAllModules()` and `VM.AllowAllModules()` grant everything (the `kavun` CLI does both). A refusal at
compile time is a compile error (`module 'os' is not allowed to import`); at run time it is the recoverable kind
`module_not_allowed` (`module 'os' is not allowed`). The VM keeps its grant across `Reset`. `kavun.Eval` runs with
the default grant. The **grant is the security boundary** for what a script may do to the host: grant world
modules only to scripts you trust with them.

### Standard streams

`VM.SetStdout(w)` and `VM.SetStdin(r)` set the streams the VM hands to modules that print or read (default
`os.Stdout` / `os.Stdin`); `VM.Stdout()` / `VM.Stdin()` answer them, also through the `core.VM` interface.

### Your own builtin module

An embedder registers its own builtin modules beside the standard ones. A builtin module is a type; an
embedder's lives in the embedder range (type ids 192–255, shared with embedder types). Declare its
functions and constants through `core.NewModule`, each under a member id (a builtin name, or one you bind with
`member.Define`), and a builtin-function slot 0–127:

```go
const myModule = uint8(200)

var (
    memberDouble = member.ID(2100) // embedder member ids start at member.FirstUserDefined (2048)
    memberLabel  = member.ID(2101)
)

func init() {
    member.Define(memberDouble, "double")
    member.Define(memberLabel, "label")
    err := core.NewModule(myModule, "mymod", true). // true: pure — granted by default
        PureFn(memberDouble, 0, func(_ core.VM, args []core.Value) (core.Value, error) {
            n, _ := args[0].AsInt()
            return core.IntValue(2 * n), nil
        }, 1, core.Fixed).
        Const(memberLabel, core.NewStringValue("mine")).
        Register()
    if err != nil {
        panic(err)
    }
}
```

`PureFn` claims a function pure (the optimizer may fold its calls), `Fn` does not. `New(f)` makes `import` answer a
fresh value per import instead of one singleton, `Type(typeID, descr)` registers a type the module owns (a handle or
record type its functions answer, in the embedder range like the module), and `Descr()` lets you amend the module
type's hooks before `Register`. `core.UnregisterModule(typeID)` removes an embedder module and its types again
(teardown, tests); `kavun.ModuleNames()` lists every registered module. The standard modules are built the same
way, one package each under `stdlib/` — `stdlib/fs` is a compact example of a module with an owned type.

### Imports from source

```go

// Enable/disable file imports
script.DisableFileImport()
script.EnableFileImport()

// In-memory source module
script.AddCustomModule("helpers", []byte(`
export {add: func(a, b) { return a + b }}
`))
```

## Handling errors

A run ends in one of three ways: `nil`, the context's error, or a `*kavun.RuntimeError`. Compile errors are a
separate type and come from `Script.Compile`, never from a run.

```go
type RuntimeError struct {
    Kind    string              // "division_by_zero", "conversion", "user", … or "" if the failure carried no kind
    Fatal   bool                // bypassed every recover(); the VM was stopped
    Message string              // human-readable detail, without the kind prefix
    Payload core.Value          // the error value's payload — a string for runtime errors,
                                // whatever the script passed for raise(x)
    Trace   []ast.SourceFilePos // active frames, innermost first
}
```

Branch on the struct, never on the text:

```go
if err := compiled.Run(machine); err != nil {
    var re *kavun.RuntimeError
    switch {
    case errors.As(err, &re):
        if re.Fatal {
            log.Printf("script stopped: %s: %s at %s", re.Kind, re.Message, re.Trace[0])
            break
        }
        // A recoverable error simply means the script did not handle it.
        log.Printf("script failed: %s: %s", re.Kind, re.Message)
    default:
        // context cancellation, or a host-side problem
        log.Printf("run failed: %v", err)
    }
}
```

`Error()` renders `Runtime Error: <kind>: <message>` plus one `\n\tat file:line:col` line per frame. The sentinel
comparisons keep working through `Unwrap`:

```go
errors.Is(err, errs.ErrStackOverflow)   // true for a stack overflow
errs.AsError(err)                        // the underlying *errs.Error
errs.IsCritical(err)                     // same answer as re.Fatal
```

### `Payload` is the structured channel

A runtime failure's payload is its message string. A script's own `raise(x)` passes `x` through untouched, so a
script can hand the host a structured rejection rather than a sentence:

```go
raise({reason: "limit_exceeded", field: "amount", max: 10000})
```

```go
if errors.As(err, &re) && re.Kind == "user" {
    reason, _ := re.Payload.AccessIndex(core.NewStringValue("reason"))
    // …
}
```

Every kind a run can answer, with its severity and category, is listed in
[types/error.md § Every error kind](types/error.md#every-error-kind).

### Which failures are the caller's fault

`Kind` says what failed, not whose fault it was — and it cannot: `int(x)` does not know where `x` came from.
Three kinds carry that signal:

| `Kind` | read it as |
| --- | --- |
| `requirement` | the script's own input check rejected your data; `Payload` says which field and why |
| `conversion`, `json_decoding`, `io` | the data or the world did not cooperate |
| anything else | most likely a defect in the script |

### Recoverable vs. fatal

| | reaches the script's `recover()` | means |
| --- | --- | --- |
| recoverable | yes | the script or its data is wrong, or the world refused; the script chose not to handle it |
| fatal | no | the VM could not continue: stack overflow, an internal invariant, a panicking host hook, a host setup mistake |

A panicking hook — in a custom `ValueTypeDescr`, a host builtin, anywhere — never crosses `Run`. It is contained
and answered as a fatal `internal` error carrying the panic text and the script's stack trace. Treat one as a bug
report about the host code, not as a condition to handle. See `docs/extending-types.md`.

### A script can answer instead of failing

When the host wants a decision rather than an error, the script assigns its outcome to a bound global and handles
the failure itself with a top-level `defer` (see `docs/language.md` § "Deferred calls"). The run then returns
`nil` and the host reads the outcome back:

```go
// script:
//   decision = {ok: true}
//   defer func() {
//       e := recover()
//       if e != undefined { decision = {ok: false, why: e.value()} }
//   }()
//   …rules…

if err := compiled.Run(machine); err != nil { /* a defect, not a rejection */ }
decision, _ := compiled.Get("decision")
```

### What survives a failed run

The globals hold whatever the script assigned up to the point of failure, and `Compiled.Get`/`GetAll` read them
after an error just as after a success.

## Reusing a VM after a failure

`Compiled.Run` and `Compiled.RunContext` call `VM.Reset` before every run, so **the same VM is usable again after
every outcome**: success, a recoverable error, a fatal error, a stack overflow, a contained panic, and a cancelled
context. No `Clear()` is needed for correctness — see [Memory Management](#memory-management) for what it is for.

One run at a time, though: a `VM` must never be shared between goroutines (see [Concurrency](#concurrency)).

Cancellation is the one outcome that is not an error the script could have seen: `RunContext` answers `ctx.Err()`,
and the script's deferred calls — top-level ones included — do **not** run.

## Concurrency

`Script`, `Compiled`, and `VM` are **not thread-safe**. For parallel execution:

1. Each goroutine must use its own `Compiled` (via `Clone`)
2. Each goroutine must use its own VM
3. Protect shared resources with explicit locking

Safe pattern for parallel runs:

```go
base, err := script.Compile()
if err != nil {
    panic(err)
}

// Each goroutine clones the compiled code
clone, err := base.Clone()
if err != nil {
    panic(err)
}

// Each goroutine has isolated runtime resources
machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

if err := clone.Run(machine); err != nil {
    panic(err)
}
```

For cancellable execution, use `RunContext(ctx, machine)`.

## dec128 process-global settings

Kavun's `decimal` is [dec128](https://github.com/jokruger/dec128), which has five process-global settings
(`SetDefaultScale`, `SetArithmeticRounding`, `SetLossPolicy`, `SetTrimOutput`, `SetNullValue`). Kavun never
sets them, and scripts cannot read or change them; the documented behaviour of `decimal` assumes dec128's
defaults. They are process-wide, so a host that changes them — directly or through another package — changes
what `/`, `sqrt()`, `pow()`, `json.encode` and, past the 19-place ceiling, `+ - *` return inside every script.
That is the host's decision to make. The members that take an explicit `(scale, mode)` — `round`,
`div_round`, `mul_round` and the rest of the `*_round` family — never read them.

## Time zone data

A script names zones by their IANA name (`t.in_zone("Europe/Kyiv")`, `time({…, zone: "Europe/Kyiv"})`); `"UTC"`
is always available. The host's own zone is never visible to a script: `"Local"` and `""` are not zone names,
and `clock.now()` answers in UTC. A host that wants scripts to work in "its" zone passes the zone **name** in
as a value, which keeps the dependency visible.

Zone rules come from the host's IANA tzdata, which Kavun does not bundle. Go reads it in this order: the
`ZONEINFO` environment variable (a directory or an uncompressed `zoneinfo.zip`), the system copy
(`/usr/share/zoneinfo` on Unix), `$GOROOT/lib/time/zoneinfo.zip`, and finally the embedded copy if the program
imports `time/tzdata`. The `kavun` CLI does import it, so it works on minimal container images; a library does not
make that choice for its embedder — add `import _ "time/tzdata"` to your own `main` if your hosts may lack tzdata.

Zone rules change several times a year, so the **same script can answer a different wall clock — and so a
different date — on hosts with different tzdata versions**. UTC and fixed offsets are never affected. Two
consequences for a host:

- **Know which version you run.** In a container image the version is fixed when the image is built (the OS
  `tzdata` package; `cat /usr/share/zoneinfo/+VERSION` or the first line of `/usr/share/zoneinfo/tzdata.zi` shows
  it on most systems). A Go build's embedded copy is the version of that Go release.
- **To replay a past result exactly**, keep the `zoneinfo.zip` of each tzdata version you ran with, record the
  version label next to the stored result (the zip itself carries no version), and run the replay with
  `ZONEINFO=/path/to/zoneinfo-<version>.zip`. `ZONEINFO` is process-wide: replaying several versions side by side
  in one process is not supported.

**Sentinel instants.** Data from other systems often carries `0001-01-01` ("unset", .NET's `default(DateTime)`)
or `9999-12-31` ("no end"). Both are real instants inside `time`'s range (years 1…9999) and both are
**truthy** — only `time()`, the unix epoch, is falsy. If a sentinel means "missing", map it to `undefined` before
it reaches the script, so the script's `if t` and its maybe-missing defaults see what the data means.

## Memory Management

By default, VM reuse is lazy: stack and frame references are not fully cleared between runs. This improves performance
but keeps some references alive longer (until overwritten).

For more aggressive memory release when memory pressure is critical:

```go
if err := compiled.Run(machine); err != nil {
    panic(err)
}

// Optional: explicitly release remaining stack/frame references
machine.Clear()
```

Use `Clear()` when you prioritize releasing memory over peak throughput.

## Advanced Patterns

### One-Shot Execution

If you prefer a simpler one-shot flow without explicit resource management:

```go
func RunOnce(src []byte) error {
    script := kavun.NewScript(src)
    machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
    compiled, err := script.Compile()
    if err != nil {
        return err
    }
    return compiled.Run(machine)
}
```

This pattern is simpler but loses the benefits of reusing compiled code and VM state across multiple executions.

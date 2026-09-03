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

Control what scripts can import:

```go
// Selected stdlib modules
script.SetAllowedModules("math", "json")

// Enable/disable file imports
script.DisableFileImport()
script.EnableFileImport()

// In-memory source module
script.AddCustomModule("helpers", []byte(`
export add := func(a, b) { return a + b }
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
    reason, _ := re.Payload.Access(core.NewStringValue("reason"), bytecode.AccessIndex)
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

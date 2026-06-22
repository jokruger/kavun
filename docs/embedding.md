# Embedding Kavun In Go

The recommended embedding API is `kavun.Script`. It wraps parsing, compilation, globals setup, and VM execution into a
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
    "github.com/jokruger/kavun/core"
    "github.com/jokruger/kavun/stdlib"
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
    script := kavun.NewScript(src)
    script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
    script.Add("out", core.Undefined)

    // Create allocators and VM
    cta := core.NewArena(nil)      // Compile-time allocator
    rta := core.NewArena(nil)      // Runtime allocator
    machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

    // Compile once
    compiled, err := script.Compile(cta)
    if err != nil {
        panic(err)
    }

    // Run repeatedly with the same compiled code
    for i := 0; i < 100; i++ {
        if err := compiled.Run(machine); err != nil {
            panic(err)
        }
    }

    fmt.Println("result:", compiled.GetValue("out"))
}
```

## Allocators

Memory in Kavun is managed through two separate allocators:

- **Compile-time allocator** — used during parsing and compilation. Created once and persists for the lifetime of compiled code.
- **Runtime allocator** — used during script execution. Created for each run (or reused between runs) and reset by `Compiled.Run(...)`.

**Critical requirement**: you must use separate allocator instances for compile and runtime paths. Reusing the same
allocator can invalidate compile-time data when the runtime allocator resets.

```go
// Correct: separate allocators
cta := core.NewArena(nil)
rta := core.NewArena(nil)

compiled, err := script.Compile(cta)
if err != nil {
    panic(err)
}

if err := compiled.Run(machine); err != nil {
    panic(err)
}
```

If you pass `nil` for the compile-time allocator, Kavun creates a default one internally.

**How reuse works:**

- `Compiled.Run(machine)` resets the runtime allocator and reinitializes VM state before each execution.
- At lower-level, explicit reuse is done with `rta.Reset()` and `machine.Reset(rta, bytecode, globals)`.

## Inputs and Outputs

Pass data to scripts by setting globals before compilation:

```go
script.Add("x", core.IntValue(20))
script.Add("y", core.IntValue(22))
script.Add("out", core.Undefined)
```

Before each run, update input values with `compiled.Set(...)`:

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

After execution, retrieve output values with `Get`, `GetValue`, or `GetAll`:

```go
out := compiled.GetValue("out")
sum, _ := out.AsInt()
fmt.Println(sum)
```

API reference:

- `compiled.GetValue(name)` — returns a `core.Value`
- `compiled.Get(name)` — returns a `*kavun.Variable` wrapper
- `compiled.GetAll()` — returns all globals

## Modules and Imports

Control what scripts can import with a module map:

```go
modules := vm.NewModuleMap()

// Selected stdlib modules
modules.AddMap(stdlib.GetModuleMap("math", "json"))

// Host builtin module
modules.AddBuiltinModule("host", map[string]core.Value{
    "answer": core.IntValue(42),
})

// In-memory source module
modules.AddSourceModule("helpers", []byte(`
export add := func(a, b) { return a + b }
`))

script.SetImports(modules)
```

For general-purpose applications, use all stdlib modules:

```go
script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
```

### File Imports

File imports are disabled by default. Enable them explicitly:

```go
script.EnableFileImport(true)
if err := script.SetImportDir("./scripts"); err != nil {
    panic(err)
}
```

For custom file extensions, use the lower-level compiler API (`Compiler.SetImportFileExt`).

## Configuration and Limits

Configure common execution constraints:

```go
script.SetMaxConstObjects(10_000)
script.SetAssignmentMode(kavun.AssignmentModeSmart)
```

Available options:

- `script.SetMaxConstObjects(n)` — maximum number of constant objects
- `script.SetAssignmentMode(mode)` — assignment behavior mode

## Concurrency

`Script`, `Compiled`, `VM`, and allocators are **not thread-safe**. For parallel execution:

1. Each goroutine must use its own `Compiled` (via `Clone`)
2. Each goroutine must use its own runtime arena and VM
3. Protect shared resources with explicit locking

Safe pattern for parallel runs:

```go
base, err := script.Compile(core.NewArena(nil))
if err != nil {
    panic(err)
}

// Each goroutine clones the compiled code
clone, err := base.Clone(core.NewArena(nil))
if err != nil {
    panic(err)
}

// Each goroutine has isolated runtime resources
rta := core.NewArena(nil)
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
    rta := core.NewArena(nil)
    machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
    compiled, err := script.Compile(nil)
    if err != nil {
        return err
    }
    return compiled.Run(machine)
}
```

This pattern is simpler but loses the benefits of reusing compiled code and VM state across multiple executions.

### Custom Allocator Payload

Allocator behavior can be extended with a custom payload that follows the allocator lifecycle.

```go
type MyPayload struct {
    buf []byte
}

func (p *MyPayload) Reset() {
    p.buf = p.buf[:0]
}

opts := core.DefaultArenaOptions()
opts.Payload = &MyPayload{}

arena := core.NewArena(opts)
payload := arena.Payload() // retrieve custom payload when needed
```

The payload must implement `Reset()` and is reset together with the arena. This is useful for custom type registration
and type-specific allocation or caches (see unit tests for custom type registration patterns).

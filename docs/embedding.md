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

# Runtime Integration

GS scripts can be embedded inside host applications or executed through the CLI. The Go API in `script.go` matches what the unit tests in `tests/unit` exercise, so the same patterns apply regardless of where you run the VM.

## Running Scripts from a Host Application

```go
package main

import (
    "context"

    "github.com/jokruger/gs"
    "github.com/jokruger/gs/alloc"
    "github.com/jokruger/gs/stdlib"
)

func main() {
    a := alloc.NewHeapAllocator()
    script := gs.NewScript(a, []byte(`
        sum := 0
        each := func(seq, fn) { for x in seq { fn(x) } }
        each([1, 2, 3], func(x) { sum += x })
    `))
    script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

    compiled, err := script.RunContext(context.Background())
    if err != nil {
        panic(err)
    }
    sum := compiled.Get("sum").Value()
    _ = sum // use the result
}
```

Key APIs:

- `gs.NewScript(alloc, source)` – prepares a script with local variables.
- `script.Add(name, value)` / `script.Remove(name)` – manage inputs. Use the allocator to construct `core.Object` instances before injecting them.
- `script.SetImports(map)` – control which modules may be loaded. Use `stdlib.GetModuleMap(stdlib.AllModuleNames()...)` or hand-roll a whitelist.
- `script.EnableFileImport(bool)` and `script.SetImportDir(path)` – allow `import("./module")` in scripts. By default only builtin modules can be imported; file imports are disabled until you opt in.
- `script.SetMaxAllocs(n)` / `script.SetMaxConstObjects(n)` – enforce limits for untrusted code. Pass `-1` to remove the limit (the default).
- `script.Run()` compiles and runs the script, returning a `*gs.Compiled`.
- `script.Compile()` compiles a script without executing it. The resulting
  `*gs.Compiled` can be re-used across goroutines via `Clone()`.

## Working with `*Compiled`

A compiled script exposes helpers to read and write globals:

- `compiled.Run()` / `compiled.RunContext(ctx)` – re-run the bytecode.
- `compiled.Get(name)` – fetch the value of a global (returns `*gs.Variable`).
- `compiled.GetAll()` – snapshot of all globals.
- `compiled.Set(name, value)` – replace a global before re-running. Only names defined at compile time can be updated.
- `compiled.Clone()` – create a copy safe for concurrent use.

The VM enforces the maximum allocation count passed into the script (`SetMaxAllocs`) and returns descriptive errors when a limit is exceeded.

## Eval Helper

The `Eval` helper in `eval.go` compiles a single expression, runs it, and returns its value:

```go
res, err := gs.Eval(ctx, alloc.NewHeapAllocator(), `input ? "ok" : "fail"`,
    map[string]core.Object{"input": alloc.NewBool(true)})
```

It wraps the expression in `__res__ := (...)`, runs the script, and returns the value of `__res__`.

## Modules and Imports

`import("name")` looks up `name` in the module map you register. Builtin modules live in `stdlib/` and `stdlib.GetModuleMap` returns a map containing both builtin (Go-based) modules and source modules (defined in `.gs` files such as `stdlib/srcmod_enum.gs`). The VM copies each module into an immutable record and adds `__module_name__` so you can introspect where imports came from.

When you need to expose host functionality, create your own `vm.ModuleMap`, add builtin or source modules, and pass it to `script.SetImports`. The CLI uses this exact mechanism before executing scripts.

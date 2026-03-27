# Getting Started

This guide walks you from a blank checkout of [GS](https://github.com/jokruger/gs) to a runnable script and highlights the quickest way to try language features.

## Install the CLI

Use Go's toolchain to install the CLI directly from the repository:

```bash
go install github.com/jokruger/gs/cmd/gs@latest
```

If you prefer to work from a local clone:

```bash
git clone https://github.com/jokruger/gs
cd gs
# build the CLI that lives in cmd/gs
just build
cp cmd/gs/gs ~/bin/   # or any directory on your PATH
```

Note: repository is using [Just](https://github.com/casey/just) as a task runner. But you can also build the CLI with `go generate ./... ; go build -o gs ./cmd/gs` if you don't have Just installed.

Run `gs --help` to confirm the binary is on your `PATH`. The CLI exposes a couple of flags you will see referenced later:

- `-o <file>` – compile to a bytecode artifact instead of executing immediately.
- `-resolve` – resolve relative import paths. Tests exercise this flag and it will become the default once the new module loader stabilizes.
- `-version` – print the CLI build tag.

## First Script

Create a file named `hello.gs`:

```text
fmt := import("fmt")

greeting := {
    intro: "Hello",
    target: "GS",
    tags:  ["lambda", "record"]
}

fmt.println(
    greeting.intro + ", " + greeting.target + "!",
    string(greeting.tags.map(t => t.upper))
)
```

Execute it directly:

```bash
gs hello.gs
```

You can also compile once and keep the bytecode around for fast startups:

```bash
gs -o hello.bc hello.gs
gs ./hello.bc
```

Scripts support shebang execution as well. Add `#!/usr/bin/env gs` as the first line of your `.gs` file and `chmod +x` it to run it like any other executable.

## Working in the REPL

Running `gs` with no arguments launches an interactive Read–Eval–Print loop. It preloads the core builtins and the `println` helper that backs the unit tests, so it is perfect for experimenting with conversions, selectors, or lambda syntax before committing the code to a file.

## Imports and Modules

Modules are loaded with `import("name")`. The CLI registers every module from `stdlib/` by default. When embedding GS you control the surface area by calling `script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))` (see `reference/runtime.md`).

Imported modules are immutable records; this keeps shared state predictable:

```text
fmt := import("fmt")
fmt.println("ready")   # ✅
fmt.println = func() {} # Runtime error: module exports are immutable
```

When a module path is relative (for example `import("./utils")` inside a source file) enable the `-resolve` flag so the CLI loads the file relative to the importing script. This mirrors the behavior exercised in `tests/unit` when the compiler resolves source modules.

## Next Steps

- Take the [Language Tour](language-tour.md) for a deeper walkthrough of values, control flow, variadic calls, and modules.
- Review the [type system reference](../reference/type-system.md) when you need precise information on selectors, records, or immutability.
- Explore the [standard library catalog](../stdlib/README.md) to learn the snake_case function names exported by each module.
- When in doubt, inspect `tests/unit` in the repository—every documented feature has a corresponding test.

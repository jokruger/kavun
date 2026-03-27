# CLI

The `gs` executable compiles and executes `.gs` source files, runs the REPL, and can emit bytecode files for later use. The source lives under `cmd/gs` in the repository.

## Installation

Use the Go toolchain to install a binary directly from GitHub:

```bash
go install github.com/jokruger/gs/cmd/gs@latest
```

To build from a clone:

```bash
git clone https://github.com/jokruger/gs
cd gs
go build ./cmd/gs
cp cmd/gs/gs ~/bin/   # put it on your PATH
```

Run `gs --help` to confirm everything is wired correctly.

## Executing Source Files

```
gs path/to/script.gs
```

Use `-resolve` to enable relative import paths (module paths are resolved relative to the file that called `import`). The CLI automatically registers all modules from `stdlib/`, so `import("fmt")` works out of the box.

## Compiling to Bytecode

```
gs -o myscript.bc path/to/script.gs
```

Bytecode embeds the modules referenced at compile time, so keep the `-resolve` flag consistent between compilation and execution if you rely on local imports.

## Shebang Support

Adding a shebang to the top of a `.gs` file lets you execute it directly:

```text
#!/usr/bin/env gs
fmt := import("fmt")
fmt.println("Hello GS")
```

```
chmod +x hello.gs
./hello.gs
```

## REPL

Running `gs` without arguments starts the interactive Read–Eval–Print loop. Use it to inspect values (`len([])`), poke at new syntax, or test module calls.

## Flags

- `-o <file>` – write bytecode to `<file>` instead of executing the script.
- `-resolve` – resolve relative import paths during compilation and execution.
- `-version` – print the CLI build tag.
- `-help` – show usage information.

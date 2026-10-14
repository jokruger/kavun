# Installing

## Install the CLI with Go tooling

Requires Go 1.26 or later.

```sh
go install github.com/jokruger/kavun/cmd/kavun@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is on your `PATH`.

## Using the CLI

### Running a script

Pass a `.kvn` source file as the first argument:

```sh
kavun hello.kvn
```

### CLI flags

- `-o <file>`: write compiled bytecode to a file
- `-key <file>`: sign the compiled file (with `-o`), and refuse to run a compiled file whose signature does not
  verify with this key; default `$KAVUN_KEY_FILE`
- `-genkey -o <file>`: write a new random key file (never overwriting one)
- `-info <file>`: print a compiled file's versions, signature status, imports and bindings
- `-version`: print the CLI version
- `-strict-assign`: require variables to already exist for plain `=` assignment

By default, Kavun uses smart `=` assignment at compile time (`x = expr` declares `x` in current scope if unresolved).
Use `-strict-assign` to enforce explicit declaration before `=`.

```sh
kavun -strict-assign hello.kvn
```

### Compiled files and signing

`kavun -o app app.kvn` compiles to a bytecode file that `kavun app` runs. To be sure a compiled file is yours and
unchanged, sign it with a key:

```sh
kavun -genkey -o app.key          # one line of hex, readable by you only; keep it secret
kavun -key app.key -o app app.kvn # signed
kavun -key app.key app            # runs only if the signature verifies
```

`KAVUN_KEY_FILE=app.key` sets the key for every command instead of `-key`. A key is never given on the command line
itself. With a key, an unsigned file, a changed one or one signed with another key is refused; without one, every
compiled file runs as given (a signed file too — its signature is just not checked).

CLI keys are for development and tests: they exercise the signed path locally. In production the application
compiles and signs itself, with a key that never leaves it (`docs/embedding.md` § Where the key lives).

A compiled file also carries the bytecode version it was built for, and a newer engine may refuse it
(`incompatible bytecode: … recompile from source`). `kavun -info app` (with `-key` to check the signature) shows
why a file is refused instead of refusing:

```text
core version:  1.0 (this engine: 1.0)
host version:  0.0 (this process: 0.0)
signed:        yes, signature valid
imports:       io
bindings:      (none)
```

### Hashbang / shebang scripts

Add a hashbang line as the first line of your script to make it directly executable:

```go
#!/usr/bin/env kavun

io = import("io")
io.print("Hello Kavun!")
```

Then make the file executable and run it:

```sh
chmod +x hello.kvn
./hello.kvn
```

## Building from source

The project uses [just](https://github.com/casey/just) as its build tool. Install it before proceeding
(`brew install just` on macOS, or see the just documentation for other platforms).

Clone the repository and enter the project directory:

```sh
git clone https://github.com/jokruger/kavun.git
cd kavun
```

### Common recipes

| Command        | Description                                                          |
| -------------- | -------------------------------------------------------------------- |
| `just build`   | Generate sources and compile the `kavun` binary into `./build/kavun` |
| `just install` | Build and copy the binary to `$HOME/bin/`                            |
| `just test`    | Run the full test suite                                              |
| `just clean`   | Remove build artefacts and profiling files                           |

Run `just` with no arguments to list all available recipes.

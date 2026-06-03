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
- `-version`: print the CLI version
- `-strict-assign`: require variables to already exist for plain `=` assignment

By default, Kavun uses smart `=` assignment at compile time (`x = expr` declares `x` in current scope if unresolved).
Use `-strict-assign` to enforce explicit declaration before `=`.

```sh
kavun -strict-assign hello.kvn
```

### Hashbang / shebang scripts

Add a hashbang line as the first line of your script to make it directly executable:

```go
#!/usr/bin/env kavun

fmt = import("fmt")
fmt.println("Hello Kavun!")
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

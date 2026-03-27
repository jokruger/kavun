# GS (Go Script)

## Overview

GS is a lightweight, embeddable scripting language written in Go. It takes its roots from Tengo but focuses on a modernized syntax (records with selector access, arrow-function lambdas, lambda-based pipelines, variadic calls with spread syntax) and a VM that is easy to sandbox inside Go applications.

## Goals & Focus

- **Embeddable first** – every feature is designed to be surfaced through Go's APIs, so applications can run user scripts without escaping the Go toolchain.
- **Predictable data model** – records vs maps, immutable wrappers, and automatic conversions are all locked down by tests to keep behavior stable.
- **Modular standard library** – modules under `stdlib/` can be whitelisted or omitted entirely when sandboxing.
- **Ergonomic syntax** – lambda literals (`x => x * 2`), selector-based conversions (`value.string`), and `immutable(expr)` keep scripts readable.

## Key Language Features

- **Records & Maps** – `{}` literals build selector-friendly records. `map()` produces helper-rich maps (`map.keys`, `map.filter`, ...).
- **Automatic type conversion** – numeric expressions widen as needed, and all scalar values expose conversion selectors (`string.int`, `int.time`, etc.).
- **Immutable expressions** – wrap any array/map/record with `immutable(...)` to freeze the outer container without copying nested values.
- **First-class functions** – traditional `func` blocks, lambda literals, and variadic + spread syntax enable pipelines like `[1,2,3].map((i, x) => x + i).reduce(0, (acc, x) => acc + x)`.
- **Member-driven APIs** – every value (arrays, strings, bytes, maps, time, etc.) exposes properties/functions via selectors, which keeps scripts discoverable and self-documenting.

## Tooling

- **CLI (`cmd/gs`)** – compiles `.gs` files, runs scripts, emits bytecode with `-o`, resolves relative imports with `-resolve`, and exposes a REPL.
- **Host APIs** – `gs.NewScript` and `gs.Eval` let Go programs add variables, control module imports, set allocation limits, and execute bytecode safely.

## Standard Library

Modules live under `stdlib/` and include `base64`, `enum`, `fmt`, `hex`, `json`, `math`, `os`, `rand`, `text`, and `times`. Each module uses snake_case function names and returns immutable records so callers cannot mutate shared state. The module map is configurable, allowing embedders to whitelist functionality.

## Quick Start

```bash
go install github.com/jokruger/gs/cmd/gs@latest
```

```
#!/usr/bin/env gs

fmt := import("fmt")
fmt.println("Hello", "GS")
```

## Documentation

### Guide

- [Getting Started](docs/guide/getting-started.md)
- [Language Tour](docs/guide/language-tour.md)

### Reference

- [CLI](docs/reference/cli.md)
- [Runtime](docs/reference/runtime.md)
- [Type System](docs/reference/type-system.md)
- [Operators](docs/reference/operators.md)
- [Functions](docs/reference/functions.md)
- [Formatting](docs/reference/formatting.md)
- [Standard Library](docs/stdlib/README.md)


## Project Layout

The project keeps most tests under `tests/unit` instead of co-locating every `_test.go` file with the production code.

This is a deliberate choice. GS has a fairly broad runtime surface, and keeping tests in a dedicated tree makes larger behavior-oriented test cases easier to read, organize, and maintain. In practice, many tests exercise language and VM semantics across package boundaries, so grouping them by scenario is often clearer than scattering them throughout the source tree.

This is not the most idiomatic Go layout, and that tradeoff is intentional: for this project, readability and manageability take priority over strict colocation.

When adding or changing behavior, please add or update the relevant tests in `tests/unit`.

## Contributing

1. Fork the repository and clone your fork locally.
2. Make your changes in a focused branch.
3. Run the test suite.
4. Add or update tests in `tests/unit` for any change that affects language or runtime behavior.
5. Open a pull request describing the motivation for the change and any new or changed semantics.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Acknowledgements

This project is based on script language [Tengo](https://github.com/d5/tengo) by Daniel Kang. A special thanks to Tengo's creator and contributors.
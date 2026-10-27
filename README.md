[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/banner-direct-single.svg)](https://stand-with-ukraine.pp.ua)

# Kavun

![Kavun Logo](kavun-small.png)

Kavun (кавун, watermelon) is a lightweight, high-performance, embeddable scripting language for Go, built around
expression-oriented programming and consistent language design principles. Its feature set, including f-strings,
arrow-function lambdas, data-type member functions, and fluent chaining, enables transformation-heavy code to be written
as clear expressions instead of loop-and-branch boilerplate. It runs on a bytecode VM implemented in Go, making
embedding and sandboxing straightforward in Go services and tools.

## Key Features

- **Dynamically typed**, with coercive equality (`1 == "1"`) and a small, predictable truthiness table.
- **Expression-oriented** — arrow lambdas (`x => x * 2`), fluent method chaining, `if`/`for` init statements
  sharing scope with their block.
- **Destructuring & parallel assignment** — `a, b = b, a`, `a, b, c = [1, 2, 3]`, `a, b = {a: 1, b: 2}`.
- **Placeholder syntax (`_`)** — `add(1, _, 3)` is sugar for `x => add(1, x, 3)`; see the
  [cheat sheet](docs/cheatsheet.md#placeholder-syntax-_).
- **f-strings & runtime format templates** — `f"n={n:5d}"`, plus `format(template, args)` for the same
  `{...}`/format-spec syntax at runtime.
- **`decimal` as a first-class type** — exact arithmetic for money, not a float workaround.
- **`date` and a strict `time`** — civil days (`d"2026-01-31"`) next to zone-aware instants; text is parsed by
  one strict grammar or an explicit layout, never guessed.
- **`fin` module** — loan payments, NPV/IRR, depreciation and rate conversion as exact decimals at a stated
  rounding (backed by [fin128](https://github.com/jokruger/fin128)).
- **`defer`/`recover`** — Go-style cleanup and error handling without Go panics on the hot path.
- **Non-mutating by default** — collection methods return new values; `_in_place` and `freeze()` /
  `freeze_shallow()` are the explicit opt-ins.
- **Deterministic, single-threaded, sandboxable** — no goroutines/channels exposed to scripts, so embedding in
  finance, decisioning, and game-logic hosts stays reproducible and auditable.
- **AST-level optimizer** — purity-driven constant folding and dead-code elimination (see
  [purity contract](docs/purity.md)).

## Quick Start

Install the cli with Go's toolchain:

```bash
go install github.com/jokruger/kavun/cmd/kavun@latest
```

Or download a prebuilt binary from the [latest release](https://github.com/jokruger/kavun/releases/latest):

Then you can run Kavun scripts with the `kavun` command or using hashbang:

```go
#!/usr/bin/env kavun

io = import("io")

result = [1, 2, 3, 4, 5, 6]
  .keep(x => x % 2 == 0)
  .map(x => x * x)
  .reduce(0, (sum, x) => sum + x)

io.print(f"sum of even squares: {result}")
```

See more [examples](docs/examples.md), or the [cheat sheet](docs/cheatsheet.md) for a one-page syntax reference.

## Benchmark Results

Full benchmark results are available in the
[Kavun Benchmarks report](https://github.com/jokruger/kavun-benchmark/blob/main/results/REPORT.md).
A summary is shown below:

| Rank | Engine | CPU geomean | Avg rank | Worst ratio | Wins | Mem geomean | Tasks run | Missing |
|------|--------|-------------|----------|-------------|------|-------------|-----------|---------|
| 1 | kavun | 1.05× | 1.44 | 1.33× | 6 | 1.40× | 9 | 0 |
| 2 | gopherlua | 1.44× | 2.56 | 3.03× | 3 | 163.50× | 9 | 0 |
| 3 | golua | 1.51× | 3.33 | 1.85× | 0 | 228.44× | 9 | 0 |
| 4 | starlark | 2.32× | 4.11 | 5.56× | 0 | 158.82× | 9 | 0 |
| 5 | tengo | 2.94× | 4.33 | 44.34× | 0 | 1177.80× | 9 | 0 |
| 6 | goja | 4.87× | 6.22 | 11.91× | 0 | 297.35× | 9 | 0 |
| 7 | risor | 5.82× | 6.00 | 136.10× | 0 | 3102.55× | 9 | 0 |

## Documentation

- [Installing](docs/installing.md) - Instructions for installing the Kavun CLI.
- [Embedding](docs/embedding.md) - Guide to embedding the Kavun runtime in Go applications.
- [Language Reference](docs/language.md) - Syntax, expressions, statements, functions, modules, built-ins, and diagnostics.
- [Cheat Sheet](docs/cheatsheet.md) - One-page quick reference for syntax and key language features.
- [Type Reference](docs/types.md) - The type families and the language-wide rules: argument dispatch, conversions,
  member/operator correspondence, callbacks, mutability, truthiness.
- [Reference](docs/reference/builtins.md) - Generated from the source: one page per [type](docs/reference/types/array.md)
  (`docs/reference/types/`), per [module](docs/reference/stdlib.md), the [builtin functions](docs/reference/builtins.md)
  and the [function matrix](docs/reference/function_matrix.md).
- [Standard Library](docs/stdlib.md) - General notes on the builtin modules: the module type, the grant, the failure policy. The per-module reference is [docs/reference/stdlib.md](docs/reference/stdlib.md).
- [Examples](docs/examples.md) - Short, runnable snippets showcasing key language features.
- [Virtual Machine](docs/vm.md) - Virtual machine specifics and limitations.
- [Coding Conventions](docs/conventions.md) - Guidelines for code style and contributions, and how a native function
  is declared (`//kavun:` directives, `go generate ./...`).
- [Extending Types](docs/extending_types.md) - Adding a host type: operators, members, callables.

## Contributing

Before contributing, please review [`docs/conventions.md`](docs/conventions.md) for project layout, coding standards and
repository contracts.

1. Fork the repository and clone your fork locally.
2. Make your changes in a focused branch.
3. Run `go generate ./...` after changing a native function's doc comment or directive, then the test suite.
4. Add or update tests (end-to-end cases in `kavun_test.go`) for any change that affects language or runtime
   behavior.
5. Open a pull request describing the motivation for the change and any new or changed semantics.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Acknowledgements

This project is based on script language Tengo by Daniel Kang. A special thanks to Tengo's creator and contributors.

See the `NOTICE` file for the full third-party license texts, and `LICENSE` for this project's.
# Kavun

![Kavun Logo](kavun-small.png)

Kavun (кавун, watermelon) is a lightweight, high-performance, embeddable scripting language for Go, built around
expression-oriented programming and consistent language design principles. Its feature set, including f-strings,
arrow-function lambdas, data-type member functions, and fluent chaining, enables transformation-heavy code to be written
as clear expressions instead of loop-and-branch boilerplate. It runs on a bytecode VM implemented in Go, making
embedding and sandboxing straightforward in Go services and tools.

## Quick Start

Install the cli with Go's toolchain:

```bash
go install github.com/jokruger/kavun/cmd/kavun@latest
```

Or download a prebuilt binary from the [latest release](https://github.com/jokruger/kavun/releases/latest):

Then you can run Kavun scripts with the `kavun` command or using hashbang:

```go
#!/usr/bin/env kavun

fmt = import("fmt")

result = [1, 2, 3, 4, 5, 6]
  .filter(x => x % 2 == 0)
  .map(x => x * x)
  .reduce(0, (sum, x) => sum + x)

fmt.println(f"sum of even squares: {result}")
```

See more [examples](docs/examples.md).

## Benchmark Results

Full benchmark results are available in the
[Kavun Benchmarks report](https://github.com/jokruger/kavun-benchmark/blob/main/results/REPORT.md).
A summary is shown below:

| Rank | Engine | CPU geomean | Avg rank | Worst ratio | Wins | Mem geomean | Tasks run | Missing |
|------|--------|-------------|----------|-------------|------|-------------|-----------|---------|
| 1 | kavun | 1.01× | 1.11 | 1.05× | 8 | 1.04× | 9 | 0 |
| 2 | gopherlua | 1.59× | 2.78 | 4.87× | 1 | 180.03× | 9 | 0 |
| 3 | golua | 1.61× | 3.33 | 2.35× | 0 | 251.53× | 9 | 0 |
| 4 | starlark | 2.55× | 4.22 | 5.17× | 0 | 174.88× | 9 | 0 |
| 5 | tengo | 3.15× | 4.33 | 69.56× | 0 | 1296.87× | 9 | 0 |
| 6 | goja | 5.16× | 6.22 | 10.85× | 0 | 327.41× | 9 | 0 |
| 7 | risor | 6.32× | 6.00 | 230.43× | 0 | 3416.20× | 9 | 0 |

## Documentation

- [Installing](docs/installing.md) - Instructions for installing the Kavun CLI.
- [Embedding](docs/embedding.md) - Guide to embedding the Kavun runtime in Go applications.
- [Language Reference](docs/language.md) - Syntax, expressions, statements, functions, modules, built-ins, and diagnostics.
- [Type Reference](docs/types.md) - Detailed builtin type semantics, conversions, and member functions.
- [Standard Library](docs/stdlib.md) - Overview of standard library modules and their APIs.
- [Examples](docs/examples.md) - Short, runnable snippets showcasing key language features.
- [Virtual Machine](docs/vm.md) - Virtual machine specifics and limitations.
- [Coding Conventions](docs/conventions.md) - Guidelines for code style and contributions.

## Contributing

Before contributing, please review [`docs/conventions.md`](docs/conventions.md) for project layout, coding standards and
repository contracts.

1. Fork the repository and clone your fork locally.
2. Make your changes in a focused branch.
3. Run the test suite.
4. Add or update tests in `tests/unit` for any change that affects language or runtime behavior.
5. Open a pull request describing the motivation for the change and any new or changed semantics.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Acknowledgements

This project is based on script language Tengo by Daniel Kang. A special thanks to Tengo's creator and contributors.

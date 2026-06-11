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

| Rank | Engine    | CPU geomean | Avg rank | Worst ratio | Wins | Mem geomean | Tasks run | Missing |
| ---- | --------- | ----------- | -------- | ----------- | ---- | ----------- | --------- | ------- |
| 1    | kavun0    | 1.03×       | 1.44     | 1.16×       | 5    | 1.20×       | 9         | 0       |
| 2    | kavun     | 1.09×       | 1.78     | 1.25×       | 3    | 1.04×       | 9         | 0       |
| 3    | gopherlua | 1.63×       | 3.78     | 3.98×       | 1    | 208.63×     | 9         | 0       |
| 4    | golua     | 1.68×       | 4.33     | 2.40×       | 0    | 291.49×     | 9         | 0       |
| 5    | starlark  | 2.62×       | 5.11     | 5.15×       | 0    | 202.66×     | 9         | 0       |
| 6    | tengo     | 3.30×       | 5.33     | 59.98×      | 0    | 1502.91×    | 9         | 0       |
| 7    | goja      | 5.35×       | 7.22     | 11.08×      | 0    | 379.43×     | 9         | 0       |
| 8    | risor     | 6.55×       | 7.00     | 180.74×     | 0    | 3958.94×    | 9         | 0       |

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

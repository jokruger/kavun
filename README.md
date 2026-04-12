# GS (Go Script)

GS (Go Script) is a lightweight, high-performance, embeddable scripting language for Go, built around expression-oriented programming and consistent language design principles. Its feature set, including arrow-function lambdas, data-type member functions, and fluent chaining, enables transformation-heavy code to be written as clear expressions instead of loop-and-branch boilerplate. It runs on a bytecode VM implemented in Go, making embedding and sandboxing straightforward in Go services and tools.

## Quick Start

Install the cli with Go's toolchain:

```bash
go install github.com/jokruger/gs/cmd/gs@latest
```

Then you can run GS scripts with the `gs` command or using hashbang:

```go
#!/usr/bin/env gs

fmt := import("fmt")
fmt.println("Hello", "GS")
```

## Documentation

TODO

## Contributing

Before contributing, please review [`docs/project.md`](docs/project.md) and [`docs/conventions.md`](docs/conventions.md) for project layout, coding standards and repository contracts.

1. Fork the repository and clone your fork locally.
2. Make your changes in a focused branch.
3. Run the test suite.
4. Add or update tests in `tests/unit` for any change that affects language or runtime behavior.
5. Open a pull request describing the motivation for the change and any new or changed semantics.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

### Acknowledgements

This project is based on script language Tengo by Daniel Kang. A special thanks to Tengo's creator and contributors.
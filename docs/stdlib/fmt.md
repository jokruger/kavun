# Module `fmt`

```text
fmt := import("fmt")
```

## Functions

- `print(args...)`: writes the string representation of each argument to stdout without separators.
- `println(args...)`: writes the string representation of each argument to stdout with space separators, followed by a newline.
- `printf(format, args...)`: writes a formatted string (no trailing newline). See [`reference/formatting.md`](../reference/formatting.md) for verb details.
- `sprintf(format, args...) => string`: returns a formatted string (alias of the
  `format()` builtin with convenience syntax).

# Module `enum`

This is a source module (defined in `stdlib/srcmod_enum.gs`) that ships with the CLI and exposes helpers for enumerable values.

```text
enum := import("enum")
```

## Functions

- `all(x, fn) => bool`: returns true when `fn` evaluates to a truthy value for every item.
- `any(x, fn) => bool`: returns true when `fn` succeeds for at least one item.
- `chunk(x, size) => [array]`: splits an array into groups of `size` (last chunk may be smaller).
- `at(x, key) => object`: fetches an item by index (arrays) or key (maps).
- `each(x, fn)`: iterates over `x` calling `fn(key, value)` for each element.
- `filter(x, fn) => [object]`: collects items for which `fn` returns truthy.
- `find(x, fn) => object/undefined`: returns the first value passing `fn`.
- `find_key(x, fn) => int/string/undefined`: returns the key or index of the first matching element.
- `map(x, fn) => [object]`: transforms each item using `fn(key, value)`.
- `key(k, _) => object`: identity helper that returns the first argument.
- `value(_, v) => object`: identity helper that returns the second argument.

All helpers return `undefined` when the provided value is not enumerable.

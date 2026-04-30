# immutable wrappers

Immutable container wrappers.

## Overview

The `immutable(x)` function wraps a container (array, bytes, dict, record, or runes) to make it immutable at the
container level. Attempting to modify an immutable container raises a runtime error. Individual immutable containers can
be identified via their type name (e.g., `"immutable-array"`).

## Creating Immutable Containers

### Using the `immutable()` Function

```go
a = immutable([1, 2, 3])
r = immutable({x: 10})
```

## Behavior

### Read Operations Work Normally

```go
a = immutable([1, 2, 3])
value = a[0]        // 1 (read works)
len = a.len()       // 3 (read works)
```

### Write Operations Fail

```go
a = immutable([1, 2, 3])
a[0] = 99           // runtime error - immutable
a[3] = 4            // runtime error - immutable
```

### Type Name

Immutable containers have their type names prefixed with `"immutable-"`:

```go
type_name(immutable([1, 2, 3]))         // "immutable-array"
type_name(immutable({a: 1}))            // "immutable-record"
type_name(immutable(dict({a: 1})))      // "immutable-dict"
```

## Creating Mutable Copies

The `copy()` function always returns a mutable deep copy, even from an immutable value:

```go
original = immutable([1, 2, 3])
mutable_copy = copy(original)

mutable_copy[0] = 99   // Success - copy is mutable
println(original[0])   // 1 (original unchanged)
```

## Notes

- Immutability applies to the container level, not to nested values
- If nested values are mutable types (arrays, dicts), they can be modified
- For complete deep immutability, ensure nested values are also wrapped
- `copy()` always produces a mutable result regardless of source mutability
- Immutable containers still support all read operations efficiently

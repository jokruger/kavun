# undefined

Represents the absence of a value.

## Overview

The `undefined` value is used to represent the absence of a meaningful value. It's returned in situations where:

- A field or index doesn't exist
- A conversion fails (unless a fallback is provided)
- Operations attempt to access non-existent resources

## Behavior

### Field and Index Access

Any field or index access on `undefined` returns `undefined`:

```go
u = undefined
u.anything        // undefined
u[0]              // undefined
u.deeply.nested   // undefined
```

### Truthiness

`undefined` is falsy in boolean contexts:

```go
if undefined {
    // This block is NOT executed
}

undefined && true   // false
undefined || false  // false
```

### Conversion Fallbacks

Many conversion builtins return `undefined` on conversion failure unless a fallback is provided:

```go
int("not a number")           // undefined
int("not a number", 0)        // 0 (uses fallback)

float("invalid")              // undefined
float("invalid", 3.14)        // 3.14 (uses fallback)
```

## Member Functions

### General Functions

#### `copy()`

Returns `undefined`.

**Arguments:** None

**Returns:** `undefined`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `undefined` is immutable, this method
returns the receiver unchanged.

```go
undefined.copy()    // undefined
```

#### `format(spec)`

Renders the value as a string using the [Format Mini-Language](../format-mini-language.md).

**Arguments:**

- `spec` (`string`, required) - format mini-language spec. Pass `""` for the default rendering.

**Returns:** `string`

**Description:** Equivalent to using the value as the operand of an f-string interpolation, e.g.
`f"{x:<spec>}"` - except the spec is parsed on each call rather than at compile time. With no argument or with an empty
string the type's default rendering is returned. The set of accepted verbs and modifiers is type-specific;
see [Format Mini-Language](../format-mini-language.md) for the full grammar.

```go
undefined.format()         // "undefined"
```

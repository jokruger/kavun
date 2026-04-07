# Conventions

## Coding Conventions

This section outlines coding conventions for Go code, including VM specific guidelines and general best practices.

### Variadic Arguments: Immutability Contract

Functions that accept variadic arguments (`...Value`) or slice of arguments (`[]Value`) must **never mutate** the arguments slice or its elements. This is both a Go best practice and a critical requirement for performance in this VM.

To avoid allocations, the VM passes stack slices directly to callees. The full capacity of these slices extends to the end of the stack array. If a callee appends to `args`, it corrupts subsequent stack frames.

Functions should not have side effects on caller state beyond their explicit return values. Mutating arguments violates this principle.

## GS Language Conventions

This section defines conventions for naming, behavior, and design choices affecting the GS language itself.

### Properties vs Member Functions

For builtin scalar/container types, use **member functions for all operations**, including zero-argument operations.

- Use `len()` instead of `len`.
- Use `sum()` instead of `sum`.
- Use `min()` instead of `min`.
- Use `is_empty()` instead of `empty`.

Rationale:

- One mental model: `name(...)` always means "evaluate behavior now".
- Future-proofing: zero-arg APIs can later accept optional parameters without breaking style.
- Better chain consistency: methods compose predictably.
- Avoids ambiguity around computed properties vs stored fields.

Reserved use of properties:

- Properties are allowed for plain data objects/records/modules that expose stored fields.
- Builtin type capabilities should remain method-based.

### Naming Style

- Use `snake_case` for all member APIs.
- Use short, concrete verbs/nouns.
- Avoid abbreviations unless universally recognized (`len`, `min`, `max`, `avg`).
- Use verb-based names for transformations and aggregations, noun-based names for queries.

Examples:

- Transformations: `map`, `filter`, `sort`, `reverse`, `upper`, `lower`, `trim`
- Aggregations: `reduce`, `sum`, `avg`, `min`, `max`, `count`
- Queries: `len`, `is_empty`, `is_sorted`, `contains`

### Predicate Prefixes (`is_`, `has_`, `can_`)

Boolean-returning methods must use explicit predicate prefixes.

- State checks: `is_empty()`, `is_sorted()`, `is_zero()`
- Ownership/content checks: `has_prefix()`, `has_key()`
- Capability checks: `can_parse_int()`

Do not expose bare adjectives/nouns for booleans (`empty()`, `sorted()`, `zero()`).

### Conversion Prefixes (`to_`, `as_`)

Use `to_*` for explicit type conversion methods.

- `to_int()`, `to_float()`, `to_string()`, `to_bytes()`

Avoid `as_*` for general conversions to keep rules simple and unambiguous.

If both strict and non-strict forms are needed, use `to_*` plus `try_to_*`.

- `to_int()` -> strict conversion (fails on invalid value)
- `try_to_int()` -> non-throwing conversion (returns `(value, ok)` or `null`)

### Mutating vs Non-mutating Methods

Default convention: methods are non-mutating and return new values where relevant.

- `sort()` returns a sorted copy.
- `upper()` returns a new string.

If an in-place variant is required, mark it explicitly with `_in_place`.

- `sort_in_place()`
- `reverse_in_place()`

Never provide two methods with the same base name where one mutates and one does not.

### Arity and Optional Arguments

- Keep zero-arg methods zero-arg when semantically clear: `sum()`, `len()`, `is_empty()`.
- Add optional behavior through explicit overload-like alternatives instead of hidden behavior switches where possible.

Examples:

- `sort()` and `sort_by(fn)` instead of a single polymorphic method with many argument shapes.
- `join(sep)` instead of `join()` with implicit separators.

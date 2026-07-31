# Conventions

## Coding Conventions

This section outlines coding conventions for Go code, including VM specific guidelines and general best practices.

### Variadic Arguments: Immutability Contract

Functions that accept variadic arguments (`...Value`) or slice of arguments (`[]Value`) must **never mutate** the
arguments slice or its elements. This is both a Go best practice and a critical requirement for performance in this VM.

To avoid allocations, the VM passes stack slices directly to callees. The full capacity of these slices extends to the
end of the stack array. If a callee appends to `args`, it corrupts subsequent stack frames.

Functions should not have side effects on caller state beyond their explicit return values. Mutating arguments violates
this principle.

## Kavun Language Conventions

This section defines conventions for naming, behavior, and design choices affecting the Kavun language itself.

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

### Mutating vs Non-mutating Methods

Default convention: methods are non-mutating and return new values where relevant.

- `sort()` returns a sorted copy.
- `upper()` returns a new string.

If an in-place variant is required, mark it explicitly with `_in_place`.

- `sort_in_place()`
- `reverse_in_place()`

Never provide two methods with the same base name where one mutates and one does not.

This convention is a direct consequence of the [Purity Contract](purity.md): operators and methods must be pure by
default so the AST optimizer can fold constant subexpressions safely. Impure operations should be exposed as
top-level builtin functions (registered with `Pure = false`); `_in_place` methods are the last-resort escape hatch.

### Range/Slice Bounds: Inclusive-Start, Exclusive-Stop

`range()`, three-part slicing (`a[start:stop:step]`), and the `..`/`..:` range literal (`low..high[:step]`, see
[Range literals](language.md#range-literals)) all use the same bound convention: the start is inclusive, the stop is
exclusive. `1..5` and `range(1, 5)` both mean 1, 2, 3, 4 — never 5.

This was a deliberate choice, not just "that's what `range()` already did." The half-open convention (inclusive
start, exclusive stop) is the stronger engineering default independent of which language you're coming from — see
Dijkstra's ["Why numbering should start at zero"](https://www.cs.utexas.edu/users/EWD/transcriptions/EWD08xx/EWD831.html)
for the canonical argument:

- **Composes cleanly**: `a..b` followed by `b..c` covers `a..c` with no gap or overlap. An inclusive `a..b` would
  need `a..(b-1)` then `b..c` to compose the same way.
- **Empty ranges need no sentinel**: `a..a` is naturally empty. Inclusive semantics have no way to spell "empty
  starting at `a`" without something like `a..(a-1)`.
- **Length is just subtraction**: `stop - start`, with no `+1` anywhere to get wrong.
- **No boundary overflow**: an inclusive upper bound at a type's max value has no valid "one past the end" to fall
  back to (this is exactly why Rust needed a dedicated `..=` operator alongside plain `..`).

It also keeps `..` honest as *pure sugar*: it compiles straight through to a call to the `range` builtin (see
[compiler/compiler_impl.go](../compiler/compiler_impl.go) `compileSliceExpr`), so it must mean exactly what
`range()` means. If `..` were inclusive while `range()` stayed exclusive, the sugar would have to translate
`a..b` into `range(a, b+1)` — which only makes sense for integers and breaks the moment `range()` grows support for
another type with no well-defined "successor" (a float, a decimal, a string).

### Arity and Optional Arguments

- Keep zero-arg methods zero-arg when semantically clear: `sum()`, `len()`, `is_empty()`.
- Add optional behavior through explicit overload-like alternatives instead of hidden behavior switches where possible.

Examples:

- `sort()` and `sort_by(fn)` instead of a single polymorphic method with many argument shapes.
- `join(sep)` instead of `join()` with implicit separators.

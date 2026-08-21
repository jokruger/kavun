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

### Value model: scalars vs. containers

Every `core.Value` is a fixed-size header — `Type`, `Immutable`, a `Data` word, and a `Ptr` (`core/value.go`).
Assigning or passing a `Value` always copies this header only, never anything it may point to. For types that
store their whole payload in `Data` (`int`, `float`, `bool`, `decimal`, ...), that header copy is a full,
independent copy of the value. For types that store payload behind `Ptr` (`array`, `dict`, `record`, `bytes`,
`runes`, and also `string`), the header copy shares the same backing storage as the original — mutating through
one binding is visible through every other binding that shares that pointer.

`string` is the one type in the second group (`Ptr`-backed) that scripts still perceive as scalar, and that's
not an accident of implementation — it's what makes the user-facing model correct: `string` has no mutating
operation at all, so its shared backing bytes are never observable as shared. Whether a type is "scalar" or
"container" from a script author's point of view is determined entirely by whether it has a mutating operation,
not by whether its `Value` is `Data`-inline or `Ptr`-backed internally.

**Constraint for anyone adding a new builtin type:** if it's `Ptr`-backed (heap storage, shared on
assignment/passing — the normal case for anything non-trivial), it must never gain a mutating operation unless
it's meant to behave as a container in the user-facing model. If a new type should read as scalar to script
authors — copied, never surprising, no shared-body behavior — it must never be given an `_in_place` method or
any other body-mutating hook, regardless of whether it happens to be `Ptr`-backed internally (see `string`).

### Implementing operators (`BinaryOp`/`UnaryOp`/`Equal`)

See [Extending types: operators](extending-types.md) for the full contract — the three-rule dispatch model
(domain-specific, narrow safe-conversion, reflected delegate) for `BinaryOp`/`UnaryOp`, the simpler one-hop
delegate model for `Equal` (`final` flag, no error return — `==`/`!=` are total operators), and the one hard
cross-cutting rule shared by both (never wildcard-match `undefined`/`error`). Same weight as the Purity Contract
above: enforced by code review, not by any automated check, and required of any new builtin or embedder
(`SetValueType`) type that implements operators at all.

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

### Mutating vs Non-mutating Functions (Kavun-language, user-defined)

This is a **requirement**, not a recommendation: a Kavun function must not mutate any of its arguments' shared
body unless its own name carries the `_in_place` suffix. This extends the member-method convention above from
builtin types up to whole functions written in Kavun — since containers share their body across assignment and
argument-passing by design, a caller can only trust what a function does to what they passed by reading its
name, not its implementation.

```go
// Wrong: mutates the caller's array with no signal at the call site.
func normalize(items) {
    items.sort_in_place()
    return items
}

// Right: the name tells the caller items will be mutated.
func normalize_in_place(items) {
    items.sort_in_place()
    return items
}
```

A function that invokes a caller-supplied callback on its own arguments is held to the same rule: unless it can
guarantee the callback never mutates, it must be named `_in_place` too — a callback's behavior isn't known
until runtime, so it can't be assumed safe by default.

**Current status:** this rule is enforced today only by code review and this documented contract — the compiler
and VM do not yet verify it. Host-configurable, compiler-enforced checking (mirroring `Script.SetAssignmentMode`,
`script.go:90`) is planned but requires a sound alias-tracing/interprocedural static analysis (parameter alias
tracking through a function body, propagated across calls to other user-defined functions, with any
runtime-supplied callback treated as unprovable/mutating by default) that hasn't been designed yet. Until that
lands, follow this convention as a hard rule, not a "usually" — a violation won't be caught for you.

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

It also keeps `..` honest as *pure sugar*: `compileRangeExpr` (`compiler/compiler_impl.go`) compiles it to a call to
the exact same underlying `range` builtin as writing `range(...)` by hand, so it must mean exactly what `range()`
means. If `..` were inclusive while `range()` stayed exclusive, the sugar would have to translate `a..b` into
`range(a, b+1)` — which only makes sense for integers and breaks the moment `range()` grows support for another type
with no well-defined "successor" (a float, a decimal, a string).

Note this is "the same call" in the sense of always invoking `range`'s one true implementation, not "the same
identifier reference": `low..high` is a language construct, immune to a local `range := ...` reassignment, the same
way a `b"..."` bytes literal is immune to reassigning `byte` — see
[Shadowing and reassigning builtins](language.md#shadowing-and-reassigning-builtins).

### Arity and Optional Arguments

- Keep zero-arg methods zero-arg when semantically clear: `sum()`, `len()`, `is_empty()`.
- Add optional behavior through explicit overload-like alternatives instead of hidden behavior switches where possible.

Examples:

- `sort()` and `sort_by(fn)` instead of a single polymorphic method with many argument shapes.
- `join(sep)` instead of `join()` with implicit separators.

# container semantics

Shared semantics of array-like containers (`array`, `bytes`, `runes`, `dict`, `record`): reference behavior,
immutability, propagation through derived operations, and the aliasing pitfalls of `append_in_place`.

## Reference Semantics

Containers are reference-typed. Assignment shares the underlying buffer; mutation through one variable is visible
through any other variable that refers to the same container:

```go
fmt = import("fmt")
a = [1, 2, 3]
b = a
b[0] = 99
fmt.println(a)              // [99, 2, 3] - a sees b's mutation
```

Use `copy()` to obtain an independent value:

```go
fmt = import("fmt")
a = [1, 2, 3]
b = copy(a)
b[0] = 99
fmt.println(a)              // [1, 2, 3] - unchanged
```

## Immutable Wrappers

The `immutable(x)` function wraps a container (`array`, `bytes`, `dict`, `record`, `runes`) to make it immutable at the
container level. Attempting to modify an immutable container raises a runtime error. Individual immutable containers
can be identified via their type name (e.g., `"immutable-array"`).

### Creating Immutable Containers

```go
a = immutable([1, 2, 3])
r = immutable({x: 10})
```

### Read Operations Work Normally

```go
a = immutable([1, 2, 3])
value = a[0]            // 1 (read works)
len = a.len()           // 3 (read works)
```

### Write Operations Fail

```go
a = immutable([1, 2, 3])
a[0] = 99               // runtime error - immutable
a[3] = 4                // runtime error - immutable
```

### Type Name

Immutable containers have their type names prefixed with `"immutable-"`:

```go
type_name(immutable([1, 2, 3]))         // "immutable-array"
type_name(immutable({a: 1}))            // "immutable-record"
type_name(immutable(dict({a: 1})))      // "immutable-dict"
type_name(immutable(bytes("ab")))       // "immutable-bytes"
type_name(immutable(runes("ab")))       // "immutable-runes"
```

### Creating Mutable Copies

The `copy()` function always returns a mutable deep copy, even from an immutable value:

```go
fmt = import("fmt")
original = immutable([1, 2, 3])
mutable_copy = copy(original)

mutable_copy[0] = 99    // Success - copy is mutable
fmt.println(original[0])    // 1 (original unchanged)
```

## Propagation Through Slicing and Chunking

Two-part slicing (`v[a:b]`), stepped slicing (`v[a:b:s]`), and `chunk(n)` all return a fresh, independent buffer —
mutating the result never affects the source, and vice versa. Because the result is always a fresh, independently-owned
value, it is always mutable, regardless of the source's mutability — the same convention `copy()` already follows:

```go
a = immutable([1, 2, 3, 4])
type_name(a[1:3])          // "array"  (independent copy, always mutable)
type_name(a[::-1])         // "array"  (independent copy, always mutable)
type_name(a.chunk(2)[0])   // "array"  (independent copy, always mutable)
type_name(copy(a))         // "array"  (same convention)
```

For the explicit opt-in that shares backing storage instead — trading this safety for performance in cases where
you've confirmed nothing else needs the original — see the next section.

## Slicing and Chunking Views

`slice_view(start, end)` and `chunk_view(size)` are the named twins that share backing storage with the source,
exactly like `v[a:b]`/`chunk(n)` used to before they were flipped to always copy. Use `.is_view()` to check whether
a given `array`/`bytes`/`runes` value shares storage with something else:

```go
a = [1, 2, 3, 4, 5]
v = a.slice_view(1, 3)
v.is_view()                 // true
a[1:3].is_view()            // false - the safe default is a real copy, never a view
```

A view also inherits the source's immutability, since it's a handle onto the same body, not an independent value:

```go
a = immutable([1, 2, 3, 4])
type_name(a.slice_view(1, 3))    // "immutable-array"
```

### The danger: silent effect on source/sibling views

Because a view shares the same backing array as its source, mutating through the view is visible through the source
— and through any other view/slice still pointing at the same array — and vice versa:

```go
a = [1, 2, 3, 4, 5]
b = a.slice_view(1, 3)      // b shares a's backing array
b[0] = 99
a[1]                        // 99 - mutated through b's view
```

This is exactly the aliasing model containers already have on plain assignment (see
[Reference Semantics](#reference-semantics)) — a view is just another handle onto the same body, not a special case.

### Capacity-dependent reallocation

`append()` is always a fresh, independent copy (see [Append](#append) below), so growing a view with plain `append()`
is always safe — it never touches the view's or source's backing array, regardless of spare capacity:

```go
a = [1, 2, 3, 4, 5]
v = a.slice_view(0, 2)      // v = [1, 2]
v2 = v.append(99)           // v2 = [1, 2, 99] - independent copy; a and v are untouched
```

`append_in_place()` is the operation that carries the danger: it mutates the view's own backing struct directly, and
a view's storage is a re-slice of the source's backing array, which may have spare capacity beyond the view's own
bounds. Growing a view with `append_in_place()` can silently write into memory the source (or a sibling view) still
considers its own data, or it may reallocate instead — the outcome is implementation-defined and must not be relied
upon, exactly like the [Append In Place Aliasing](#append-in-place-aliasing) pitfall below, but reachable through a
view without ever calling `append_in_place` directly on the source:

```go
a = [1, 2, 3, 4, 5]
v = a.slice_view(0, 2)      // v = [1, 2], but its backing array still has spare capacity from a
v.append_in_place(99)       // silently overwrites a[2] here - implementation-defined, not guaranteed
a[2]                        // 99 - the source was corrupted through the view
```

Never grow a view with `append_in_place()` unless you've confirmed no other reference needs the memory beyond the
view's own bounds.

### The amortized-growth idiom

Because `append()` always copies, growing a container with it in a loop is O(n²), not amortized O(n) — every
iteration copies everything seen so far:

```go
x = []
for i := 0; i < n; i = i + 1 {
    x = x.append(i)   // correct, but O(n²): a fresh full copy every iteration
}
```

For amortized O(n) growth, use `append_in_place()` instead. This is safe in the classic "grow in a loop" shape
because the loop always reassigns to (or otherwise only ever reads through) the same variable, so there's only ever
one live handle to the (possibly relocated) buffer at any point in the loop — true whether or not a view was ever
involved upstream:

```go
x = []
for i := 0; i < n; i = i + 1 {
    x.append_in_place(i)   // safe and amortized O(n): x is the only handle to its buffer at each step
}
```

The unsafe pattern is holding onto a *view*, or any other second live reference, while separately mutating or growing
through a different reference to the same source — that's the scenario the two sections above warn about, not this
loop shape. (A future optimizer pass may auto-rewrite the safe `x = x.append(i)` loop shape into `append_in_place`
when it can prove no other alias exists, but this is not implemented today; write `append_in_place()` explicitly
where the O(n²) cost of plain `append()` matters.)

### Interaction with `freeze()` / `freeze_shallow()`

`freeze()` always detaches first (it's `copy()` plus deep immutability), so a frozen value is never affected by what
later happens to a view derived from its source, and freezing a view doesn't affect the source either:

```go
a = [1, 2, 3]
v = a.slice_view(0, 2)
f = v.freeze()               // f is an independent, fully-detached, deep-immutable clone
a[0] = 99
f                            // [1, 2] - unaffected, even though v itself would have shown 99
```

`freeze_shallow()` does **not** detach — freezing a view in place only flips that view's own header. The source (and
any other view into the same body) is untouched and can still mutate the shared backing array, which remains
observable through the "frozen" view, since freezing never protected the shared body in the first place:

```go
a = [1, 2, 3]
v = a.slice_view(0, 2)
v = v.freeze_shallow()      // v's own header is now immutable
a[0] = 99
v[0]                         // 99 - the shared body changed; v was never protected from it
```

## Append

`append(...)` always returns a fresh, independent container with the given items added — the same convention
`copy()`/`slice()`/`chunk()` already follow. It never touches the receiver's backing storage, works regardless of
the receiver's mutability, and calling it with zero items is a legal no-op that still returns an independent copy
rather than the receiver itself:

```go
fmt = import("fmt")
x = [1, 2, 3]
v1 = x.append(100)      // v1 = [1, 2, 3, 100]
v2 = x.append(200)      // v2 = [1, 2, 3, 200]
fmt.println(x, v1, v2)  // [1, 2, 3] [1, 2, 3, 100] [1, 2, 3, 200] - independent, no aliasing between any of them
```

There is no capacity-dependent hazard here at all — this holds for `array`, `bytes`, and `runes` alike. The
trade-off is performance: `append()` always does a full copy, so growing a container with it in a loop is O(n²) (see
[The amortized-growth idiom](#the-amortized-growth-idiom) above). For amortized O(n) growth, use `append_in_place()`
instead — the explicit mutating twin, covered next.

## Append In Place Aliasing

`append_in_place(...)` mutates the receiver's own shared body directly and returns that same receiver — not a copy,
the identical object. This means **every** existing alias into the receiver sees the mutation, deterministically,
with no "maybe" about it (unlike a Go slice's own `append`, which this operation's underlying implementation still
uses — see below):

```go
fmt = import("fmt")
x = [1, 2, 3]
b = x                        // b shares x's body (plain assignment, see Reference Semantics above)
x.append_in_place(4)
fmt.println(x, b)            // [1, 2, 3, 4] [1, 2, 3, 4] - b sees it too, no reassignment needed
```

### The one place the outcome is still implementation-defined: a second, independent container

The receiver-sharing above is always guaranteed. What's *not* guaranteed is whether growing one independently-owned
container ever silently reaches into memory a completely unrelated container happens to still be holding, when the
two once shared a common ancestor — the classic case being two views derived from the same source (see
[Capacity-dependent reallocation](#capacity-dependent-reallocation) above) rather than two ordinary containers with
no shared history, which never have this problem regardless of capacity.

### How to Get Predictable Amortized Growth Without Aliasing

If you need independent, amortized-growable containers derived from the same starting point, `copy()` each one
first, so none of them ever shares an ancestor with another live handle:

```go
x = [1, 2, 3]
v1 = copy(x).append_in_place(100)   // v1 is independent of x
v2 = copy(x).append_in_place(200)   // v2 is independent of x and v1
```

The same rules apply to `bytes` and `runes`. `append_in_place` on an immutable container is rejected at runtime, so
the aliasing pitfall only applies to mutable receivers.

## Dict/Record Conversion Views

`dict` and `record` are the one pair of types that already share the same underlying representation
(`map[string]Value`), so converting between them can be a real zero-copy operation — unlike converting from
`array`/`bytes`/`runes`/`string`/`range`, which always builds a brand-new map (there's no shared representation
to reuse in those directions).

- `dict_val.record()` / `dict(record_val)` — the safe default: an independent **shallow** copy. A fresh
  top-level container (its own key set, independent of the source), but nested values are shared, not
  recursively cloned — same convention as `copy_shallow()`. Always returns a mutable result, regardless of the
  source's mutability.
- `dict_val.record_view()` / `dict_view(record_val)` / `record(dict_val)` / `record_view(dict_val)` — the
  `_view` twins share the source's underlying map directly, both directions. Both the top-level key set *and*
  nested values are the exact same backing storage, so mutating either wrapper's keys or nested values is
  visible through the other:

```go
d = dict({a: [1, 2]})
r = d.record_view()
r.b = 99          // adds "b" to the SAME map d uses
d["b"]            // 99 - visible through d too
r.a[0] = 42
d["a"][0]         // 42 - nested value shared as well
```

  The copying form (`record()`/`dict()`) only shares nested values, not the key set — adding or removing a key
  on one side never affects the other:

```go
d = dict({a: [1, 2]})
r = d.record()
r.b = 99
d["b"]            // undefined - r's key set is independent
r.a[0] = 42
d["a"][0]         // 42 - nested value still shared (shallow copy)
```

- `record` has no member functions at all (no `MethodCall` switch — every `record_val.foo(...)` call is
  dispatched as "call a stored field," not a builtin operation), so `record_val.dict()`/`record_val.dict_view()`
  don't exist and never will unless that's resolved separately. `dict(record_val)`/`dict_view(record_val)` are
  the only spellings for the record-to-dict direction.
- A view's mutability is inherited from its source (immutable source → immutable view); the copying form is
  always mutable, same as `copy()`/`copy_shallow()`.

## Notes

- Immutability applies to the container level, not to nested values.
- If nested values are mutable types (arrays, dicts), they can still be modified through any reference to them.
- For complete deep immutability, ensure nested values are also wrapped.
- `copy()` always produces a mutable result regardless of source mutability.
- Immutable containers still support all read operations efficiently.

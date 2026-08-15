# container semantics

Shared semantics of array-like containers (`array`, `bytes`, `runes`, `dict`, `record`): reference behavior,
immutability, propagation through derived operations, and the aliasing pitfalls of `append`.

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

A view's own storage is a re-slice of the source's backing array, which may have spare capacity beyond the view's own
bounds. Appending to a view can silently write into memory the source (or a sibling view) still considers its own
data, or it may reallocate instead — the outcome is implementation-defined and must not be relied upon, exactly like
the [Append Aliasing](#append-aliasing) pitfall below, but reachable through a view without ever calling `append`
directly on the source:

```go
a = [1, 2, 3, 4, 5]
v = a.slice_view(0, 2)      // v = [1, 2], but its backing array still has spare capacity from a
v2 = v.append(99)           // may silently overwrite a[2], or may reallocate - implementation-defined
```

Never grow a view (`append`, or any future `append_in_place`) unless you've confirmed no other reference needs the
memory beyond the view's own bounds.

### The safe amortized-loop idiom

The classic "grow in a loop" pattern stays safe because it always reassigns to the same variable, so there's only
ever one live handle to the (possibly relocated) buffer at any point in the loop — true whether or not a view was
ever involved upstream:

```go
x = []
for i := 0; i < n; i = i + 1 {
    x = x.append(i)   // safe: x is the only handle to its buffer at each step
}
```

The unsafe pattern is holding onto a *view* while separately mutating or growing through a different reference to the
same source — that's the scenario the two sections above warn about, not this loop shape.

### Interaction with `freeze()` / `freeze_in_place()`

`freeze()` always detaches first (it's `copy()` plus deep immutability), so a frozen value is never affected by what
later happens to a view derived from its source, and freezing a view doesn't affect the source either:

```go
a = [1, 2, 3]
v = a.slice_view(0, 2)
f = v.freeze()               // f is an independent, fully-detached, deep-immutable clone
a[0] = 99
f                            // [1, 2] - unaffected, even though v itself would have shown 99
```

`freeze_in_place()` does **not** detach — freezing a view in place only flips that view's own header. The source (and
any other view into the same body) is untouched and can still mutate the shared backing array, which remains
observable through the "frozen" view, since freezing never protected the shared body in the first place:

```go
a = [1, 2, 3]
v = a.slice_view(0, 2)
v = v.freeze_in_place()      // v's own header is now immutable
a[0] = 99
v[0]                         // 99 - the shared body changed; v was never protected from it
```

## Append Aliasing

`append` returns a value that **may or may not** share its backing buffer with the source, depending on the source's
spare capacity. This means assigning the result of `append` to a variable other than the source produces unpredictable
behavior.

### The Safe Pattern

Always assign the result of `append` back to the source variable:

```go
x = [1, 2, 3]
x = x.append(4)         // safe - reassign to x
x = x.append(5, 6)      // safe
```

In this pattern, the source variable is the only handle to the (possibly relocated) buffer, so aliasing cannot cause
surprises.

### The Unsafe Pattern

Storing `append`'s result in a different variable while keeping the source around exposes implementation-defined
aliasing:

```go
fmt = import("fmt")
x = [1, 2, 3]
v1 = x.append(100)      // v1 = [1, 2, 3, 100]
v2 = x.append(200)      // v2 = [1, 2, 3, 200]
fmt.println(v1)         // ??? could be [1, 2, 3, 100] OR [1, 2, 3, 200]
```

The outcome depends on the hidden capacity of `x`'s backing buffer:

- If `x` has spare capacity, both appends write into the same memory at the same offset. `v2`'s write **overwrites**
  the slot `v1` exposes, so `v1` and `v2` end up equal — both showing `200`.
- If `x` has no spare capacity, the first append allocates a new buffer for `v1`; the second append again allocates,
  producing an independent `v2`. `v1` keeps `100`.

You cannot rely on either outcome — the capacity is an internal detail that may change with the size of the container,
the runtime allocator, or future versions of Kavun.

### How to Get Predictable Behavior

If you need an independent extended container without disturbing the source, copy first:

```go
x = [1, 2, 3]
v1 = copy(x).append(100)   // v1 is independent of x
v2 = copy(x).append(200)   // v2 is independent of x and v1
```

The same rules apply to `bytes` and `runes`. `append` on an immutable container is rejected at runtime, so the aliasing
pitfall only applies to mutable sources.

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

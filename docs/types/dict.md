# dict

Dictionary/map type with string keys and values of any type.

## Overview

The `dict` type is similar to a `record` but only supports index access for elements; selector access is reserved for
dict member functions. Use `dict` when you need to perform operations on the dictionary itself (filtering, querying
keys/values, etc.).

**Key Characteristic:** Dicts use index notation (`d["key"]`) for element access and selector notation (`d.method()`)
for operations.

## Declaration and Usage

### Construction

```go
d = dict({a: 1, b: 2})
d2 = dict({})              // empty dict
```

### Index Access for Elements

All element access uses index notation:

```go
d = dict({a: 1, b: 2})
d["a"]       // 1
d["b"]       // 2
d["missing"] // undefined (non-existent keys)
```

### Adding and Modifying Elements

```go
d = dict({a: 1})
d["b"] = 2           // Add new element
d["a"] = 10          // Modify existing element
```

### Selector Access NOT Allowed for Elements

Attempting to use selector notation for element access raises an error:

```go
d = dict({a: 1})
d.a          // runtime error - dot access not allowed for elements
```

### Selector Access for Member Functions

Selector notation is used for calling member functions:

```go
d = dict({a: 1, b: 2})
d.keys()     // array of keys
d.len()      // number of elements
```

### Reference Semantics

```go
fmt = import("fmt")

d1 = dict({a: 1})
d2 = d1

d1["a"] = 10
fmt.println(d2["a"])   // 10 (both point to same dict)

d3 = copy(d1)      // Independent copy
d1["a"] = 1
fmt.println(d3["a"])   // 10 (d3 is unchanged)
```

## Record and Dict Relationship

Records and dicts represent the same underlying structure and can reference the same data:

```go
fmt = import("fmt")

r = {a: 1, b: 2}
d = dict(r)

// They point to the same data
r.a = 10
fmt.println(d["a"])   // 10 (both reflect the change)
```

## Operators

`+` merges two dicts (or a dict with a record — see below), and `-` removes a key non-mutatingly.

```go
fmt = import("fmt")

fmt.println(dict({a: 1, b: 2}) + dict({b: 99, c: 3}))   // dict({"a": 1, "b": 99, "c": 3})
```

Key collisions are resolved right-hand-side-wins (last-writer-wins, the same rule as Python's `{**a, **b}` or a
JS object spread), regardless of which literal is written first. `dict` also merges with `record` in either
direction, and the result is always `dict` the moment either operand is a `dict` (only `record + record` stays
`record` — see [record](record.md#operators)):

```go
fmt.println(dict({a: 1}) + record({b: 2}))    // dict({"a": 1, "b": 2})
fmt.println(record({b: 2}) + dict({a: 1}))    // dict({"a": 1, "b": 2})
```

`dict - "key"` returns a new dict without that key, leaving the original untouched — the operator-form
equivalent of [`.delete("key")`](#deletekey) below (in fact implemented via the same non-mutating path). Removing
a key that doesn't exist is a no-op, not an error. Only a `string` right-hand side is accepted; anything else is
a runtime error, and there is no reflected form (`"key" - dict` is not defined):

```go
fmt.println(dict({a: 1, b: 2}) - "a")      // dict({"b": 2})
fmt.println(dict({a: 1, b: 2}) - "zzz")    // dict({"a": 1, "b": 2}) -- missing key, no-op
```

### Equality and ordering

`==`/`!=` compare structurally — same keys, same values (recursively, via each value's own equality) — and cross
`dict`/`record` freely, the same way `+` does:

```go
fmt.println(dict({a: 1, b: 2}) == dict({b: 2, a: 1}))   // true -- key order never matters
fmt.println(dict({a: 1}) == record({a: 1}))             // true -- dict/record compare across each other
fmt.println(dict({a: 1}) == dict({a: 1, b: 2}))         // false
```

There is **no ordering** (`< > <= >=`) between two dicts, or between a `dict` and a `record` — this is a
deliberate omission, not an oversight. Unlike a `string` or `array`, a dict has no natural total order: any
scheme based on comparing keys/values lexicographically would be arbitrary, which is exactly why Python 3
removed dict ordering entirely (Python 2 had one, and it was widely considered a design mistake). The one
ordering-shaped question that *would* have principled meaning — "does every key/value in `a` also appear in
`b`?", i.e. subset/superset — is a genuinely different, *partial* order (two dicts can simply be incomparable,
neither a subset nor a superset of each other), unlike every other use of `<` in Kavun, which always means "a
total order, or a decidable runtime error." Overloading `<` with that different a meaning for one type pair
isn't planned; if a containment check is ever added, it belongs as an explicit method (e.g.
`a.is_subset_of(b)`), not an operator — see `TODO.md`.

## Member Functions

### General Functions

#### `copy()`

Returns a deep, mutable copy of the dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Equivalent to the builtin `copy(x)`. The result is an independent value; mutations to the copy do not
affect the original. When called on an `immutable-dict`, the returned copy is mutable. See
[container semantics](container-semantics.md) for details.

```go
d = dict({a: 1, b: 2})
c = d.copy()
c["a"] = 99
// d is still dict({a: 1, b: 2}), c is dict({a: 99, b: 2})
```

#### `copy_shallow()`

Returns a shallow, mutable copy of the dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Clones only the top-level dict — a fresh, independently-owned key set — but nested values are
shared with the source, not recursively cloned. Use this when you only need to stop the outer dict from
growing/aliasing the original and don't need nested content protected too; use `copy()` for a fully independent
deep clone. When called on an `immutable-dict`, the returned copy is mutable.

```go
d = dict({a: [1, 2]})
c = d.copy_shallow()
c["b"] = 99         // top level is independent
c["a"][0] = 88       // but nested values are still shared
d["a"][0]            // 88 - d sees the nested mutation
```

#### `freeze()`

Returns a fully independent, deep-immutable copy of the dict.

**Arguments:** None

**Returns:** `dict` (immutable)

**Description:** Equivalent to `copy()` followed by recursively marking the fresh clone (and everything nested
inside it) immutable. Always detaches first, so the source and every existing alias into it are completely
unaffected.

```go
d = dict({a: [1, 2]})
f = d.freeze()
is_immutable(f)       // true
d["a"][0] = 99
f["a"][0]              // 1 - unaffected
```

#### `freeze_shallow()`

Marks the dict's own header immutable without detaching.

**Arguments:** None

**Returns:** `dict` (immutable)

**Description:** Genuinely pure — never mutates anything reachable, just returns a new header with the
immutable flag set, pointing at the *same* shared body. Requires reassignment to affect your own variable
(`d = d.freeze_shallow()`), and a pre-existing sibling binding into the same body stays independently mutable
and can still change what the "frozen" variable sees. See
[container semantics](container-semantics.md#interaction-with-freeze-freeze_shallow) for the full contract.

```go
d = dict({a: 1})
d = d.freeze_shallow()
is_immutable(d)    // true
```

#### `format([spec])`

Renders the value as a string using the [Format Mini-Language](../format-mini-language.md).

**Arguments:**

- `spec` (optional, `string`) - format mini-language spec. Defaults to `""`.

**Returns:** `string`

**Description:** Equivalent to using the value as the operand of an f-string interpolation, e.g.
`f"{x:<spec>}"` - except the spec is parsed on each call rather than at compile time. With no argument or with an empty
string the type's default rendering is returned. The set of accepted verbs and modifiers is type-specific;
see [Format Mini-Language](../format-mini-language.md) for the full grammar.

```go
dict({a: 1}).format()        // 'dict({"a": 1})'
```

### Conversion Functions

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Returns the dict as a record (allowing field access via dot notation), as an independent
shallow copy — a fresh top-level record, but nested values are shared with the source, not recursively cloned
(same convention as `copy_shallow()`). For the explicit performance opt-in that shares the source's underlying
storage directly instead (both top level and nested), see `record_view()`. See
[container semantics](container-semantics.md) for the full copy-vs-view contract.

```go
fmt = import("fmt")
d = dict({name: "Alice"})
r = d.record()
fmt.println(r.name)   // "Alice"
```

#### `record_view()`

Converts to record, sharing the source's storage.

**Arguments:** None

**Returns:** `record` (shares storage with the source)

**Description:** The explicit sharing twin of `record()` — unlike `record()`'s independent top level, this
shares the source dict's underlying storage directly, both top level *and* nested: assigning a new field
through the returned record, or mutating a nested value, is visible through the source dict too (and vice
versa). Maximum performance when you don't need independence, at the cost of the usual view aliasing hazards —
see [container semantics](container-semantics.md#dict-record-conversion-views).

```go
fmt = import("fmt")
d = dict({name: "Alice", nested: [1, 2]})
r = d.record_view()
r.nested[0] = 99
fmt.println(d)          // dict({"name": "Alice", "nested": [99, 2]})
r.name = "Bob"
fmt.println(d)          // dict({"name": "Bob", "nested": [99, 2]}) - top-level change shared too
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Returns the same dict value.

```go
dict({a: 1}).dict()    // dict({a: 1})
```

### Query and Accessor Functions

#### `is_empty()`

Checks if dict is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the dict has no keys.

```go
dict({}).is_empty()        // true
dict({a: 1}).is_empty()    // false
```

#### `len()`

Gets number of keys.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of key-value pairs.

```go
dict({a: 1, b: 2, c: 3}).len()    // 3
dict({}).len()                     // 0
```

#### `keys()`

Gets array of keys.

**Arguments:** None

**Returns:** `array`

**Description:** Returns an array of all keys (unsorted).

```go
d = dict({a: 1, b: 2})
d.keys()           // array with "a" and "b" (order not guaranteed)
```

#### `values()`

Gets array of values.

**Arguments:** None

**Returns:** `array`

**Description:** Returns an array of all values. Order is not guaranteed.

```go
d = dict({a: 1, b: 2})
d.values()         // array with 1 and 2
```

#### `contains(x)`

Checks if dict contains key.

**Arguments:**

- `x` (string): Key to search for

**Returns:** `bool`

**Description:** Returns `true` if the key exists.

```go
d = dict({a: 1, b: 2})
d.contains("a")    // true
d.contains("c")    // false
```

#### `delete(key)`

Returns a new dict with a key removed.

**Arguments:**

- `key` (string): Key to remove.

**Returns:** `dict`

**Description:** Pure — never mutates the receiver, works regardless of the receiver's mutability. Removing a
key that doesn't exist is a no-op that still returns an independent copy. For the mutating twin, see
`delete_in_place()`.

```go
d = dict({a: 1, b: 2})
c = d.delete("a")
c    // dict({"b": 2})
d    // dict({"a": 1, "b": 2}) - source untouched
```

#### `delete_in_place(key)`

Removes a key from the dict in place.

**Arguments:**

- `key` (string): Key to remove.

**Returns:** `dict` (the receiver)

**Description:** Mutates the receiver's own shared map directly — visible through every existing alias without
reassignment. Rejects an immutable receiver.

```go
d = dict({a: 1, b: 2})
c = d                      // c shares d's body
d.delete_in_place("a")
d    // dict({"b": 2})
c    // dict({"b": 2}) - c sees it too
immutable(dict({a: 1})).delete_in_place("a")   // Error: not_deletable
```

### Filtering and Predicate Functions

#### `filter(fn)` / `filter()`

Filters by predicate, or filters out entries with `undefined` values when called without arguments.

**Arguments:**

- `fn` (function, optional): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`. When
  omitted, all entries whose value is `undefined` are removed.

**Returns:** `dict`

**Description:** Returns a new dict with only key-value pairs where the predicate returns `true`. If called with no
arguments, returns a new dict with all entries whose value is `undefined` removed.

```go
d = dict({a: 1, b: 2, c: 3, d: 4})

// Filter by value > 2
filtered = d.filter((k, v) => v > 2)  // dict({c: 3, d: 4})

// Filter by key name
filtered = d.filter((k, v) => k != "a")  // dict({b: 2, c: 3, d: 4})

// Drop entries with undefined values
e = dict({a: 1, b: undefined, c: 3})
e.filter()                            // dict({a: 1, c: 3})
```

#### `for_each(fn)`

Executes a callback for each key-value pair.

**Arguments:**

- `fn` (function): Callback function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `undefined`

**Description:** Calls `fn` for each pair and ignores callback results except for control flow. Iteration stops when
`fn` returns falsy value. Iteration order is not guaranteed.

```go
total = 0
d.for_each((k, v) => {
    total += v
    return true
})
```

#### `count(fn)`

Counts pairs matching predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `int`

**Description:** Returns the number of key-value pairs where the predicate returns `true`.

```go
d = dict({a: 1, b: 2, c: 3})
d.count((k, v) => v > 1)    // 2 (b: 2, c: 3)
```

#### `all(fn)`

Tests if all pairs match predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `bool`

**Description:** Returns `true` if all key-value pairs satisfy the predicate.

```go
d = dict({a: 2, b: 4, c: 6})
d.all((k, v) => v % 2 == 0)    // true (all even)

d = dict({a: 1, b: 2, c: 3})
d.all((k, v) => v > 2)         // false
```

#### `any(fn)`

Tests if any pair matches predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `bool`

**Description:** Returns `true` if any key-value pair satisfies the predicate.

```go
d = dict({a: 1, b: 2, c: 3})
d.any((k, v) => v > 2)      // true (c: 3)

d = dict({a: 1, b: 1})
d.any((k, v) => v > 2)      // false
```

#### `find(fn)`

Finds key of first pair matching predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `string` or `undefined`

**Description:** Returns the key of the first key-value pair for which the predicate returns `true`. Iteration stops on
the first match. Returns `undefined` if no pair matches. Iteration order is unspecified, so for dicts with multiple
matches the returned key may vary between runs.

```go
d = dict({a: 1, b: 2, c: 3})
d.find(k => k == "b")        // "b"
d.find(k => k == "q")        // undefined
d.find((k, v) => v == 2)     // "b"
d.find((k, v) => v == 99)    // undefined
```

## Examples

### Working with Configuration

```go
fmt = import("fmt")

// Store and query configuration
config = dict({
    debug: false,
    timeout: 30,
    port: 8080,
    host: "localhost"
})

fmt.println("Server running on " + config["host"] + ":" + config["port"].string())

// Check if keys exist
if config.contains("ssl_cert") {
    fmt.println("SSL configured")
} else {
    fmt.println("No SSL configuration")
}
```

### Filtering Data

```go
fmt = import("fmt")

// Filter dictionary by criteria
users = dict({
    alice: {age: 25, active: true},
    bob: {age: 17, active: false},
    carol: {age: 30, active: false}
})

// Adults only
adults = users.filter((name, user) => user.age >= 18)
fmt.println("Adults:", adults)

// Active users
active = users.filter((name, user) => user.active)
fmt.println("Active:", active)
```

### Aggregation

```go
fmt = import("fmt")

// Calculate statistics
scores = dict({
    alice: 85,
    bob: 92,
    carol: 78,
    dave: 95
})

// Count high scorers
high_scores = scores.count((name, score) => score >= 90)
fmt.println("High scores (>= 90):", high_scores)

// Check if all passed (>= 70)
all_passed = scores.all((name, score) => score >= 70)
fmt.println("All passed:", all_passed)
```

### Key-Value Iteration

```go
fmt = import("fmt")

// Iterate through dict
cache = dict({user_1: "Alice", user_2: "Bob", user_3: "Carol"})

for key in cache.keys() {
    value = cache[key]
    fmt.println(key, "=>", value)
}
```

## Comparison with Record

| Feature          | Dict                          | Record                              |
| ---------------- | ----------------------------- | ----------------------------------- |
| Element Access   | Index only (`d["key"]`)       | Index and dot (`r["key"]`, `r.key`) |
| Member Functions | Many available                | None                                |
| Iteration        | Use `.keys()` and `.values()` | Must convert to dict                |
| Use Case         | Maps, queries, operations     | Object/data representation          |

Choose `dict` when you need to manipulate/query the collection, or `record` for simple data representation.

## Notes

- Dict keys are always strings
- Dict values can be any type (including nested dicts/records)
- Dicts are reference-typed (use `copy()` for independent copies)
- All operations on elements must use index notation, never dot notation
- Member functions use dot notation exclusively

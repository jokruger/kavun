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

**Description:** Returns the dict as a record, allowing field access via dot notation.

```go
fmt = import("fmt")
d = dict({name: "Alice"})
r = d.record()
fmt.println(r.name)   // "Alice"
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

### Filtering and Predicate Functions

#### `filter(fn)`

Filters by predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument `(key)` or two arguments `(key, value)`.

**Returns:** `dict`

**Description:** Returns a new dict with only key-value pairs where the predicate returns `true`.

```go
d = dict({a: 1, b: 2, c: 3, d: 4})

// Filter by value > 2
filtered = d.filter((k, v) => v > 2)  // dict({c: 3, d: 4})

// Filter by key name
filtered = d.filter((k, v) => k != "a")  // dict({b: 2, c: 3, d: 4})
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

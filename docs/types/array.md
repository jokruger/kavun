# array

Mutable collections of heterogeneous values.

## Overview

Arrays are ordered, mutable collections that can hold values of any type. Arrays are reference-typed, meaning `a = b`
makes both variables point to the same array; to get an independent copy, use `copy()`.

## Declaration and Usage

### Array Literals

```go
a = [1, 2, 3]
b = ["hello", "world"]
c = [1, "two", 3.0, true]  // mixed types
empty = []
```

### Construction

```go
// From other types
from_range = range(1, 4).array()      // [1, 2, 3]
from_string = "abc".array()           // [97, 98, 99]
```

### Reference Semantics

```go
a = [1, 2, 3]
b = a
a[0] = 99
println(b[0])    // 99 (both point to same array)

c = copy(a)      // Independent copy
a[0] = 1
println(c[0])    // 99 (c is unchanged)
```

### Indexing and Slicing

```go
a = [10, 20, 30, 40, 50]
a[0]             // 10
a[2]             // 30
a[0:2]           // [10, 20]
a[2:4]           // [30, 40]
a[-1]            // 50 (last element)
a[:-1]           // [10, 20, 30, 40]
a[-3:-1]         // [30, 40]
a[4:2]           // []
a[1:5:2]         // [20, 40]
a[5:1:-1]        // [50, 40, 30, 20]
a[::-1]          // [50, 40, 30, 20, 10]
```

Single-element indexing supports negative indices. Two-part slice bounds follow the same rules: negative bounds count
from the end, omitted bounds default to the natural edge, oversized bounds clamp, and an inverted slice returns an empty
result. Arrays also support three-part slices `start:end:step`; `step` may be negative (reverse traversal) but cannot be
zero. Out-of-bounds index access raises `index out of bounds`.

### Mutation

```go
a = [1, 2, 3]
a[0] = 99        // Change element
a[5] = 100       // Extend array (fills with undefined for indices 3-4)
```

## Member Functions

### Conversion Functions

#### `array()`

Converts to array.

**Arguments:** None

**Returns:** `array`

**Description:** Returns the same array.

```go
[1, 2, 3].array()    // [1, 2, 3]
```

#### `bytes()`

Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Converts array elements to bytes (elements must be 0-255).

```go
[72, 101, 108, 108, 111].bytes()  // bytes("Hello")
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts elements to runes and builds a string from them.

```go
[72, 101, 108, 108, 111].string()  // "Hello"
```

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts array to record where keys are string indices (`"0"`, `"1"`, ...).

```go
[48, 49, -1].record()   // {"0": 48, "1": 49, "2": -1}
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts array to dict where keys are string indices (`"0"`, `"1"`, ...).

```go
[48, 49, -1].dict()     // dict({"0": 48, "1": 49, "2": -1})
```

### Transformation and Filtering Functions

#### `sort()`

Sorts array elements.

**Arguments:** None

**Returns:** `array`

**Description:** Sorts the array in ascending order. Elements must be comparable.

```go
[3, 1, 4, 1, 5].sort()         // [1, 1, 3, 4, 5]
["c", "a", "b"].sort()         // ["a", "b", "c"]
```

#### `reverse()`

Reverses the array.

**Arguments:** None

**Returns:** `array`

**Description:** Returns a new array with elements in reverse order.

```go
[].reverse()                   // []
[1, 2, 3].reverse()            // [3, 2, 1]
["a", "b", "c"].reverse()      // ["c", "b", "a"]
```

#### `chunk(size[, copy])`

Splits an array into arrays of up to `size` elements.

**Arguments:**

- `size` (int): Positive chunk size
- `copy` (bool, optional): When `true`, each chunk owns a copied backing array. Defaults to `false`.

**Returns:** `array`

**Description:** Returns an array of arrays. The final chunk contains the remaining elements when the length is not
evenly divisible by `size`. By default, chunks are reference slices of the original array for performance; pass `true`
as the second argument for independent chunk arrays.

```go
[1, 2, 3, 4, 5].chunk(2)       // [[1, 2], [3, 4], [5]]
[1, 2, 3].chunk(10)            // [[1, 2, 3]]
[1, 2, 3].chunk(2, true)       // copied chunks
```

#### `filter(fn)`

Filters by predicate.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument (value) or two (index, value).

**Returns:** `array`

**Description:** Returns a new array with only elements where the predicate returns `true`.

```go
[1, 2, 3, 4, 5].filter(x => x % 2 == 0)        // [2, 4]
[10, 20, 30].filter((i, v) => i > 0)           // [20, 30]
```

#### `map(fn)`

Transforms elements.

**Arguments:**

- `fn` (function): Transformation function. Accepts one argument (value) or two (index, value).

**Returns:** `array`

**Description:** Returns a new array with each element transformed by the function.

```go
[1, 2, 3].map(x => x * 2)                      // [2, 4, 6]
[1, 2, 3].map((i, v) => i * v)                 // [0, 2, 6]
```

#### `for_each(fn)`

Executes a callback for each element.

**Arguments:**

- `fn` (function): Callback function. Accepts one argument (value) or two (index, value).

**Returns:** `undefined`

**Description:** Calls `fn` for each element and ignores callback results except for control flow. Iteration stops when
`fn` returns falsy value.

```go
sum = 0
[1, 2, 3].for_each(v => {
    sum += v
    return true
})
```

### Predicate Functions

#### `all(fn)`

Tests if all elements match predicate.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `bool`

**Description:** Returns `true` if all elements satisfy the predicate.

```go
[2, 4, 6].all(x => x % 2 == 0)     // true
[1, 2, 3].all(x => x % 2 == 0)     // false
```

#### `any(fn)`

Tests if any element matches predicate.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `bool`

**Description:** Returns `true` if any element satisfies the predicate.

```go
[1, 3, 5].any(x => x % 2 == 0)     // false
[1, 2, 3].any(x => x % 2 == 0)     // true
```

#### `find(fn)`

Finds index of first matching element.

**Arguments:**

- `fn` (function): Predicate function. Accepts one argument (value) or two (index, value).

**Returns:** `int` or `undefined`

**Description:** Returns the index of the first element for which the predicate returns `true`. Iteration stops on the
first match. Returns `undefined` if no element matches.

```go
[10, 20, 30].find(x => x == 20)      // 1
[10, 20, 30].find(x => x == 99)      // undefined
[10, 20, 30].find((i, v) => i == 2)  // 2
```

#### `contains(x)`

Checks if array contains value.

**Arguments:**

- `x` (any): Value to search for

**Returns:** `bool`

**Description:** Returns `true` if the exact value is found.

```go
[1, 2, 3].contains(2)      // true
[1, 2, 3].contains(4)      // false
```

### Aggregation Functions

#### `count(fn)`

Counts elements matching predicate.

**Arguments:**

- `fn` (function): Predicate function

**Returns:** `int`

**Description:** Returns the number of elements where the predicate returns `true`.

```go
[1, 2, 3, 4, 5].count(x => x > 2)    // 3
[1, 2, 3].count(x => x % 2 == 0)     // 1
```

#### `reduce(init, fn)`

Reduces array to single value.

**Arguments:**

- `init` (any): Initial accumulator value
- `fn` (function): Reducer function. Accepts two arguments (accumulator, value).

**Returns:** `any`

**Description:** Iteratively applies the reducer function to produce a single value.

```go
[1, 2, 3].reduce(0, (acc, v) => acc + v)         // 6
[1, 2, 3].reduce(1, (acc, v) => acc * v)         // 6
["a", "b", "c"].reduce("", (acc, v) => acc + v)  // "abc"
```

#### `min()`

Finds minimum element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the smallest element. Returns `undefined` for empty array. Elements must be comparable.

```go
[3, 1, 4, 1, 5].min()    // 1
["c", "a", "b"].min()    // "a"
[].min()                 // undefined
```

#### `max()`

Finds maximum element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the largest element. Returns `undefined` for empty array. Elements must be comparable.

```go
[3, 1, 4, 1, 5].max()    // 5
["c", "a", "b"].max()    // "c"
[].max()                 // undefined
```

#### `sum()`

Sums numeric elements.

**Arguments:** None

**Returns:** `number`

**Description:** Returns the sum of all numeric elements (int, float, decimal).

```go
[1, 2, 3, 4, 5].sum()          // 15
[1.5, 2.5, 3.0].sum()          // 7.0
[decimal(1), decimal(2)].sum() // decimal(3)
```

#### `avg()`

Calculates average of numeric elements.

**Arguments:** None

**Returns:** `number | undefined`

**Description:** Returns the arithmetic mean. Returns `undefined` for empty array.

```go
[1, 2, 3, 4, 5].avg()    // 3
[10, 20, 30].avg()       // 20
[].avg()                 // undefined
```

### Query and Accessor Functions

#### `is_empty()`

Checks if array is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the array has no elements.

```go
[].is_empty()      // true
[1, 2, 3].is_empty()  // false
```

#### `len()`

Gets array length.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the number of elements.

```go
[1, 2, 3].len()    // 3
[].len()           // 0
```

#### `first()`

Gets first element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the first element. Returns `undefined` for empty array.

```go
[1, 2, 3].first()  // 1
[].first()         // undefined
```

#### `last()`

Gets last element.

**Arguments:** None

**Returns:** `any | undefined`

**Description:** Returns the last element. Returns `undefined` for empty array.

```go
[1, 2, 3].last()   // 3
[].last()          // undefined
```

## Examples

### Data Transformation

```go
// Parse and transform data
scores = [85, 92, 78, 95, 88]

// Convert to percentages and filter
results = scores
    .map(s => (s.float() / 100.0) * 100.0)
    .filter(p => p >= 80.0)

println("Passing scores: " + results.len().string())
```

### Accumulation with Reduce

```go
// Calculate total price with tax
items = [
    {name: "Item A", price: 10.0},
    {name: "Item B", price: 20.0},
    {name: "Item C", price: 15.0}
]

total = items.reduce(0.0, (sum, item) => sum + item.price)
tax = total * 0.08
println("Total: $" + total.string())
println("Tax: $" + tax.string())
```

### Complex Filtering

```go
// Multi-condition filtering
users = [
    {name: "Alice", age: 25, active: true},
    {name: "Bob", age: 17, active: true},
    {name: "Carol", age: 30, active: false},
    {name: "Dave", age: 28, active: true}
]

active_adults = users
    .filter(u => u.active)
    .filter(u => u.age >= 18)

println("Active adults: " + active_adults.len().string())
```

### Nested Array Operations

```go
// Process nested arrays
matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]

// Flatten (conceptually)
flattened = []
for row in matrix {
    for val in row {
        flattened = flattened + [val]
    }
}

total = flattened.sum()  // 45
```

### Array Statistics

```go
// Calculate statistics
data = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]

count = data.len()
minimum = data.min()
maximum = data.max()
average = data.avg()

println("Count: " + count.string())
println("Min: " + minimum.string())
println("Max: " + maximum.string())
println("Avg: " + average.string())
```

### Deduplication

```go
// Remove duplicates by filtering
values = [1, 2, 2, 3, 3, 3, 4, 4, 4, 4]

// Simple version using contains
unique = []
for v in values {
    if !unique.contains(v) {
        unique = unique + [v]
    }
}

println(unique)  // [1, 2, 3, 4]
```

## Performance Considerations

- Arrays maintain reference semantics for efficiency
- Use `copy()` to create independent copies when needed
- Sorting modifies the array in-place
- Operations like `filter()` and `map()` create new arrays
- Large arrays with complex predicates may be memory-intensive

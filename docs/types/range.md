# range

Lazy sequences of integers.

## Overview

Ranges represent lazy sequences of integers. They are not evaluated until needed, making them efficient for large or
infinite sequences. Ranges are commonly used in loops and can be converted to arrays when materialization is needed.

## Declaration and Usage

### Construction

Ranges are created with the `range()` function:

```go
range(0, 5)        // 0, 1, 2, 3, 4
range(5, 0)        // Empty (start >= end)
range(0, 10, 2)    // 0, 2, 4, 6, 8 (step 2)
range(5, 0, 1)     // 5, 4, 3, 2, 1 (descending)
```

### Parameters

- **start** (int): Starting value (inclusive)
- **end** (int): Ending value (exclusive)
- **step** (int, optional): Step increment (default 1). Can be negative for descending ranges.

### Using in Loops

```go
for v in range(1, 4) {
    println(v)    // 1, 2, 3
}

for i in range(0, 10, 2) {
    println(i)    // 0, 2, 4, 6, 8
}
```

### Lazy Evaluation

Ranges don't generate values until accessed:

```go
r = range(0, 1000000)  // Very efficient, no memory allocation
r[100]                 // Access single element: 100
r[-1]                  // Access last element: 999999
```

Single-element indexing supports negative indices. Out-of-bounds access raises `index out of bounds`.

## Member Functions

### Conversion Functions

#### `array()`

Converts to array.

**Arguments:** None

**Returns:** `array`

**Description:** Materializes the range into an array of all values.

```go
range(0, 5).array()       // [0, 1, 2, 3, 4]
range(0, 10, 3).array()   // [0, 3, 6, 9]
range(5, 0, 1).array()    // [5, 4, 3, 2, 1]
```

#### `bytes()`

Converts to bytes.

**Arguments:** None

**Returns:** `bytes`

**Description:** Materializes range values as bytes (values must be 0-255).

```go
range(65, 68).bytes()     // bytes with [65, 66, 67] ('A', 'B', 'C')
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts range values to runes and builds a string.

```go
range(65, 68).string()    // "ABC"
```

#### `record()`

Converts to record.

**Arguments:** None

**Returns:** `record`

**Description:** Converts range to a record with string index keys and range values.

```go
range(1, 3, 1).record()   // {"0": 1, "1": 2}
```

#### `dict()`

Converts to dict.

**Arguments:** None

**Returns:** `dict`

**Description:** Converts range to a dict with string index keys and range values.

```go
range(1, 3, 1).dict()     // dict({"0": 1, "1": 2})
```

### Iteration Functions

#### `for_each(fn)`

Executes a callback for each range value.

**Arguments:**

- `fn` (function): Callback function. Accepts one argument `(value)` or two arguments `(index, value)`, and must return `bool`.

**Returns:** `undefined`

**Description:** Calls `fn` for each value without materializing the range. Iteration stops when `fn` returns `false`.

```go
sum = 0
range(1, 4).for_each(v => {
    sum += v
    return true
})
```

### Query and Accessor Functions

#### `is_empty()`

Checks if range is empty.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` if the range contains no values.

```go
range(0, 5).is_empty()    // false
range(5, 5).is_empty()    // true (start == end)
```

#### `len()`

Gets number of values.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the count of values in the range.

```go
range(0, 5).len()         // 5
range(0, 10, 2).len()     // 5
range(5, 0, 1).len()      // 5
```

#### `contains(x)`

Checks if range contains value.

**Arguments:**

- `x` (int): Value to search for

**Returns:** `bool`

**Description:** Returns `true` if the value is in the range.

```go
range(0, 10).contains(5)    // true
range(0, 10).contains(10)   // false (10 is beyond range)
range(0, 10, 2).contains(5) // false (5 is not on step boundary)
range(0, 10, 2).contains(6) // true
```

## Examples

### Looping

```go
// Iterate from 0 to 9
for i in range(0, 10) {
    println(i)
}

// Iterate with custom step
for i in range(0, 20, 5) {
    println(i)  // 0, 5, 10, 15
}

// Descending
for i in range(10, 0, 1) {
    println(i)  // 10, 9, 8, ..., 1
}
```

### Creating Sequences

```go
// Create arrays of sequences
numbers = range(1, 11).array()    // [1, 2, ..., 10]

// Even numbers
evens = range(0, 20, 2).array()   // [0, 2, 4, ..., 18]

// Countdown
countdown = range(10, 0, 1).array()  // [10, 9, 8, ..., 1]
```

### Filtering Ranges

```go
// Convert to array first, then filter
r = range(0, 20)
numbers = r.array()
odd_numbers = numbers.filter(n => n % 2 != 0)

println(odd_numbers)  // [1, 3, 5, ..., 19]
```

### Mapping Over Ranges

```go
// Convert range and apply transformations
r = range(1, 6)
squared = r.array().map(n => n * n)

println(squared)  // [1, 4, 9, 16, 25]
```

### Working with Large Ranges

```go
// Efficient with large ranges (lazy evaluation)
large_range = range(0, 1000000)

// Check without materializing entire range
large_range.contains(500000)    // true (efficient)
len = large_range.len()         // 1000000

// Convert to array only when needed
first_100 = range(0, 100).array()
```

### Generating Test Data

```go
// Create test data
ids = range(1, 101).array()          // [1, 2, ..., 100]

// Create with identifiers
user_ids = range(1000, 1010).array()  // [1000, 1001, ..., 1009]

// Process each
for id in ids {
    println("Processing user " + id.string())
}
```

### Mathematical Sequences

```go
// Powers of 2
powers = range(0, 10).array().map(n => 2 ** n)
// [1, 2, 4, 8, 16, 32, 64, 128, 256, 512]

// Fibonacci-like (would need more complex logic)
// Multiples of 5
multiples = range(1, 11).array().map(n => n * 5)
// [5, 10, 15, 20, 25, 30, 35, 40, 45, 50]
```

### Conditional Ranges

```go
// Create range based on condition
max = 20
if max > 10 {
    nums = range(0, max, 2).array()  // [0, 2, 4, ..., 18]
} else {
    nums = range(0, max).array()
}
```

## Performance Notes

- Ranges use lazy evaluation - values are computed on-demand
- No memory allocation for the entire range until converted to array
- Ideal for large ranges where only some values are accessed
- Converting to array forces materialization of all values
- `contains()` is efficient even for large ranges

## Range Direction

Ranges always progress from start toward end. Determine direction by comparing start and step:

```go
range(0, 10, 1)     // Ascending: 0 to 9
range(0, 10, 2)     // Ascending by 2: 0, 2, 4, 6, 8
range(10, 0, 1)     // Descending: 10, 9, 8, ..., 1
range(10, 0, -1)    // Invalid: step and direction conflict (may produce empty range)
```

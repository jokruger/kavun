# bool

Boolean values representing true or false.

## Overview

Boolean values are used in control flow and logical operations. Kavun has two boolean values: `true` and `false`.

## Declaration and Usage

```go
fmt = import("fmt")
ok = true
flag = false

// Used in control flow
if ok {
    fmt.println("ok is true")
}

// Logical operations
ok && false   // false
ok || false   // true
!ok           // false
```

## Behavior

### Logical Operations

- AND (`&&`): Returns `true` only if both operands are truthy
- OR (`||`): Returns `true` if either operand is truthy
- NOT (`!`): Inverts truthiness

```go
true && true      // true
true && false     // false
false || false    // false
true || false     // true
!true             // false
!false            // true
```

### Control Flow

Booleans are used directly in conditionals and loop conditions:

```go
fmt = import("fmt")

if true {
    fmt.println("always runs")
}

for true {
    fmt.println("infinite loop")
    break
}

for i = 0; i < 5; i = i + 1 {
    fmt.println(i)
}
```

### Coercive Equality and Comparisons

Booleans participate in equality comparisons and can be compared with other types in limited contexts:

```go
true == true          // true
true == false         // false
true != false         // true
```

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `bool`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `bool` is immutable, this method
returns the receiver unchanged.

```go
true.copy()     // true
```

### Conversion Functions

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns the same boolean value.

```go
true.bool()    // true
false.bool()   // false
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Converts `true` to `1` and `false` to `0`.

```go
true.int()     // 1
false.int()    // 0

// Useful for counting true conditions
count = [true, false, true].map(b => b.int()).sum()   // 2
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Converts `true` to `"true"` and `false` to `"false"`.

```go
true.string()    // "true"
false.string()   // "false"

// Used for formatting and display
message = "Status: " + ok.string()   // "Status: true"
```

## Examples

### Basic Logic

```go
fmt = import("fmt")
age = 30
is_waiting = false

// Simple boolean operations
is_valid = age >= 18 && age < 65
is_ready = !is_waiting

if is_valid && is_ready {
    fmt.println("Proceed")
}
```

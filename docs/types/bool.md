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

`bool` widens to `0`/`1` and compares exactly against `byte`, `rune`, `int`, `decimal`, and `float` — either
operand order, always agreeing (`a == b` and `b == a` never disagree):

```go
true == true          // true
true == false         // false
true != false         // true

true == 1             // true -- widens to 1
1 == true             // true -- same either order
true == byte(1)       // true
true == decimal("1")  // true
true == 1.0           // true
false == 0            // true
true == 2             // false -- widens to 1, not "truthy"; 1 != 2
```

`bool` also joins the text tier for equality, comparing against its own canonical text form (`"true"`/`"false"`,
never a truthiness-based leak):

```go
true == "true"       // true
false == "false"     // true
true == "false"      // false -- not "any non-empty string is truthy"
true == "1"          // false -- canonical form is "true", not "1"
```

### Ordering

`bool` orders against itself and against every type in the numeric family (`byte`, `rune`, `int`, `decimal`,
`float`) — widening to `0`/`1` first, the same conversion equality uses. `false` sorts before `true`, which is
what makes `[false, true, false].sort()` a meaningful operation. `bool` still has **no ordering against
`string`/`bytes`/`runes`** (numeric-vs-text ordering is undefined everywhere, not just for `bool`):

```go
false < true      // true
true < false      // false
true <= true      // true

true < 5          // true -- widens to 1
true < byte(5)    // true
true < 'z'        // true
true < decimal("2")  // true
true < 2.5        // true

true < "1"        // runtime error: bool has no ordering against string
```

### Arithmetic

`bool` has **no arithmetic operators at all**, including with itself — `+ - * / %` are all runtime errors, whether
the other operand is `bool`, `int`, or anything else. This is deferred scope, not an oversight; convert explicitly
with `.int()` first if you need to count/sum booleans (see `int()` below).

```go
true + 1          // runtime error
true + true       // runtime error
```

### Unary `^`

Unary `^` is logical negation, the same effect as `!` but a separate operator:

```go
^true       // false
^false      // true
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
true.format()                // "true"
false.format(">7")           // "  false"
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

### Sequence Functions

#### `repeat(n)`

Repeats the boolean `n` times into an array.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `array`

**Description:** Returns a new array of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
b := true
b.repeat(3)      // [true, true, true]
false.repeat(0)  // []
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

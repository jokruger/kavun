# Language Reference

Kavun (кавун, watermelon) is a lightweight, high-performance dynamically typed scripting language designed for embedding in Go. It emphasizes expression-oriented programming with first-class records, arrow-function lambdas, and fluent method chaining. It runs on a sandboxable bytecode VM implemented in Go, with a module system supporting explicit exports. Source files have a `.kvn` extension and content is UTF-8 encoded.

## Builtin types overview

Kavun values are grouped into scalar and container types.

Scalar types:

- [`undefined`](types.md#undefined)
- [`bool`](types.md#bool)
- [`int`](types.md#int)
- [`float`](types.md#float)
- [`decimal`](types.md#decimal)
- [`rune`](types.md#rune)
- [`string`](types.md#string)
- [`runes`](types.md#runes)
- [`bytes`](types.md#bytes)
- [`time`](types.md#time)
- [`error`](types.md#error)

Container types:

- [`array`](types.md#array)
- [`record`](types.md#record)
- [`map`](types.md#map)
- [`range`](types.md#range)

Literal examples:

```go
i = 42
f = 3.14
d = 1.23d
c = 'A'          // rune (Unicode code point)
s = "hello"      // string, double-quoted
r = `raw string` // raw string, backtick-quoted
b = true
u = undefined
```

### Truthiness and equality

Truthiness:

| Value | Truthy? |
|---|---|
| `undefined` | no |
| `false` | no |
| `0` (int) | no |
| `0.0` (float) | yes - all floats are truthy except NaN |
| `decimal(0)` | no |
| `""` (empty string) | no |
| `[]`, `{}`, `map()` | no - empty containers are falsy |
| everything else | yes |

Equality is coercive across types. `==` tries to convert both sides to a common type:

```go
1 == "1"      // true
true == 1     // true
true == 2     // false (bool converts to 0/1)
[1] == ["1"]  // true
```

Use `type_name(x)` to inspect the actual runtime type.

## Lexical basics

Line comments start with `//`. Block comments use `/* ... */`. Statements are separated by newlines; semicolons are inserted automatically after identifiers, literals, closing brackets, and keywords like `break`, `continue`, `return`. A newline before a dot (`.`) is treated as line continuation, not a statement break.

### Numeric literals

Kavun supports several ways to write numeric values:

```go
1         // int
0b1010    // int, binary
0o755     // int, octal
0x1f      // int, hexadecimal

1.0       // float
1f        // float
1.5f      // float
1e3       // float

1d        // decimal
1.23d     // decimal
```

Rules:

- A number without a suffix is an `int` unless it has a fractional part or exponent, in which case it is a `float`.
- The `f` suffix forces a base-10 number to be parsed as `float`.
- The `d` suffix forces a base-10 number to be parsed as `decimal`.
- Prefix forms like `0b`, `0o`, and `0x` are integer literals.
- In hexadecimal literals such as `0x1f`, the `f` is a hex digit, not a float suffix.
- Suffix parsing applies only to base-10 numbers.

## Variables and scope

Kavun supports three declaration forms:

- `var x` declares `x` in the current scope and initializes it to `undefined`.
- `var x = expr` declares `x` in the current scope and assigns `expr`.
- `x := expr` is shorthand for `var x = expr`.

By default, plain `=` assignment is **smart**:

- If the identifier already exists in current or outer scope, `x = expr` assigns to that variable.
- If it does not exist, `x = expr` declares it in the current scope and assigns `expr`.

Compound assignment operators (`+=`, `-=`, etc.) are always strict and require an existing variable.

You can switch plain `=` to strict mode in the compiler/CLI, where unresolved `x = expr` becomes a compile error.

Redeclaring with `:=` in the same scope is a compile error. Variables declared inside `if`/`for` blocks are local to that block. Closures capture free variables by reference, so mutations are visible from the outer scope:

```go
counter = func() {
    n = 0
    return func() { n += 1; return n }
}
inc = counter()
inc() // 1
inc() // 2
```

### Variable scope and shadowing in block initialization

In `if` and `for` statements, plain `=` and `:=` create different scope contexts:

```go
x = 0
y = 0
if x = 10; x > 0 {
    y = 1
} else {
    y = 2
}
// x == 10, y == 1 (= modifies outer x)
```

vs

```go
x = 0
y = 0
if x := 10; x > 0 {
    y = 1
} else {
    y = 2
}
// x == 0, y == 1 (:= declares new local x in if block)
```

In the first example, `x` already exists in outer scope, so `x = 10` modifies that outer variable. In the second example, `x := 10` declares a new local variable `x` confined to the if block scope, shadowing the outer `x`. The outer `x` remains unchanged.

## Expressions

Kavun has arithmetic, comparison, logical, bitwise, membership, and conditional operators.

```go
x = 10 > 5 ? "yes" : "no"
found = "el" in "hello"     // true - substring check
found2 = 2 in [1, 2, 3]     // true - element check
has_key = "a" in {a: 1}     // true - key check
```

### Operator precedence

From lowest to highest:

| Level | Operators |
|---|---|
| 1 | `\|\|` |
| 2 | `&&` |
| 3 | `==` `!=` `<` `<=` `>` `>=` `in` |
| 4 | `+` `-` `\|` `^` |
| 5 | `*` `/` `%` `<<` `>>` `&` `&^` |

Unary operators: `-`, `+`, `!`, `^` (bitwise complement). Ternary `?:` binds looser than all binary operators.

### Complete operator list

| Category | Operators |
|---|---|
| Arithmetic and bitwise | `+` `-` `*` `/` `%` `&` `\|` `^` `<<` `>>` `&^` |
| Comparison and logical | `==` `!=` `<` `<=` `>` `>=` `&&` `\|\|` `!` |
| Membership and conditional | `in` `?:` |
| Assignment and declaration | `=` `:=` |
| Compound assignment | `+=` `-=` `*=` `/=` `%=` `&=` `\|=` `^=` `<<=` `>>=` `&^=` |
| Increment and decrement | `++` `--` |
| Variadic spread in calls | `...` |

String concatenation uses `+` and requires a string on the left. The right side is converted automatically:

```go
"value: " + 42      // "value: 42"
"flag: " + true     // "flag: true"
1 + "x"             // runtime error
```

Indexing and slicing work on strings, runes, arrays, and bytes. Out-of-bounds index returns `undefined` (not an error). Slices clamp silently when either bound is at the natural limit, but raise `invalid slice index` for negative bounds or inverted bounds:

```go
a = [1, 2, 3, 4, 5]
a[10]      // undefined
a[-1:]     // [1,2,3,4,5] - clamped
a[:100]    // [1,2,3,4,5] - clamped
a[:-1]     // error: invalid slice index
a[3:1]     // error: invalid slice index
```

Accessing any field or index on `undefined` returns `undefined`:

```go
undefined.x         // undefined
undefined[0]        // undefined
undefined.a.b.c     // undefined
```

## Statements and control flow

`if` and `for` look like Go. An `if` can include an init statement:

```go
if x = compute(); x > 0 {
    use(x)
} else {
    fallback()
}
```

`for` has four forms:

```go
for { }                         // infinite loop
for condition { }               // while-style
for i = 0; i < 10; i++ { }     // C-style
for v in collection { }         // iterator
for k, v in collection { }      // iterator with key/index
```

The iterator form (`for in`) works on arrays, strings, runes, bytes, records, maps, and ranges. When two variables are used, the first is the index (arrays/strings/runes/bytes) or key (records/maps):

```go
for i, v in [10, 20, 30] { }   // i = 0,1,2; v = element
for k, v in {a: 1, b: 2} { }   // k = key string; v = value
for c in "hello" { }           // c = rune
```

`break` and `continue` work at the innermost loop. `return` exits the current function.

## Functions and lambdas

Functions are first-class values. The short arrow syntax is idiomatic for callbacks:

```go
double = x => x * 2
add = (a, b) => a + b

// Block body needs explicit return
clamp = (v, lo, hi) => {
    if v < lo { return lo }
    if v > hi { return hi }
    return v
}

// Regular function literal
f = func(x, y) {
    return x + y
}
```

Variadic parameters collect extra arguments into an immutable array:

```go
f = func(a, b, ...rest) { return rest }
f(1, 2, 3, 4)   // rest == [3, 4]
```

To spread an array into a call, use `...` after the last argument:

```go
args = [3, 4]
f(1, 2, args...)
```

A function with no `return` statement returns `undefined`.

## Modules

`import("name")` is an expression that loads a module and returns its exported value. Module source can be a builtin module or a Kavun source file.

A Kavun module uses `export` to publish its result. The exported value is automatically made immutable. `export` inside a function body is a compile error.

```go
// math_utils.kvn
export {
    square: func(x) { return x * x },
    cube:   func(x) { return x * x * x },
}
```

```go
// main.kvn
m = import("math_utils")
m.square(4)   // 16
```

A module can also export a single function directly, making the import callable:

```go
// double.kvn
export func(x) { return x * 2 }
```

```go
double = import("double")
double(21)   // 42
```

## Built-in functions

Type conversion builtins accept an optional fallback as second argument. They return `undefined` (or the fallback) when conversion fails:

```go
int("42")                   // 42
int("bad", 0)               // 0
float("3.14")               // 3.14
string(99)                  // "99"
string(undefined)           // undefined  <- not the string "undefined"
bool(0)                     // false
bool(0.0)                   // true  <- float zero is truthy
decimal("1.25")             // decimal value
decimal("bad")              // undefined
decimal("bad", decimal(0))  // decimal(0)
runes("abc")                // runes value
bytes("abc")                // bytes value
time("2024-01-01")          // time value
rune(0)                     // rune 0
```

Collections and helpers:

```go
len(x)                  // length of collection/string/range
copy(x)                 // deep mutable copy
append(arr, v1, v2)     // returns new array
delete(obj, "key")      // mutates record/map in place
splice(arr, start, deleteCount, ...items)  // mutates array, returns deleted slice
map()                   // empty map
map({a: 1})             // map from record
range(0, 10)            // range(start, stop[, step])
error("msg")            // error value with optional payload
type_name(x)            // runtime type name
```

Type predicates:

`is_int`, `is_float`, `is_decimal`, `is_bool`, `is_rune`, `is_string`, `is_runes`, `is_bytes`, `is_array`, `is_record`, `is_map`, `is_range`, `is_time`, `is_error`, `is_undefined`, `is_function`, `is_callable`, `is_iterable`, `is_immutable`

```go
is_array([1, 2])   // true
is_callable(len)   // true
type_name({})      // "record"
```

Formatting:

```go
format("x=%d y=%v", 1, [2, 3])   // "x=1 y=[2, 3]"
```

Read more about formatting verbs in [Formatting](formatting.md).

## Errors and diagnostics

Error messages include a source position:

```sh
Runtime Error: invalid binary operator: int + string
    at myfile.kvn:12:5
```

For runtime errors that bubble through multiple call frames, each frame is shown. Common runtime errors:

- `invalid binary operator: T op T` - operator not supported for the given types
- `wrong number of arguments: want=N, got=M`
- `not callable: T` - tried to call a non-function value
- `unresolved reference 'x'` - variable not declared
- `redeclared: 'x'` - `:=` used on an already-declared variable in the same scope
- `index out of bounds` - assignment to an out-of-range array index
- `invalid slice index` - negative or inverted slice bounds
- `object is not assignable` - assignment target is `undefined`

The `error(payload)` built-in creates an error value that can be returned from functions and inspected:

```go
e = error("something went wrong")
e.value()      // "something went wrong"
is_error(e)    // true
```

## Detailed type documentation

For detailed per-type semantics, conversions, member functions, and type-specific edge cases, see [Type reference](types.md).
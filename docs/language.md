# Language Reference

Kavun (кавун, watermelon) is a lightweight, high-performance dynamically typed scripting language designed for embedding
in Go. It emphasizes expression-oriented programming with first-class records, arrow-function lambdas, and fluent method
chaining. It runs on a sandbox-able bytecode VM implemented in Go, with a module system supporting explicit exports.
Source files have a `.kvn` extension and content is UTF-8 encoded.

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
- [`dict`](types.md#dict)
- [`range`](types.md#range)

Literal examples:

```go
i = 42
f = 3.14
d = 1.23d
c = 'A'              // rune (Unicode code point)
s = "hello"          // string, double-quoted
rs = u"привіт"       // runes (unicode string), u"..." syntax
r = `raw string`     // raw string, backtick-quoted
raw_re = r"\d+\w*"   // raw string (no escape processing), r"..." syntax
fs = f"x={i:5d}"     // f-string (interpolated), f"..." syntax
b = true
u = undefined
```

See [F-Strings](f-strings.md) and [Format Mini-Language](format-mini-language.md) for the full f-string syntax
(expression interpolation, format specs, escape rules, semantics, and differences from Python's f-strings).
For runtime templating with the same placeholder syntax see the [`format(template, args)` builtin](format-function.md).

### Truthiness and equality

Truthiness:

| Value                | Truthy?                                |
| -------------------- | -------------------------------------- |
| `undefined`          | no                                     |
| `false`              | no                                     |
| `0` (int)            | no                                     |
| `0.0` (float)        | yes - all floats are truthy except NaN |
| `decimal(0)`         | no                                     |
| `""` (empty string)  | no                                     |
| `[]`, `{}`, `dict()` | no - empty containers are falsy        |
| everything else      | yes                                    |

Equality is coercive across types. `==` tries to convert both sides to a common type:

```go
1 == "1"      // true
true == 1     // true
true == 2     // false (bool converts to 0/1)
[1] == ["1"]  // true
```

Use `type_name(x)` to inspect the actual runtime type.

## Lexical basics

Line comments start with `//`. Block comments use `/* ... */`. Statements are separated by newlines; semicolons are
inserted automatically after identifiers, literals, closing brackets, and keywords like `break`, `continue`, `return`.
A newline before a dot (`.`) is treated as line continuation, not a statement break.

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

Redeclaring with `:=` in the same scope is a compile error. Variables declared inside `if`/`for` blocks are local to
that block. Closures capture free variables by reference, so mutations are visible from the outer scope:

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

In the first example, `x` already exists in outer scope, so `x = 10` modifies that outer variable. In the second
example, `x := 10` declares a new local variable `x` confined to the if block scope, shadowing the outer `x`. The outer
`x` remains unchanged.

### Shadowing and reassigning builtins

Builtin functions (`len`, `append`, `int`, `string`, etc.) behave like pre-seeded global values: they may be shadowed
in inner scopes via `:=` and reassigned at the top level via `:=` or `=`.

```go
len := func(x) { return 0 }       // top-level: replaces `len` in this script
out := len("anything")             // 0

f := func() {
    len := 10                      // shadows builtin inside this function
    return len
}
g := f()                           // 10
h := len("ab")                     // outer scope still sees the builtin: 2
```

Reassignment is **per-script** and does not affect:

- the original builtin registry inside the VM (a single VM running multiple scripts is unaffected — each script
  compiles against a fresh symbol table);
- imported modules (each module compiles with its own table seeded from the original builtins);
- already-emitted bytecode at earlier call sites in the same script (a reference compiled before the reassignment line
  still resolves to the original builtin).

Compound assignments (`+=`, `-=`, etc.) on a builtin name remain a compile error, since builtins have no addressable
storage to read-modify-write.

## Expressions

Kavun has arithmetic, comparison, logical, bitwise, membership, and conditional operators.

```go
x = 10 > 5 ? "yes" : "no"
found = "el" in "hello"      // true - substring check
found2 = 2 in [1, 2, 3]      // true - element check
has_key = "a" in {a: 1}      // true - key check
missing = "z" not in "hello" // true - negated membership check
```

### Operator precedence

From lowest to highest:

| Level | Operators                                 |
| ----- | ----------------------------------------- |
| 1     | `\|\|`                                    |
| 2     | `&&`                                      |
| 3     | `==` `!=` `<` `<=` `>` `>=` `in` `not in` |
| 4     | `+` `-` `\|` `^`                          |
| 5     | `*` `/` `%` `<<` `>>` `&` `&^`            |

Unary operators: `-`, `+`, `!`, `^` (bitwise complement). Ternary `?:` binds looser than all binary operators.

### Complete operator list

| Category                   | Operators                                                  |
| -------------------------- | ---------------------------------------------------------- |
| Arithmetic and bitwise     | `+` `-` `*` `/` `%` `&` `\|` `^` `<<` `>>` `&^`            |
| Comparison and logical     | `==` `!=` `<` `<=` `>` `>=` `&&` `\|\|` `!`                |
| Membership and conditional | `in` `not in` `?:`                                         |
| Assignment and declaration | `=` `:=`                                                   |
| Compound assignment        | `+=` `-=` `*=` `/=` `%=` `&=` `\|=` `^=` `<<=` `>>=` `&^=` |
| Increment and decrement    | `++` `--`                                                  |
| Variadic spread in calls   | `...`                                                      |

String concatenation uses `+` and requires a string on the left. The right side is converted automatically:

```go
"value: " + 42      // "value: 42"
"flag: " + true     // "flag: true"
1 + "x"             // runtime error
```

Indexing works on strings, runes, arrays, bytes, and ranges. Slicing works on strings, runes, arrays, and bytes.
Single-element indexing supports negative indices: `[-1]` is the last element, `[-2]` the second from the end, and
so on. Out-of-bounds index access raises `index out of bounds`. Two-part slices follow the same rules: negative bounds
count from the end, omitted bounds default to the natural edge, oversized bounds clamp silently, and an inverted slice
returns an empty result. Arrays, strings, runes, and bytes also support three-part slices `start:end:step`: `step` is
optional, can be negative, and cannot be zero.

```go
a = [1, 2, 3, 4, 5]
a[-1]      // 5
a[10]      // runtime error: index out of bounds
a[-1:]     // [5]
a[:100]    // [1,2,3,4,5]
a[:-1]     // [1,2,3,4]
a[-3:-1]   // [3,4]
a[3:1]     // []
a[1:5:2]   // [2,4]
a[::-1]    // [5,4,3,2,1]
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

The iterator form (`for in`) works on arrays, strings, runes, bytes, records, dicts, and ranges. When two variables are
used, the first is the index (arrays/strings/runes/bytes) or key (records/dicts):

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

### Named return value

A function can declare an optional named result between the parameter list and the body. The name is bound as a local
variable initialized to `undefined`. A bare `return` (or falling off the end of the body) yields the current value of
that variable; an explicit `return expr` overrides it.

```go
counter := func() n {
    n = 0
    n = n + 1
    // implicit return n
}
counter()   // 1

clamp := func(x, lo, hi) result {
    if x < lo { result = lo; return }
    if x > hi { result = hi; return }
    result = x
}
```

Named results are most useful in combination with `defer` (see below): a deferred function can read or modify the
named result before the caller sees it.

The result name must not be `_` and must not collide with a parameter name.

## Deferred calls

The `defer` statement schedules a function or method call to run when the surrounding function exits — whether through
a `return`, falling off the end, or a runtime error. Deferred calls run in LIFO order.

```go
open_and_use := func(path) {
    f := fs.open(path)
    defer f.close()      // always runs, even on error
    use(f)
}
```

Argument expressions of a `defer`'d _plain_ call are evaluated immediately; the values are captured for later use:

```go
f := func() {
    x := 10
    defer record(x)      // records 10
    x = 20
}
```

Defer the call of an anonymous function to capture variables by reference instead:

```go
f := func() {
    x := 10
    defer func() { record(x) }()   // records 20
    x = 20
}
```

`defer` is only valid inside a function body, and the deferred expression must be a function or method call.

## Errors and recovery

Kavun has a built-in `error` value type (see `docs/types/error.md`). Two ways an error can flow:

1. **As a value** — built with `error(payload)` and passed around explicitly. `is_error(v)` checks for one.
2. **As a raised error** — the runtime aborts the current execution and unwinds frames until a `recover()` catches it.

Errors are split into two severities:

- **Logical** errors (most runtime errors: division by zero, type errors, missing members, …) and user-raised errors
  via `raise()` can be caught by `recover()`.
- **Critical** errors (stack overflow, allocation limits, internal logic invariants) are not recoverable — they always
  terminate the program.

### `raise(err)`

The `raise(err)` builtin raises a Kavun error so that surrounding deferred `recover()` calls can catch it. If `err` is
not already an error value, it is wrapped: `raise("boom")` is equivalent to `raise(error("boom"))`, and
`raise({code: 42})` is equivalent to `raise(error({code: 42}))`.

### `recover()`

`recover()` is a builtin that, when called **directly inside a deferred function**, returns the in-flight error and
clears it (so the caller sees a normal return). Outside a deferred function, or when there is no in-flight error,
`recover()` returns `undefined`.

Combine `defer`, `recover()`, and a named result for Go-style error handling:

```go
safe_div := func(a, b) result {
    defer func() {
        e := recover()
        if e != undefined {
            result = error({op: "safe_div", cause: e.value()})
        }
    }()
    result = a / b   // may raise on b == 0
}

r := safe_div(10, 0)
if is_error(r) { fmt.println("failed:", r.value()) }
```

Inside `recover()`'s returned error you can inspect:

- `e.kind()` — stable string tag (e.g. `"division_by_zero"`) for runtime errors, `"user"` for errors created in script
- `e.is_runtime()` — `true` if raised by the runtime, `false` if raised via `error(...)` (i.e. `kind() == "user"`)
- `e.value()` — the payload (a string with the runtime message for runtime errors, or whatever was passed to `error(...)` for user errors)

#### Where `recover()` is effective

`recover()` only clears an in-flight error when called **directly** inside a deferred function literal — concretely, the
current call frame must be a script function that was invoked as a defer. The following forms do **not** enable
recovery, and a raised error will escape:

- `defer obj.method()` — method dispatch does not establish a deferred-for frame.
- `defer some_builtin()` — host/builtin calls run without a Kavun frame.
- `defer func() { helper() }()` where `helper` calls `recover()` — `helper` is a separate frame and its
  `recover()` returns `undefined`.

Do the `recover()` call in the deferred literal itself and pass the recovered value to any helpers:

```go
defer func() {
    if e := recover(); e != undefined {
        log_failure(e)   // helper handles the value; recover() stays in the literal
    }
}()
```

#### `return EXPR` and named results

For a function with a named result, `return EXPR` is sugar for `name = EXPR; return` — the expression is assigned to
the named result before defers run, so a deferred function can observe and mutate it by name:

```go
inc := func(x) r {
    defer func() { r = r + 1 }()
    return x   // returns x + 1
}
```

This matches Go. Functions without a named result return EXPR unchanged regardless of any defers.

## Modules

`import("name")` is an expression that loads a module and returns its exported value. Module source can be a builtin
module or a Kavun source file.

A Kavun module uses `export` to publish its result. The exported value is automatically made immutable. `export` inside
a function body is a compile error.

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

Type conversion builtins accept an optional fallback as second argument. They return `undefined` (or the fallback) when
conversion fails:

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
delete(obj, "key")      // mutates record/dict in place
splice(arr, start, deleteCount, ...items)  // mutates array, returns deleted slice
dict()                  // empty dict
dict({a: 1})            // dict from record
range(0, 10)            // range(start, stop[, step])
error("msg")            // error value with a string payload
error({code: 42})       // error value with a structured payload
raise(err)              // raise an error so a deferred recover() can catch it
recover()               // inside a deferred function, return & clear the in-flight error
type_name(x)            // runtime type name
format(template, args)  // runtime f-string-style formatting (see below)
```

Formatting:

```go
format("hello {x} from {y}!", {x: "Kavun", y: "Kherson"})  // "hello Kavun from Kherson!"
format("hello {0} from {1}!", ["Kavun", "Kherson"])        // "hello Kavun from Kherson!"
format("pi = {x:.3f}", {x: 3.14159})                       // "pi = 3.142"
format("n = {x:{fmt}}", {x: 42, fmt: "05d"})               // "n = 00042"
```

`format(template, args)` is the runtime counterpart to f-strings. The template uses the same `{name}` / `{0}`
placeholder syntax and the same [Format Mini-Language](format-mini-language.md) for `:fspec`. `args` must be an
`array` (for indexed placeholders) or a `dict` / `record` (for named placeholders); the two modes cannot be mixed
in one template, and expressions are not allowed inside `{...}`. See [`format`](format-function.md) for the full
reference.

Type predicates:

`is_int`, `is_float`, `is_decimal`, `is_bool`, `is_rune`, `is_string`, `is_runes`, `is_bytes`, `is_array`, `is_record`,
`is_dict`, `is_range`, `is_time`, `is_error`, `is_undefined`, `is_function`, `is_callable`, `is_iterable`,
`is_immutable`

```go
is_array([1, 2])   // true
is_callable(len)   // true
type_name({})      // "record"
```

## Errors and diagnostics

Error messages include a source position:

```sh
Runtime Error: invalid binary operator: int + string
    at myfile.kvn:12:5
```

For runtime errors that bubble through multiple call frames, each frame is shown. Common runtime errors:

- `invalid_binary_operator: T op T` - operator not supported for the given types
- `wrong_num_arguments: (call) expected N argument(s), got M`
- `not_callable: type T is not callable` - tried to call a non-function value
- `unresolved reference 'x'` - variable not declared
- `redeclared: 'x'` - `:=` used on an already-declared variable in the same scope
- `index_out_of_bounds` - assignment to an out-of-range array index
- `not_sliceable` / invalid slice bounds
- `not_assignable: type T does not support assignment via indexing or field access`

The `error(payload)` built-in creates an error value that can be returned from functions and inspected. The payload
can be any value — typically a string message, or a structured dict/record for programmatic recovery:

```go
e1 = error("something went wrong")
e1.value()    // "something went wrong"
is_error(e1)  // true

e2 = error({code: 42, message: "boom"})
e2.value().code    // 42
```

Calling `error()` with no arguments is rejected — an empty error carries no information.

## Detailed type documentation

For detailed per-type semantics, conversions, member functions, and type-specific edge cases, see
[Type reference](types.md).

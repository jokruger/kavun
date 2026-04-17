# Language Reference

GS (Go Script) is a lightweight, high-performance dynamically typed scripting language designed for embedding in Go. It emphasizes expression-oriented programming with first-class records, arrow-function lambdas, and fluent method chaining. It runs on a sandboxable bytecode VM implemented in Go, with a module system supporting explicit exports. Source files have a `.gs` extension and content is UTF-8 encoded.

## Lexical basics

Line comments start with `//`. Block comments use `/* ... */`. Statements are separated by newlines — semicolons are inserted automatically after identifiers, literals, closing brackets, and keywords like `break`, `continue`, `return`. A newline before a dot (`.`) is treated as line continuation, not a statement break.

Literal types at a glance:

```go
i := 42
f := 3.14
c := 'A'          // char (Unicode rune)
s := "hello"      // string, double-quoted
r := `raw string` // raw string, backtick-quoted
b := true
u := undefined
```

## Variables and scope

Use `:=` to declare a new variable. Use `=` or compound operators (`+=`, `-=`, etc.) to assign to an existing one. `var x` is shorthand for `x := undefined`.

```go
a := 1
a += 2            // a == 3

var msg           // msg == undefined
msg = "ready"
```

Redeclaring with `:=` in the same scope is a compile error. Variables declared inside `if`/`for` blocks are local to that block. Closures capture free variables by reference, so mutations are visible from the outer scope:

```go
counter := func() {
    n := 0
    return func() { n += 1; return n }
}
inc := counter()
inc() // 1
inc() // 2
```

## Types and values

Scalar types: `undefined`, `bool`, `int`, `float`, `char`, `string`, `bytes`, `time`, `error`.  
Container types: `array`, `record`, `map`, `range`.

**Truthiness** is summarized below:

| Value | Truthy? |
|---|---|
| `undefined` | no |
| `false` | no |
| `0` (int) | no |
| `0.0` (float) | **yes** — all floats are truthy except NaN |
| `""` (empty string) | no |
| `[]`, `{}`, `map()` | no — empty containers are falsy |
| everything else | yes |

**Equality** is coercive across types. `==` tries to convert both sides to a common type:

```go
1 == "1"      // true
true == 1     // true
true == 2     // false (bool converts to 0/1)
[1] == ["1"]  // true
```

Use `type_name(x)` to inspect the actual type at runtime.

## Expressions

GS has the usual arithmetic, comparison, logical, and bitwise operators. The only notable addition is `in` for membership tests, and `?:` for ternary conditionals:

```go
x := 10 > 5 ? "yes" : "no"
found := "el" in "hello"     // true — substring check
found2 := 2 in [1, 2, 3]     // true — element check
has_key := "a" in {a: 1}     // true — key check
```

**Operator precedence** (lowest to highest):

| Level | Operators |
|---|---|
| 1 | `\|\|` |
| 2 | `&&` |
| 3 | `==` `!=` `<` `<=` `>` `>=` `in` |
| 4 | `+` `-` `\|` `^` |
| 5 | `*` `/` `%` `<<` `>>` `&` `&^` |

Unary: `-`, `+`, `!`, `^` (bitwise complement). Ternary `?:` binds looser than all binary operators.

**Complete operator list**

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

**Indexing and slicing** work on strings, arrays, and bytes. Out-of-bounds index returns `undefined` (not an error). Slices clamp silently when either bound is at the natural limit, but raise `invalid slice index` for negative bounds or inverted bounds:

```go
a := [1, 2, 3, 4, 5]
a[10]      // undefined
a[-1:]     // [1,2,3,4,5] — clamped
a[:100]    // [1,2,3,4,5] — clamped
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
if x := compute(); x > 0 {
    use(x)
} else {
    fallback()
}
```

`for` has four forms:

```go 
for { }                         // infinite loop
for condition { }               // while-style
for i := 0; i < 10; i++ { }     // C-style
for v in collection { }         // iterator
for k, v in collection { }      // iterator with key/index
```

The iterator form (`for in`) works on arrays, strings, bytes, records, maps, and ranges. When two variables are used, the first is the index (arrays/strings/bytes) or key (records/maps):

```go
for i, v in [10, 20, 30] { }    // i = 0,1,2; v = element
for k, v in {a: 1, b: 2} { }   // k = key string; v = value
for c in "hello" { }            // c = char
```

`break` and `continue` work at the innermost loop. `return` exits the current function.

## Functions and lambdas

Functions are first-class values. The short arrow syntax is idiomatic for callbacks:

```go
double := x => x * 2
add := (a, b) => a + b

// Block body needs explicit return
clamp := (v, lo, hi) => {
    if v < lo { return lo }
    if v > hi { return hi }
    return v
}

// Regular function literal
f := func(x, y) {
    return x + y
}
```

Variadic parameters collect extra arguments into an immutable array:

```go
f := func(a, b, ...rest) { return rest }
f(1, 2, 3, 4)   // rest == [3, 4]
```

To spread an array into a call, use `...` after the last argument:

```go
args := [3, 4]
f(1, 2, args...)
```

A function with no `return` statement returns `undefined`.

## Collections

### Arrays

Arrays are mutable and reference-typed (`a := b` makes both point at the same array):

```go
a := [1, 2, 3]
b := a
a[0] = 99
// b[0] == 99
```

Key methods: `len()`, `is_empty()`, `first()`, `last()`, `sort()`, `contains(x)`, `filter(fn)`, `map(fn)`, `reduce(init, fn)`, `all(fn)`, `any(fn)`, `count(fn)`, `sum()`, `avg()`, `min()`, `max()`, `to_string()`, `to_bytes()`, `to_record()`.

Lambda callbacks for `filter`/`map`/etc. can accept one argument (value) or two (index, value):

```go
[1, 2, 3, 4].filter(x => x % 2 == 0)          // [2, 4]
[1, 2, 3].map((i, v) => i * v)                // [0, 2, 6]
[1, 2, 3].reduce(0, (acc, v) => acc + v)      // 6
```

### Records

Records are the primary object type. Keys are strings. Both dot notation and index notation work:

```go
r := {name: "Alice", age: 30}
r.name       // "Alice"
r["age"]     // 30
r.missing    // undefined

r.city = "Berlin"   // add new key
delete(r, "age")    // remove key

"name" in r  // true — key existence check
```

Records are also reference-typed.

### Maps

`map` is similar to a record but only supports index access (dot notation is used for member functions) and is backed by a separate runtime type. Create with `map()` or `map(record)`:

```go
m := map({a: 1, b: 2})
m["a"]       // 1
m.a          // runtime error — dot access not allowed on map

m.keys()     // array of keys (unsorted)
m.values()   // array of values
```

### Strings and chars

Strings are immutable and indexed by Unicode rune, not byte:

```go
s := "ウクライナ"
s[0]         // char 'ウ'
s[0:2]       // "ウク"
len(s)       // 5 (rune count)
```

Char literals are single runes. Arithmetic with chars works on their Unicode code point:

```go
'A' + 1   // 66 (int)
'9' - '0' // 9 (int)
```

Key string methods: `len()`, `is_empty()`, `first()`, `last()`, `upper()`, `lower()`, `trim([cutset])`, `contains(x)`, `to_array()`, `to_bytes()`, `to_int()`, `to_float()`, `to_bool()`, `to_char()`.

### Bytes

Bytes are mutable byte arrays. Indexing returns an `int` (0–255):

```go
b := bytes("abc")
b[0]                            // 97
b[0:2]                          // bytes slice
bytes("abc") + bytes("def")     // concatenation
```

### Ranges

Ranges are lazy sequences. `range(start, stop)` and `range(start, stop, step)` with `step > 0`:

```go
range(0, 5).to_array()       // [0, 1, 2, 3, 4]
range(5, 0, 1).to_array()    // [5, 4, 3, 2, 1]
range(0, 10, 2).contains(4)  // true

for v in range(1, 4) { }     // v = 1, 2, 3
```

### Immutability

`immutable(x)` makes a container immutable at the container level. Mutating an immutable container raises a runtime error. `copy()` always returns a mutable deep copy, even of an immutable value:

```go
a := immutable([1, 2, 3])
a[0] = 9       // runtime error
type_name(a)   // "immutable-array"

b := copy(a)   // mutable copy
b[0] = 9       // ok
```

## Modules

`import("name")` is an expression that loads a module and returns its exported value. Module source can be a builtin module or a GS source file.

A GS module uses `export` to publish its result. The exported value is automatically made immutable. `export` inside a function body is a compile error.

```go
// math_utils.gs
export {
    square: func(x) { return x * x },
    cube:   func(x) { return x * x * x },
}
```

```go
// main.gs
m := import("math_utils")
m.square(4)   // 16
```

A module can also export a single function directly, making the import callable:

```go
// double.gs
export func(x) { return x * 2 }
```

```go
double := import("double")
double(21)   // 42
```

## Built-in functions

**Type conversion** — all accept an optional fallback as second argument. Returns `undefined` (or the fallback) when conversion fails:

```go
int("42")          // 42
int("bad", 0)      // 0
float("3.14")      // 3.14
string(99)         // "99"
string(undefined)  // undefined  ← not the string "undefined"
bool(0)            // false
bool(0.0)          // true  ← float zero is truthy
bytes("abc")       // bytes value
time("2024-01-01") // time value
```

**Collections:**

```go
len(x)                  // length of any collection, string, or range
copy(x)                 // deep mutable copy
append(arr, v1, v2)     // returns new array
delete(obj, "key")      // mutates record/map in place
splice(arr, start, deleteCount, ...items)  // mutates array, returns deleted slice
map()                   // empty map
map({a: 1})             // map from record
range(0, 10)            // range(start, stop[, step])
error("msg")            // error value with optional payload
```

**Type predicates:**

`is_int`, `is_float`, `is_bool`, `is_char`, `is_string`, `is_bytes`, `is_array`, `is_record`, `is_map`, `is_range`, `is_time`, `is_error`, `is_undefined`, `is_function`, `is_callable`, `is_iterable`, `is_immutable`

```go
is_array([1, 2])   // true
is_callable(len)   // true
type_name({})      // "record"
```

**Formatting:**

```go
format("x=%d y=%v", 1, [2, 3])   // "x=1 y=[2, 3]"
```

## Errors and diagnostics

Error messages include a source position:

```sh
Runtime Error: invalid binary operator: int + string
    at myfile.gs:12:5
```

For runtime errors that bubble through multiple call frames, each frame is shown. Common runtime errors:

- `invalid binary operator: T op T` — operator not supported for the given types
- `wrong number of arguments: want=N, got=M`
- `not callable: T` — tried to call a non-function value
- `unresolved reference 'x'` — variable not declared
- `redeclared: 'x'` — `:=` used on an already-declared variable in the same scope
- `index out of bounds` — assignment to an out-of-range array index
- `invalid slice index` — negative or inverted slice bounds
- `object is not assignable` — assignment target is `undefined`

The `error(payload)` built-in creates an error value that can be returned from functions and inspected:

```go
e := error("something went wrong")
e.value()      // "something went wrong"
is_error(e)    // true
```

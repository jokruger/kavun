# Kavun Cheat Sheet

A one-page, skimmable reference to the Kavun language. For full semantics see
[Language Reference](language.md), [Type Reference](types.md), and [Standard Library](stdlib.md).

## Hello world

```go
fmt = import("fmt")
fmt.println("hello, world")
```

## What makes Kavun different

- **Single-threaded, sandboxed, embeddable in pure Go** — no cgo, no goroutines/channels exposed to scripts;
  the host controls concurrency and resource limits, the script stays deterministic.
- **Deterministic & reproducible by design** — same input, same bytecode, same output, every time; this is what
  makes the AST optimizer's constant-folding safe (see [purity contract](purity.md)) and what makes scripts
  auditable for finance/decisioning use cases.
- **`decimal` is a first-class type**, not a float workaround — exact arithmetic for money.
- **`defer`/`recover`** give Go-style cleanup and error handling without Go panics on the hot path.
- **Chained field/index access on `undefined` short-circuits to `undefined`** (`a.b.c` never panics on a
  missing `a`) — no `?.` operator needed, unlike most C-family languages.
- **Non-mutating by default** — collection methods return new values; an explicit mutating variant, if one is ever
  added, would be named `..._in_place`.

## Comments & statements

```go
// line comment
/* block comment */

x = 1; y = 2        // ';' separates statements on one line (auto-inserted at line end)
obj
    .method()        // leading '.' continues the previous statement
```

## Variables & assignment

Plain `=` is **smart**: it assigns if the name already exists in scope, or declares it (in the current scope) if it
doesn't. Prefer `=` everywhere — use `:=` only when you deliberately want to force a **new local** variable that
shadows an existing outer one of the same name (see below); `:=` is otherwise just extra punctuation for what `=`
already does.

```go
var x             // declares x = undefined
var x = 1         // declares + assigns

x = 1             // smart '=': declares x (first use) or assigns it (if it already exists)
x += 1            // compound ops (+= -= *= /= %= &= |= ^= <<= >>= &^=) require an existing var
x++; x--          // increment / decrement

a, b = 1, 2       // parallel assignment (both sides same count, LHS = plain names)
a, b = b, a       // swap — RHS fully evaluated before any LHS is stored

a, b, c = [10, 20, 30]      // destructuring: array -> positional
a, b = {a: 1, b: 2, c: 3}   // destructuring: dict/record -> keyed by name, missing key = undefined
a, _, c = [1, 2, 3]         // '_' discards a position (never readable, may repeat)
```

Forcing shadowing with `:=` — the one case `:=` is for:

```go
x = 0
if x := 10; x > 0 {   // ':=' forces a fresh LOCAL x scoped to this if, shadowing the outer x
    use(x)              // sees 10
}
// outer x is still 0 here
```

Builtins (`len`, `int`, `range`, ...) behave like pre-seeded globals — shadow with `:=`, reassign with `=`,
per-script only.

## Types at a glance

| Type      | Literal                | Mutable?     | Notes                              |
| --------- | ----------------------- | ------------ | ----------------------------------- |
| `int`     | `42`, `0x1f`, `0b1010`, `0o755` | value | 64-bit signed                     |
| `float`   | `3.14`, `1f`, `1e3`     | value        | IEEE-754 double                    |
| `decimal` | `1.23d`, `1d`           | value        | exact, for money/finance           |
| `bool`    | `true`, `false`         | value        |                                     |
| `rune`    | `'A'`                   | value        | Unicode code point                 |
| `byte`    | `b'A'`, `b'\x00'`       | value        | 0-255                               |
| `string`  | `"hi"`, `` `raw` ``, `r"\d+"` | immutable | UTF-8, byte-indexed             |
| `runes`   | `u"привіт"`             | immutable    | rune-indexed unicode string        |
| `bytes`   | `b"hi"`                 | reference    | binary data                        |
| `time`    | `t"2024-01-01"`         | value        | instant in time                    |
| `array`   | `[1, 2, 3]`             | reference    | heterogeneous, ordered             |
| `record`  | `{a: 1, b: 2}`          | reference    | dot + index access, fields only    |
| `dict`    | `dict({a: 1})`          | reference    | index access only (`.` = methods)  |
| `range`   | `1..5`                  | lazy         | see below                          |
| `error`   | `error("msg")`          | value        | payload can be any value           |
| `undefined` | `undefined`           | value        | absence of a value                 |

Reference types (`array`, `bytes`, `record`, `dict`, immutable containers) alias on assignment — use `copy()` for an
independent copy. Value types (everything else) copy by value.

```go
type_name(x)     // runtime type name, e.g. "int", "array"
is_int(x); is_array(x); is_callable(x); is_iterable(x); is_immutable(x)   // ... is_T for every builtin type

immutable(x)      // returns a locked, read-only view of a reference type -- mutation raises a runtime error
```

## Truthiness & equality

```go
// falsy: undefined, false, 0 (int), decimal(0), "", [], {}, dict() -- everything else is truthy
// NOTE: 0.0 (float) is truthy -- all floats are truthy except NaN

1 == "1"      // true  -- '==' coerces to a common type
true == 1     // true
[1] == ["1"]  // true
```

## Operators

```go
+ - * / % & | ^ << >> &^        // arithmetic & bitwise
== != < <= > >= && || !          // comparison & logical
in / not in                      // membership: substring / element / key
? :                              // ternary, loosest-binding
.. / ..:                         // range literal, see below
```

Precedence (low -> high): `||`  →  `&&`  →  `== != < <= > >= in not in`  →  `..` (range)  →  `+ - | ^`  →  `* / % << >> & &^`.

```go
"el" in "hello"        // substring check -> true
2 in [1, 2, 3]          // element check   -> true
"a" in {a: 1}           // key check       -> true
"value: " + 42          // string concat, RHS auto-converted -> "value: 42"
```

## Indexing & slicing

```go
a = [1, 2, 3, 4, 5]
a[-1]        // 5      -- negative index counts from the end
a[10]        // runtime error: index out of bounds
a[1:3]       // [2, 3] -- half-open: start inclusive, stop exclusive
a[1..3]      // [2, 3] -- '..' is an alternate spelling of ':' inside brackets
a[:-1]       // [1,2,3,4]
a[1:5:2]     // [2, 4] -- start:stop:step, step may be negative
a[::-1]      // [5,4,3,2,1] -- reverse
```

Works on `string`, `runes`, `array`, `bytes` (slicing); `range` supports single-element indexing only.
Accessing a field/index on `undefined` returns `undefined` (`undefined.a.b.c // undefined`).

## Range literals

```go
1..5          // range(1, 5)    -> 1,2,3,4     (exclusive stop, like range()/slices)
5..1          // range(5, 1)    -> 5,4,3,2      (direction auto-detected)
1..5:2        // range(1, 5, 2) -> 1,3
array(1..5)   // [1,2,3,4] -- materialize into an array
```

## Control flow

```go
if x > 0 { ... } else if x < 0 { ... } else { ... }
if x = compute(); x > 0 { use(x) }     // init clause, shares scope with then/else

for { }                          // infinite
for x < 10 { }                    // while-style
for i = 0; i < 10; i++ { }        // C-style
for v in collection { }           // iterator: array/string/runes/bytes -> value; record/dict -> value
for k, v in collection { }        // iterator with index (seq) or key (dict/record)

break; continue                   // innermost loop only
```

## Functions & lambdas

```go
double = x => x * 2                        // arrow lambda, expression body
add = (a, b) => a + b
clamp = (v, lo, hi) => {                    // block body needs explicit return
    if v < lo { return lo }
    if v > hi { return hi }
    return v
}
f = func(x, y) { return x + y }             // regular function literal

f = func(a, b, ...rest) { return rest }     // variadic -> immutable array
f(1, 2, 3, 4)                                // rest == [3, 4]
args = [3, 4]
f(1, 2, args...)                             // spread into a call

counter = func() n {           // named result: local var, implicit `return` yields it
    n = 0
    n = n + 1
}
counter()   // 1
```

A function with no `return` returns `undefined`.

## Placeholder syntax (`_`)

```go
add = func(a, b, c) { return a + b + c }
add(1, _, 3)          // same as: x => add(1, x, 3)
_(2, 5)                // same as: f => f(2, 5) -- '_' can be the callee too
_.name                 // same as: x => x.name
_ + _                  // same as: (x, y) => x + y
arr[_]                 // same as: x => arr[x]
_ ? "yes" : "no"       // same as: x => x ? "yes" : "no"

foo(1, bar(_, 2))      // '_' binds to the CLOSEST qualifying node -- here bar(_,2), not foo
```

Sugar for the simple, single-application case only -- no aliasing (`foo(_, _)` is always 2 params), no expression
boundaries. Need either? Write the arrow lambda by hand, e.g. `x => foo(x) + bar(x)`. Deliberate, not an
implementation gap: the whole point is that `_` stays mechanical to read at a glance -- see
[full corner-case table](language.md#placeholder-syntax-_).

```go
_._                 // same as: x => x._  -- '_' is a legal FIELD NAME, receiver is the placeholder
_._(_, _)           // same as: (a, b, c) => a._(b, c) -- method/field NAME is never a placeholder target
[1, _, 3]           // compile error -- '_' inside a composite literal is not a placeholder position at all
```

## Defer / errors / recover

```go
open_and_use = func(path) {
    f = fs.open(path)
    defer f.close()               // LIFO order, runs on return / fallthrough / error
    use(f)
}

safe_div = func(a, b) result {
    defer func() {
        e = recover()               // must be called directly inside the defer literal
        if e != undefined {
            result = error({op: "safe_div", cause: e.value()})
        }
    }()
    result = a / b                 // may raise on b == 0
}

r = safe_div(10, 0)
if is_error(r) { fmt.println("failed:", r.value()) }

// e.kind()        -- "division_by_zero", ... or "user" for script-raised errors
// e.is_runtime()  -- true for runtime errors, false for error(...) values
// e.value()       -- the payload

raise("boom")          // == raise(error("boom")); unwinds until a recover() catches it
error({code: 42})       // build an error value directly (doesn't unwind)
```

Common runtime error kinds (from `e.kind()` / error messages, full list in [language reference](language.md#errors-and-diagnostics)):

| Kind | Means |
| ---- | ----- |
| `invalid_binary_operator` | operator not supported for the given types |
| `wrong_num_arguments` | call arg count doesn't match the function |
| `not_callable` | tried to call a non-function value |
| `unresolved reference` | variable not declared |
| `redeclared` | `:=` used on an already-declared name in the same scope |
| `index_out_of_bounds` | out-of-range array/string/bytes index |
| `not_sliceable` | invalid slice target or bounds |
| `not_assignable` | type doesn't support assignment via indexing/field access |
| `division_by_zero` | `/` or `%` with a zero divisor |

## Modules

```go
// math_utils.kvn
export { square: func(x) { return x * x } }

// main.kvn
m = import("math_utils")
m.square(4)   // 16
```

Builtin modules: `fmt`, `math`, `os`, `text`, `times`, `json`, `base64`, `hex`, `rand`, `errors`.

```go
fmt = import("fmt");     fmt.println("sum:", 20 + 22)
math = import("math");   math.sqrt(144)
text = import("text");   text.trim_space("  hi  ")
json = import("json");   json.encode({a: 1})
times = import("times"); times.now()
rand = import("rand");   rand.int_n(100)
```

## Strings, f-strings & format

```go
"hello" + " " + "world"          // concatenation
"Hi %s" -- no printf verbs; use f-strings or format() instead

name = "world"; n = 42
f"hello, {name}! n={n:5d}"       // f-string: compiled once, evaluated at each run
f"{{literal braces}}"            // '{{' / '}}' escape to a literal brace

format("hello {x} from {y}!", {x: "Kavun", y: "Kherson"})   // runtime template, same {…} syntax
format("pi = {x:.3f}", {x: 3.14159})                         // "pi = 3.142"
```

Format spec (after `:`) — `[[fill]align][sign][width][,|_][.precision][~|!][verb]`:

```go
f"{n:5d}"     // width 5, decimal
f"{n:05d}"    // zero-padded
f"{n:+d}"     // force sign
f"{x:.2f}"    // 2 decimal places
f"{n:,d}"     // thousands grouping
f"{s:>10}"    // right-align in width 10
```

## Collections cheat sheet

```go
len(x); copy(x); is_empty(x); contains(x, v)
min(3, 1, 2); max(3, 1, 2)  // 1; 3 -- variadic, 0 args => undefined, 1 arg => itself, min(arr...) == arr.min()

// array
a = [3, 1, 2]
a.sort()                    // new sorted array (non-mutating; no in-place variant exists today)
a.filter(x => x > 1)        // [3, 2]
a.map(x => x * 2)           // [6, 2, 4]
a.reduce(0, (acc, v) => acc + v)   // 6
a.find(x => x == 2)         // 2
a.reverse(); a.unique(); a.flatten(); a.chunk(2)
a.sum(); a.avg(); a.min(); a.max(); a.count(x => x > 1)
a.append(4, 5)              // returns NEW array (not in-place), even with 0 items; a.append_in_place(4, 5) mutates, returns receiver
a.slice_view(1, 3); a.chunk_view(2); a.is_view()   // explicit sharing opt-ins (P3-003/P4-002); a[i:j]/chunk() always copy
a.splice(0, 1, 9)           // returns NEW array (not in-place); a.splice_in_place(0, 1, 9) mutates, returns deleted slice
                             // splice/splice_in_place also work on bytes/runes (P5-002), same shape as array's
a.copy_shallow(); a.freeze(); a.freeze_in_place()

// dict / record
r = {a: 1, b: 2}            // record: dot + index access, fields only
d = dict({a: 1, b: 2})      // dict: index access only, '.' reserved for methods
d.keys(); d.values()
d.filter((k, v) => v > 1)
d.delete("a")               // returns NEW dict, does not mutate; d.delete_in_place("a") mutates
delete(r, "a")              // free function, kept specifically because record has no member functions at all;
                             // pure like d.delete() above — delete_in_place(r, "a") is the mutating free form

// 1-arg vs 2-arg callbacks (map/filter/find/count/all/any/for_each/reduce):
// 1-arg gets the "primary item": value for array/bytes/runes/string/range, KEY for dict
// 2-arg always gets (locator, value): (index, value) for sequences, (key, value) for dict
```

## Value constructors / conversions

```go
int("42")           // 42
int("bad")          // undefined      -- no fallback given
int("bad", 0)        // 0              -- fallback used
decimal("bad")       // decimal(NaN)   -- decimal is the one exception: never undefined
array(3)             // [undefined, undefined, undefined]  -- int arg preallocates (array/bytes/runes only)
dict(42)             // Runtime Error: invalid_argument_type -- dict has no fallback slot
```

`bool`, `byte`, `rune`, `int`, `float`, `decimal`, `time`, `string`, `runes`, `bytes`, `array`, `dict` are all
callable as top-level conversion functions; see [Built-in functions](language.md#built-in-functions) for the
full 0-arg/1-arg/fallback rules and per-type outliers.

## Naming conventions (for the code you write)

```go
snake_case              // all member names
len(); is_empty(); has_prefix(); can_parse_int()   // is_/has_/can_ prefixes for booleans
sort()                                              // non-mutating default; a future mutating variant would be
                                                     // named "..._in_place" by convention, but none exists yet
```

## Gotchas

```go
0.0 == 0                        // true, but 0.0 is TRUTHY -- only int 0 is falsy; NaN is the one falsy float
r = {a: 1}; d = dict({a: 1})
r.a                             // 1                             -- record: '.' is field access
d.a                             // Runtime Error: not_assignable -- dict: '.' is reserved for methods, use d["a"]
a = [1, 2]; b = a               // b aliases a (array is a reference type)
b[0] = 9                        // a[0] is now 9 too -- use copy(a) for an independent array
undefined.a.b.c                 // undefined -- chained access never panics, only the FIRST missing step matters
```

## Quick syntax index

| Want to...                | Syntax |
| -------------------------- | ------ |
| Declare a variable         | `x = 1` (declares if new) / `var x = 1` |
| Force a new local (shadow) | `if x := ...; ... { }` |
| Swap two variables         | `a, b = b, a` |
| Unpack an array            | `a, b = [1, 2]` |
| Loop N times                | `for i = 0; i < n; i++ { }` or `for v in 0..n { }` |
| Loop over a collection      | `for v in c { }` / `for k, v in c { }` |
| Ternary                     | `cond ? a : b` |
| Anonymous function          | `x => x * 2` / `func(x) { return x * 2 }` |
| Partial application (simple)| `foo(1, _, 2)` (same as `x => foo(1, x, 2)`) |
| Ensure cleanup runs          | `defer f.close()` |
| Catch an error               | `defer func() { e = recover(); ... }()` |
| Interpolate a string         | `f"n={n:5d}"` |
| Import a module              | `m = import("name")` |
| Copy a reference type        | `copy(x)` |
| Lock a container immutable   | `immutable(x)` |
| Check a value's type         | `type_name(x)` / `is_int(x)` |

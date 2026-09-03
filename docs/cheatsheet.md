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
| `string`  | `"hi"`, `` `raw` ``, `r"\d+"` | immutable | UTF-8 storage, symbol-indexed    |
| `runes`   | `u"привіт"`             | reference    | mutable symbol array               |
| `bytes`   | `b"hi"`                 | reference    | binary data                        |
| `time`    | `t"2024-01-01"`         | value        | instant in time                    |
| `array`   | `[1, 2, 3]`             | reference    | heterogeneous, ordered             |
| `record`  | `{a: 1, b: 2}`          | reference    | dot + index access, fields only    |
| `dict`    | `dict({a: 1})`          | reference    | index access only (`.` = methods)  |
| `range`   | `1..5`                  | lazy         | see below                          |
| `error`   | `error("msg")`          | value        | payload can be any value           |
| `undefined` | `undefined`           | value        | absence of a value                 |

Reference types (`array`, `runes`, `bytes`, `record`, `dict`, frozen containers) alias on assignment — use `copy()` for an
independent copy. Value types (everything else) copy by value.

```go
type_name(x)     // runtime type name, e.g. "int", "array"
is_int(x); is_array(x); is_callable(x); is_iterable(x); is_immutable(x)   // ... is_T for every builtin type
s.is_valid(); s.is_ascii()   // string/runes/rune: well-formed text? / all below 0x80? (bytes has is_ascii only)

freeze(x)         // deep copy, then deep-immutable -- the source is untouched
freeze_shallow(x) // x's header marked immutable (array/dict/record); shares the body, so `x = freeze_shallow(x)`
                  // to make it stick. Mutating a frozen value raises not_mutable / not_assignable.
```

## Truthiness & equality

```go
// falsy: undefined, false, 0, 0.0, decimal(0), "", [], {}, dict(), range(), the zero time
// every error value is TRUTHY; asking NaN for its truth RAISES (an error state has no truth value)
// two spellings: x.is_true() member, is_true(x) free

1 == "1"      // true  -- '==' coerces to a common type
true == 1     // true
[1] == ["1"]  // true

// bool/byte/rune/int/decimal/float all order against each other now (true < 2, byte(1) < 2.5, etc.)
// -- exact, via math/big.Rat where needed, never a lossy float64 round-trip:
9007199254740993 == float(9007199254740992)   // false -- no silent large-int precision collapse
decimal("0.1") == 0.1                         // false -- float 0.1 isn't exactly a tenth; decimal("0.5") == 0.5 is true
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
"el" in "hello"        // run check      -> true
2 in [1, 2, 3]          // element check  -> true
[2, 3] in [1, 2, 3]     // run check      -> true (an array operand is a contiguous run)
"a" in {a: 1}           // key check      -> true
1.5 in "abc"            // Runtime Error -- an unacceptable operand raises, never a silent false
"value: " + 42          // Runtime Error -- no implicit stringification; use f"value: {42}"
[1, 2] + 3              // [1, 2, 3]     -- array + takes append's reading (only an ARRAY operand spreads)
3 + [1, 2]              // [3, 1, 2]     -- an array on the right means prepend; only + has this mirror
[9] + (1..4)            // [9, range(1, 4)] -- a range is one element; spread it with .array()
[1, 2, 1] - 1           // [2]           -- array - takes remove's (every equal element / every run)
[1, 2] * 3              // [1, 2, 1, 2, 1, 2] -- * is repeat's operator form (count, not element)
"-" * 40                // "----..."     -- same on string/runes/bytes; 3 * "ab" raises (no reflected form)
"ab" + u"cd"             // "abcd"        -- the LEFT operand decides the result type
9223372036854775807 + 1 // Runtime Error -- int overflow raises (int never wraps; byte is the one modular type)
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
for k, v in collection { }        // iterator with index (seq) or key (dict/record; keys in lexical order)

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

// e.kind()            -- "division_by_zero", ... / "user" for raise / "requirement" for require
// e.is_runtime()      -- category: raised by a builtin, member or module
// e.is_user()         -- category: the script's own error(...) / raise(...)
// e.is_requirement()  -- category: the script's own require(...)
// e.value()           -- the payload
// (no is_fatal / is_system: fatal errors never reach a script; the host reads RuntimeError.Fatal)

raise("boom")          // == raise(error("boom")); unwinds until a recover() catches it
error({code: 42})       // build an error value directly (doesn't unwind)

require(cond, payload)  // input check: undefined when cond is true, else raises kind "requirement"
                        // carrying payload untouched -- the script's opening lines
require(amount > 0d, {field: "amount", reason: "must be positive"})

// defer works at the top level of a script too: runs at script end (LIFO), runs when an error
// escapes the body, and a recover() there ends the script normally. Fatal errors skip them.
status = "ok"
defer func() {
    e = recover()
    if e != undefined { status = "failed: " + e.kind() }
}()
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
| `requirement` | a `require(cond, payload)` whose condition was falsy |
| `division_by_zero` | `/` or `%` with a zero divisor |

## Modules

```go
// math_utils.kvn
export { square: func(x) { return x * x } }

// main.kvn
m = import("math_utils")
m.square(4)   // 16
```

Builtin modules: `fmt`, `math`, `os`, `regexp`, `times`, `json`, `base64`, `hex`, `rand`.

```go
fmt = import("fmt");     fmt.println("sum:", 20 + 22)
math = import("math");   math.sqrt(144)
re = import("regexp");   re.re_match("[0-9]+", "abc123")
json = import("json");   json.encode({a: 1})
times = import("times"); times.now()
rand = import("rand");   rand.int_n(100)
```

## Strings, f-strings & format

```go
"hello" + " " + "world"          // concatenation
// no printf verbs ("Hi %s" is just text) -- use f-strings or format() instead

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
len(x); copy(x); is_true(x); is_view(x)   // is_view: is THIS header borrowing another's body?
min(3, 1, 2); max(3, 1, 2)  // 1; 3 -- variadic selection over ARGUMENTS; min() raises; min(arr...) == arr.min()

// array
a = [3, 1, 2]
a.sort(); a.sort_in_place()        // pure member vs mutating twin (twin returns the receiver)
a.keep(x => x > 1)               // [3, 2]; a.remove(x => x > 1) is the drop side
a.map(x => x * 2)                  // [6, 2, 4] (1:1); a.flat_map(x => [x, x]) concatenates
a.reduce(0, (acc, v) => acc + v)   // 6
a.index(2); a.index_last(2)        // locators; a miss answers undefined (never -1), or a trailing default
a.contains(2); a.count(2); a.any(fn); a.all(fn)
a.reverse(); a.dedup(); a.unique(); a.flatten(); a.chunk(2)
a.sum(); a.avg(); a.min(); a.max(); a.first(); a.last()   // aggregation; an EMPTY receiver RAISES --
                                                          // pass the trailing default to answer instead: a.first(0)
a.append([4, 5])            // [3, 1, 2, 4, 5] -- an ARRAY argument SPREADS (like +); nothing else does
a.push([4, 5])              // [3, 1, 2, [4, 5]] -- push never spreads: one element per argument
a.prepend(0); a.push_first(0)      // front forms; arguments stay in order
a.insert(1, 9)              // element insert at a position (raises out of range); a.splice(i, del, ...) edits
a.trim(); a.trim(0)         // drop leading/trailing blanks (undefined + zeros) / your own element set
a.has_prefix([3, 1]); a.remove_prefix([3, 1]); a.replace(1, 9); a.pad_end(5)
a.slice(1, 3)               // clamps; a.slice_view(1, 3)/a.chunk_view(2) share storage (is_view(x) tells)
a.copy_shallow(); a.freeze(); a.freeze_shallow()
// every mutating member has the _in_place suffix and raises kind "not_mutable" on a frozen receiver

// dict / record
r = {a: 1, b: 2}            // record: dot access, fields only, NO member functions (free builtins serve it)
d = dict({a: 1, b: 2})      // dict: index access d["a"]; a dict is a set of KEYS with attached values
d.keys(); d.values()        // key order on a map is LEXICAL everywhere -- iteration, members, render, encode
d.keep((k, v) => v > 1)   // predicate: f/1 gets the KEY, f/2 gets (key, value)
d.remove("a", "b")          // key set; d.remove_in_place(...) mutates
d.merge(dict({c: 3}))       // the whole add side (variadic, last wins); merge_in_place mutates
d.map((k, v) => v * 10)     // transforms the ATTACHMENT, keys fixed
remove(r, "a")              // free forms serve record: len/copy/freeze/format/is_true/remove/is_view
for k in d { }              // yields KEYS in lexical order (dict and record alike); for _, v in d for values

// 1-arg vs 2-arg callbacks (map/keep/remove/index/count/all/any/for_each):
// f/1 gets the element (dict: the key); f/2 gets (index, element) (dict: (key, value))
// for_each makes a FULL pass, ignores the callback's return, and returns the receiver
```

## Value constructors / conversions

```go
int("42")           // 42
int("bad")          // Runtime Error -- a failed conversion RAISES, never a silent undefined
"bad".int(0)         // 0             -- the MEMBER form's trailing default is the explicit opt-out (free form takes none)
decimal("bad")       // Runtime Error -- parse always raises on invalid input, for every type
array(0, 3)          // [0, 0, 0]     -- T(x, count): n copies of x, kept whole (array/string/bytes/runes only)
dict([["a", 1]])     // dict({"a": 1}) -- the entries reading: each element is exactly [key, value]
array(a)             // a NEW array   -- T(x) on an x of type T constructs: shallow copy, always mutable
a.array()            // a NEW array   -- ... and so does the member spelling; NO conversion ever aliases
bytes(b"ab")         // bytes([97,98]) -- so this is how a constant literal becomes a writable buffer
(5).int(0)           // 5             -- a same-type default is unreachable but accepted, on every type
d.record_view()      // shares d's map -- the `_view` family is the ONLY shared storage (is_view(x) tells)
```

Literals come in two kinds: `"..."`, `u"..."`, `b"..."` and the scalars are **compile-time constants**, one
shared immutable value (`type_name(b"ab")` is `"immutable-bytes"`); `[...]` and `{...}` hold expressions, so
they are **built at run time** — a fresh, mutable body per evaluation. `bytes`/`runes` are mutable *types*:
only their literal form is constant.

`bool`, `byte`, `rune`, `int`, `float`, `decimal`, `time`, `string`, `runes`, `bytes`, `array`, `dict` are all
callable as top-level conversion functions; see [Built-in functions](language.md#built-in-functions) for the
full 0-arg/1-arg/count rules and per-type outliers.

An `int` next to a `time` means two different things, decided by position — **operator: a duration in
nanoseconds; conversion: a unix timestamp**. There is no `time` vs `int` ordering or equality; convert first.

```go
t + 1000000000        // +1 second   -- operator position: nanoseconds
t2 - t                // nanoseconds between the two instants
time(1704067200)      // conversion position: unix SECONDS
(1704067200123).time_ms()          // ... milliseconds (JS Date.now(), Java currentTimeMillis())
time(1704067200.5)                 // ... sec.frac -- float is lossy sub-second
time(1704067200.123456789d)        // ... sec.frac -- decimal is exact to nanoseconds
t.unix() / t.unix_ms() / t.unix_micro() / t.unix_nano()   // out, same four encodings
t.unix_nano().time_nano() == t     // true -- the only pair that round-trips sub-second exactly
t < 1704067200        // Runtime Error: invalid_binary_operator -- which role would the int be?
t < time(1704067200)  // say it explicitly instead
```

## Naming conventions (for the code you write)

```go
snake_case                        // all member names
len(); is_empty(); has_prefix()   // is_/has_ prefixes for booleans
sort() / sort_in_place()          // non-mutating default; the _in_place twin mutates and returns the receiver
push_first; index_last; trim_start; pad_end   // qualifiers are SUFFIXES; ends are start/end, never left/right
upper(), string(), array()        // no to_ prefix: the receiver already says what converts
```

## Gotchas

```go
rows.append(row)                // SPREADS the row's cells in (an array operand is a run) -- use rows.push(row)
r = {a: 1}; d = dict({a: 1})
r.a                             // 1             -- record: '.' is field access
d.a                             // Runtime Error -- dict: '.' is reserved for methods, use d["a"]
a = [1, 2]; b = a               // b aliases a (array is a reference type)
b[0] = 9                        // a[0] is now 9 too -- use copy(a) for an independent array
for k in dict({a: 1}) { }      // k is the KEY -- a map's element is its key; use for _, v for values
"ім'я".len()                    // 4 -- symbols, not bytes; but symbols are code points, not grapheme clusters
bytes([97,255]).string().bytes() // bytes([97,255]) -- byte<->text conversion is TOTAL and lossless: an octet
                                //   that is not a symbol becomes its escape (U+DC80..DCFF); is_valid() finds it
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
| Lock a container immutable   | `freeze(x)` (deep) / `freeze_shallow(x)` (header only) |
| Check a value's type         | `type_name(x)` / `is_int(x)` |

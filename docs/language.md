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

Every value in Kavun is shared, not copied, when you assign it to a new name or pass it as a function argument —
the new name and the original refer to the very same value. Mutating that value — through indexed assignment
(`a[0] = x`) or an `_in_place` method — is visible through every name that shares it, including a function's own
parameter name reaching back to the caller's value. Plain `x = ...` is never a mutation, for any type: it always
rebinds the name `x` to a new value, so afterward `x` no longer shares anything with whatever it held before.
Scalars (`int`, `bool`, `string`, ...) simply have no mutating operations at all, so none of this is ever
something you can actually observe for them — from your perspective they behave exactly as if they were copied.

Literal examples:

```go
i = 42
f = 3.14
d = 1.23d
c = 'A'              // rune (Unicode code point)
bc = b'A'            // byte (single-byte literal)
s = "hello"          // string, double-quoted
rs = u"привіт"       // runes (unicode string), u"..." syntax
bs = b"hello"        // bytes, b"..." syntax
ts = t"2024-01-01"   // time, t"..." syntax
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

### Byte literals

Kavun supports explicit single-byte literals with `b'...'` syntax:

```go
b'A'        // byte(65)
b'\x00'     // byte(0)
b'\n'       // newline byte
```

The content must resolve to exactly one byte (`0..255`). Empty literals, multi-character contents, and Unicode code
points above 255 are syntax errors. Hex literals such as `0x41` remain `int` literals.

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

### Parallel assignment

When both sides name the same number (2 or more) of items, `=`/`:=` assigns them positionally — the left-hand side
must be plain identifiers (or `_`); the right-hand side can be any expressions:

```go
a, b := 1, 2          // a=1, b=2
a, b = b, a           // swap: every right-hand expression is evaluated before any left-hand target is stored
a, _, c := 1, 2, 3     // '_' discards a position, same convention as destructuring below
```

A left/right count mismatch (and neither side has exactly one item, which would instead be destructuring — see
below) is a compile-time error, since both counts are known statically.

### Destructuring assignment

When the left-hand side names 2 or more targets and the right-hand side is a single expression, `=`/`:=`
de-structures that value instead of performing an ordinary assignment. A single name (`a = expr`) is always an
ordinary assignment — destructuring syntax adds no new punctuation, it is purely a function of how many names are
on the left and how many expressions are on the right.

```go
a, b, c := [10, 20, 30]     // array unpack: positional - a=10, b=20, c=30
a, b := {a: 1, b: 2, c: 3}  // dict/record unpack: keyed by name - a=1, b=2 ("c" ignored)
```

Destructuring targets must be plain identifiers (or `_`, see below) — selectors and indexed targets
(`a.b, c = ...`) are not supported.

**Array unpack** is positional: the right-hand side must be at least as long as the number of targets, or the
statement fails the same way `arr[i]` fails on an out-of-range index. Extra elements beyond the number of targets
are ignored:

```go
a, b := [1, 2, 3]       // a=1, b=2 - the 3rd element is never accessed
a, b, c := [1, 2]       // runtime error: index_out_of_bounds - there is no element for c
```

**Dict/record unpack** is keyed by name: each target's identifier is looked up as a key in the right-hand side. A
missing key fills the target with `undefined` rather than erroring — the same behavior as indexing a dict/record
with a missing key (`d["missing"]`). Keys present in the value but not named on the left are ignored:

```go
a, b := {a: 1, c: 2}   // a=1, b=undefined - "b" is not a key of the right-hand side
```

**`_` discards a target.** It is never a real variable — it cannot be read back, and it may appear more than once
in the same statement. Any other repeated name on the left is a compile error. For array unpack, `_` still consumes
a position (and is still subject to the out-of-range check); for dict/record unpack, `_` performs no lookup at all
and never requires a corresponding key to exist:

```go
a, _, c := [1, 2, 3]     // a=1, c=3 - position 1 is still bounds-checked
a, _ := {a: 1}           // a=1 - no key lookup happens for the "_" position
a, c, c := [1, 2, 3]     // compile error: 'c' used more than once in destructuring assignment
```

This is also why a plain `_ = expr` / `_ := expr` (a single discarded target, not destructuring at all) is a no-op:
the right-hand side is still evaluated for any side effects, but nothing is stored, and `_` can never be read back
afterward.

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
example, `x := 10` declares a new local variable `x` confined to the if block scope. The outer `x` remains unchanged.

#### Headers share scope with their own block

A *header* — a function/lambda's parameter list and named result, an `if`'s init clause, a `for`'s init clause, or a
`for ... in` loop's key/value names — shares **one scope** with the block(s) it introduces. A name declared by a
header cannot be redeclared with `:=` or `var` directly in that block; use `=` to reassign it instead:

```go
if x := 10; x > 5 {
    x = 0      // OK: reassignment
    x := 0     // Compile Error: 'x' redeclared in this block
}

for i := 0; i < 3; i++ {
    i = 0      // OK: reassignment
    i := 0     // Compile Error: 'i' redeclared in this block
}

func(x) {
    x = 0      // OK: reassignment
    x := 0     // Compile Error: 'x' redeclared in this block
}

for k, v in collection {
    k = 0      // OK: reassignment
    k := 0     // Compile Error: 'k' redeclared in this block
}
```

An `if` with both a `then` block and an `else` block shares its header with **both** — reusing the header's name with
`:=` is an error in either one:

```go
if x := 0; x == 0 {
    x := 1     // Compile Error: 'x' redeclared in this block
} else {
    x := 2     // Compile Error: 'x' redeclared in this block
}
```

A block nested **one level deeper** than a header's own block is always a fresh scope, free to reuse the header's
name — whether through an ordinary `:=`, or through its own nested header (a nested `if`/`for`'s own init clause, an
`else if`, or a nested function's own parameter list):

```go
if x := 10; true {
    if true { x := 20 }          // OK: fresh scope one level deeper
    if x := 20; true { y = x }   // OK: the nested if's own init reopens the name
} else if x := 30; true {
    y = x                        // OK: else-if is its own if-statement, not a
}                                 // continuation of the outer one

func(x) {
    if true { x := 99 }          // OK: fresh scope one level deeper
}
```

This only applies to a name a header actually declares. A header is always free to shadow some *other*, unrelated
outer variable of the same name — that's ordinary lexical shadowing, unaffected by the rule above:

```go
x := 1
if x := 2; true {
    x = 10     // reassigns the if-scoped x, not the outer one
}
out = x        // 1: the outer x was never touched
```

### Shadowing and reassigning builtins

Builtin functions (`len`, `copy`, `int`, `string`, etc.) behave like pre-seeded global values: they may be shadowed
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

One exception: the `low..high` range literal (see [Range literals](#range-literals)) is sugar for `range(...)` but is
not itself a reference to the identifier `range`, so reassigning `range` has no effect on it — only on `range(...)`
written out explicitly.

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
| 3.5   | `..` (range literal, see below)           |
| 4     | `+` `-` `\|` `^`                          |
| 5     | `*` `/` `%` `<<` `>>` `&` `&^`            |

Unary operators: `-`, `+`, `!`, `^` (bitwise complement). Ternary `?:` binds looser than all binary operators. The
range literal `..` binds looser than arithmetic but tighter than comparison, so `1..n+1` means `1..(n+1)` and
`1..n < 5` means `(1..n) < 5` (a runtime error, since a range isn't comparable with `<` — wrap the range in `.array()`
or similar first if you need that).

### Complete operator list

| Category                   | Operators                                                  |
| -------------------------- | ---------------------------------------------------------- |
| Arithmetic and bitwise     | `+` `-` `*` `/` `%` `&` `\|` `^` `<<` `>>` `&^`            |
| Comparison and logical     | `==` `!=` `<` `<=` `>` `>=` `&&` `\|\|` `!`                |
| Membership and conditional | `in` `not in` `?:`                                         |
| Range literal               | `..` `..:`                                                 |
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

### Range literals

`low..high` and `low..high:step` are syntax sugar for `range(low, high)` / `range(low, high, step)` (see
[Built-in functions](#built-in-functions)) — `step` defaults to whatever the builtin itself defaults to (currently
`1`), not something the syntax hardcodes. `low` and `high` can be any expression, not just literals, and the range
is exclusive of `high`, matching `range()`, Python's `range()`, and Go slice semantics — see
[Range/Slice Bounds](conventions.md#rangeslice-bounds-inclusive-start-exclusive-stop) for why this was a deliberate
choice rather than an inherited default:

```go
1..5           // range(1, 5)      -> 1, 2, 3, 4
5..1           // range(5, 1)      -> 5, 4, 3, 2 (direction is auto-detected)
1..5:2         // range(1, 5, 2)   -> 1, 3
n := 3
1..n+2         // range(1, 5)      -> '..' binds looser than '+', so this is 1..(n+2), not (1..n)+2
```

Inside index brackets, `..` is an alternate spelling of the `:` low/high separator — `arr[low..high]` and
`arr[low:high]` parse to the exact same slice, and both accept the usual omitted bounds (`arr[..3]`, `arr[1..]`).
The step separator is always `:`, in both spellings:

```go
a := [10, 20, 30, 40, 50]
a[1..3]      // same as a[1:3]      -> [20, 30]
a[1..4:2]    // same as a[1:4:2]    -> [20, 40]
a[..3]       // same as a[:3]       -> [10, 20, 30]
```

A bare range literal (outside of brackets) has no container to default a missing bound against, so unlike a slice,
both `low` and `high` are required — `5..` and `..5` on their own are parse errors.

`low..high` is a language construct, not a reference to the identifier `range` — the same way a `b"..."` bytes
literal never references the identifier `byte`. That means reassigning `range` in scope has no effect on it, unlike
writing `range(...)` yourself:

```go
range := func(a, b) { return a + b }
1..5           // still the real range(1, 5) — unaffected by the reassignment above
range(1, 5)    // 6 — calls the reassigned function, like any other shadowed builtin
```

Accessing any field or index on `undefined` returns `undefined`:

```go
undefined.x         // undefined
undefined[0]        // undefined
undefined.a.b.c     // undefined
```

### Placeholder syntax (`_`)

A bare `_` used as a direct operand of a call, method call, field selector, unary/binary operator, index, slice, or
ternary is pure syntax sugar for an arrow lambda: each distinct `_` becomes a fresh parameter (left to right), and
the enclosing node becomes the lambda body.

```go
add := func(a, b, c) { return a + b + c }
w := add(1, _, 3)    // same as: x => add(1, x, 3)
w(10)                // 14

_(2, 5)              // same as: f => f(2, 5) -- '_' can be the callee too
_.name               // same as: x => x.name
_ + _                // same as: (x, y) => x + y
arr[_]               // same as: x => arr[x]
_ ? "yes" : "no"     // same as: x => x ? "yes" : "no"
```

**Scope is exactly the closest qualifying node** — `_` nested inside a deeper qualifying node binds to that inner node
only and never bubbles out:

```go
foo(1, bar(_, 2))    // '_' binds to bar(_, 2), NOT to foo -- equivalent to foo(1, (x => bar(x, 2)))
```

One consequence of that rule, worth knowing rather than working around: `foo(_) + 1` binds `_` to the call
`foo(_)` only, producing `(x => foo(x)) + 1` — an operator applied to a function value, which fails at runtime.
If you want the whole arithmetic expression to be the lambda body, write it by hand: `x => foo(x) + 1`.

#### Design principle: simplicity over reach

This is deliberately a sugar for the **simple, single-application case only** — it has no notion of expression
boundaries, and every `_` is a distinct parameter (no aliasing: `foo(_, _)` is always a 2-arg function, never "the
same value twice"). For anything that needs an explicit boundary, reuses a value, or spans more than one
call/operator, write the arrow lambda directly — that's what it's for; `_` only elides the cases where writing
`x => ...x...` by hand would be pure ceremony.

This is a considered trade-off, not an accident of the implementation: every extension to *where* `_` is
recognized was evaluated against "does the reader still know what this means without mentally re-running the
desugaring rule?" A placeholder that could reach outward through parentheses, into composite literals, or stand in
for a method/field *name* would each individually still be mechanical to define — but stacked together they turn
`_` from "obviously an elided lambda parameter" into a second, silent expression-rewriting language layered on top
of the visible one, which is exactly the kind of implicit, hard-to-statically-read code Kavun's non-expert-
readability and auditability goals (see [purpose & conventions](conventions.md)) are meant to rule out. Each
boundary below exists for that reason, not because it was hard to implement.

#### What's covered, and the explicit exceptions

| Expression | De-sugars to | Notes |
| --- | --- | --- |
| `x[_]` | `w => x[w]` | receiver `x` is a normal captured variable; only the index is a param |
| `_[1]` | `w => w[1]` | receiver is the param; index is fixed |
| `_[_]` | `(a, b) => a[b]` | both slots are placeholders → 2 params, in slot order `(Expr, Index)` |
| `_(1)` | `f => f(1)` | the **callee** is a placeholder slot on `Call`, same as any argument |
| `_(_)` | `(f, x) => f(x)` | callee and argument both placeholders → a general 2-arg "apply" falls out for free |
| `f(1, _...)` | `w => f(1, w...)` | `_` as the whole spread source is supported — the param is bound to the iterable, not to one of its elements |
| `x._` | *(untouched)* | plain field access; `x` isn't a placeholder, so nothing to rewrite |
| `_._` | `w => w._` | receiver is the param; **the field name `_` itself is never a placeholder target** — see below |
| `_._(_, _)` | `(a, b, c) => a._(b, c)` | de-sugars fine (3 placeholders: receiver + 2 args) — the method **name** is untouched; this fails at *call* time unless the receiver actually has a callable field/method literally named `_` |
| `(_) + 1` | *(untouched)* | a parenthesized `_` is not detected — the placeholder must be bare |
| `[1, _, 3]` | *(untouched, compile error)* | `_` inside an array/record **composite literal** is not a placeholder position at all — `Array`/`Record` aren't in the rewrite's node list, so this is a plain unresolved-reference error, same as `_` anywhere else outside a qualifying node |
| `func(x){ return x }(_)` | `w => (func(x){return x})(w)` | well-defined but degenerate — wrapping a lambda you just wrote adds a layer without changing behavior |

**Method and field names are never placeholder targets — this is an explicit exception to the general rule, not
an oversight.** `x.name` and `x.name(...)` parse the `name` part as a compile-time string baked into the AST at
parse time (`Selector.Sel` is a string literal node; `MethodCall.MethodName` is a plain Go string, not an
expression at all) — there is no runtime-evaluated slot there for a placeholder (or anything else) to fill, the
same reason Kavun has no `x.(someVariable)` syntax at all, independent of this feature. Dynamic lookup-by-name
already has a home in the language: `dict`'s bracket indexing (`d[key_expr]`) is exactly "access by a
runtime-computed key", which is why `dict` reserves `.` for methods and forces `[]` for keys (see [Types at a
glance](../docs/cheatsheet.md)). Even setting the structural reason aside, `_` is already a legal literal field
name (`{_: 99}`), so overloading it to also mean "placeholder" in name position would collide with real programs
— another reason this boundary is permanent, not a "not yet".

`_` cannot appear as the target of a variadic spread **element** (there is no such notion — spread has no
"elements" to place a hole in); it *can* be the entire spread source, as shown in the table above. Discard-`_` in
assignment/destructuring targets is unaffected by any of this — the two uses are disambiguated by syntactic
position (assignment target vs. expression operand), the same way `.` means different things in field access vs.
statement continuation.

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

Arguments follow the same shared-body rule as any other assignment (see "Builtin types overview" above): passing
a container to a function shares it with the caller, so mutating it through the parameter name is visible to the
caller too.

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

This section is the complete, authoritative list of global built-in functions (everything callable without an
`import(...)`). For the full list of *member* functions on each type (e.g. `[1,2,3].sort()`), see
[Detailed type documentation](#detailed-type-documentation) below — this section only covers top-level functions.

### Value constructors / conversions

`bool`, `byte`, `rune`, `int`, `float`, `decimal`, `time`, `string`, `runes`, `bytes`, `array`, and `dict` are all
callable as top-level functions named after the type. Most of them (every one except `dict`) share one convention:

- **0 args** — returns the type's zero value (`int()` → `0`, `bool()` → `false`, `array()` → `[]`, etc.).
- **1 arg, already the target type** — returned unchanged (no copy; same reference for reference types).
- **1 arg, any other type** — converted via that type's internal `AsBool`/`AsInt`/`AsArray`/... hook. Whether this
  succeeds depends entirely on the *argument's* type, not the function you called — see each type's own
  "Conversion Functions" section (e.g. [array](types/array.md#conversion-functions),
  [bytes](types/bytes.md#conversion-functions)) for what converts into what.
- **2 args** — the second argument is a fallback: if the 1-arg conversion would have failed, the fallback is
  returned instead of `undefined`.

```go
int("42")             // 42
int("bad")            // undefined  <- conversion failed, no fallback given
int("bad", 0)         // 0          <- conversion failed, fallback used
float("3.14")         // 3.14
string(99)            // "99"
string(undefined)     // undefined  <- not the string "undefined"
bool(0)               // false
bool(0.0)             // true  <- float zero is truthy
byte(65)              // byte(65)
byte(999)             // undefined  <- out of byte range (0-255)
runes("abc")          // runes value
bytes("abc")          // bytes value
time("2024-01-01")    // time value
rune(0)               // rune 0
```

`decimal` is the one exception to "conversion failure with no fallback returns `undefined`": it **never** returns
`undefined`. Unlike `int`, `bool`, `rune`, etc. — which have no way to represent "invalid value" and so must fall
back to the generic `undefined` — `decimal` has its own valid in-band state for exactly this case: `decimal(NaN)`,
checkable with `.is_nan()` and inspectable with `.error_details()`. A failed conversion with no fallback routes
through that state instead: `decimal(NaN)` for an unparsable string/runes, or `decimal(0)` for most other
non-convertible types (e.g. `undefined`, which has no textual form to fail parsing at all). This is specific to
`decimal`'s constructor, not a general rule for any type with a `NaN`-like value — `float` also has a `NaN` state
(`0.0 / 0.0`), but `float("bad")` still returns `undefined`; `float()`'s constructor doesn't route failures through
it the way `decimal()`'s does. See [decimal's conversion rules](types/decimal.md#conversion-rules).

```go
decimal("1.25")              // decimal(1.25)
decimal("bad")               // decimal(NaN)  <- NOT undefined
decimal("bad", decimal(0))   // decimal(0)    <- fallback still works and takes priority
decimal(undefined)           // decimal(0)
```

#### Preallocating a container: `array(n)`, `bytes(n)`, `runes(n)`

`array`, `bytes`, and `runes` additionally special-case a single **`int`** argument: instead of attempting a
conversion, it preallocates a zero-filled buffer of that length. This is different from every other constructor,
where an int argument goes through the normal conversion path (`string(42)` produces `"42"`, the text
representation — it does **not** produce a 42-character buffer).

```go
array()                // []
array(3)               // [undefined, undefined, undefined]
array([1, 2])          // [1, 2]         <- passthrough, already an array
array(range(1, 4))     // [1, 2, 3]      <- converted via range's AsArray
array(true, [9])       // [9]            <- bool isn't convertible, fallback used

bytes()                // bytes([])
bytes(3)               // bytes([0, 0, 0])
bytes("abc")           // bytes("abc")   <- converted via string's AsBytes

runes()                // runes("")
runes(3)               // runes of 3 NUL runes
runes("abc")           // runes("abc")
```

`array(n)`/`bytes(n)`/`runes(n)` require `n >= 0`; a negative size raises a recoverable `invalid_value` error rather
than succeeding or crashing.

`runes(x)` converts from a much wider set of source types than `bytes(x)`/`array(x)`: the default `AsRunes`
fallback goes through `AsString`, so anything with a string representation (numbers, `bool`, `time`, ...) converts
successfully, whereas `bytes`/`array` only convert from types that implement `AsBytes`/`AsArray` explicitly
(`string`, `runes`, `array`-of-byte-ish values, `range`, ...).

#### `dict`/`record` are outliers

`dict()` does not follow the shared convention above, and `record()` — its mirror-image counterpart, converting
the other direction — follows the same outlier shape:

- `dict()` / `record()` — empty dict / record.
- `dict(d)` / `record(r)` where the argument is already the target type — returned unchanged.
- `dict(r)` where `r` is a `record`, and `record(d)` where `d` is a `dict` — converted to an independent
  **shallow** copy: a fresh top-level container, but nested values are not recursively cloned (same convention
  every other type's own `.record()`/`.dict()` conversion already follows). Mutating the result's own keys never
  affects the source's keys, or vice versa; mutating a *nested* container reachable from both still affects both
  (shallow, not deep). Result is always mutable, regardless of the source's mutability — same convention as
  `copy()`.
- `dict_view(r)` / `record_view(d)` are the explicit `_view` twins: share the source's underlying storage
  directly instead of copying — both the top-level key set and nested values are the *same* backing map, so
  mutating either side through either wrapper is visible through both. Result's mutability is inherited from
  the source's (shares its immutability, not always-mutable like the copying form). Maximum performance when
  you've confirmed nothing else needs to observe the source independently.
- `dict(x)` / `record(x)` for any other type — **raises a runtime error** (`invalid_argument_type`) instead of
  returning `undefined`. A second argument is not accepted as a fallback in either case — neither has a fallback
  slot at all (unlike every constructor above, they never silently swallow an unconvertible argument).

The same member-call spellings exist on the relevant receiver: `dict_val.record()` / `dict_val.record_view()`.
`record` has no member functions at all (no `MethodCall` switch — see `docs/types/container-semantics.md`), so
`record_val.dict()`/`record_val.dict_view()` don't exist; `dict(record_val)`/`dict_view(record_val)` are its only
spellings for that direction.

```go
dict()                 // dict({})
dict({a: 1})            // dict({"a": 1})   <- from record, independent shallow copy
record(dict({a: 1}))    // {"a": 1}         <- from dict, independent shallow copy
dict_view({a: 1})       // dict({"a": 1})   <- shares storage with the record literal
dict(42)                // Runtime Error: invalid_argument_type
```

### Collections and helpers

```go
len(x)                                       // length of collection/string/range
copy(x)                                      // deep mutable copy
copy_shallow(x)                              // shallow mutable copy (top level only)
delete(obj, "key")                           // returns obj without "key"; does not mutate obj
delete_in_place(obj, "key")                  // mutates record/dict in place
freeze(x)                                    // deep copy, then deep-immutable; source untouched
freeze_shallow(x)                           // x, header marked immutable; needs `x = freeze_shallow(x)` to stick
range(0, 10)                                 // range(start, stop[, step]) — sugar: 0..10, 0..10:step
min(a, b, ...); max(a, b, ...)               // smallest/largest argument (see below)
error("msg")                                 // error value with a string payload
error({code: 42})                            // error value with a structured payload
raise(err)                                   // raise an error so a deferred recover() can catch it
recover()                                    // inside a deferred function, return & clear the in-flight error
type_name(x)                                 // runtime type name
format(template, args)                       // runtime f-string-style formatting (see below)
```

`copy`/`copy_shallow`/`delete`/`delete_in_place`/`freeze`/`freeze_shallow` are kept as free functions (rather
than retired in favor of a member-only spelling) specifically because `record` has no member functions at all —
these six are `record`'s only way to copy itself, remove a key, or become immutable. Every other type that
supports these operations (`array`, `bytes`, `runes`, `dict`, plus every scalar for `copy`/`copy_shallow`/
`freeze`/`freeze_shallow`) has member-call equivalents too: `x.copy()`, `x.copy_shallow()`, `x.freeze()`,
`x.freeze_shallow()`, `dict_val.delete(key)` / `dict_val.delete_in_place(key)`. `append`/`splice` have no such
gap (`record`/`dict` don't support either operation at all, and `array`/`bytes`/`runes` have full member-call
coverage), so their free-function forms were retired outright — use `x.append(...)` / `x.append_in_place(...)`,
`x.splice(...)` / `x.splice_in_place(...)` (all four work on `array`, `bytes`, and `runes` alike). `append`/
`splice` are pure — an independent result, source unchanged, works regardless of the receiver's mutability;
`append_in_place`/`splice_in_place` are the explicit mutating twins. `freeze`/`freeze_shallow` are a different
shape from `delete`/`delete_in_place`, because `Immutable` lives on the `Value` header, not the shared body:
`freeze(x)` always detaches first (`copy`'s deep-clone behavior) before marking the fresh clone immutable, so it
never affects any other binding that shares `x`'s body, and the result is captured the normal way
(`y := freeze(x)`). `freeze_shallow(x)` returns `x` with its own header's immutable flag set, **without**
mutating any shared storage — like every member-call `_in_place` twin, the caller must reassign
(`x = freeze_shallow(x)`) to see the effect on their own variable, and a pre-existing sibling binding that
never gets reassigned stays independently mutable — mutating through it is still visible through the "frozen"
variable too, since both still point at the same body.

Unlike the constructors above, `error(...)` requires **at least one** argument (there is no zero-value error — an
empty error carries no information) and `range(...)` requires **at least two** (`start`, `stop`; `step` is
optional and must be `> 0`, otherwise it raises a recoverable error).

`min`/`max` are variadic over **0 or more** arguments and compare pairwise via the same ordering operators as `<`/`>`
(so they work on any type that supports comparison — numbers, strings, decimals, times, ... — not just numbers):

- **0 args** — `undefined` (there's nothing to compare, and unlike `math.min`/`math.max` there's no type-generic
  "identity" value to fall back to across arbitrary comparable types).
- **1 arg** — that argument, unchanged (no comparison performed).
- **2+ args** — the smallest/largest argument, by repeatedly applying `<`/`>`.

```go
min()                  // undefined
min(5)                 // 5
min(3, 1, 2)           // 1
max(3, 1, 2)           // 3
min("banana", "apple") // "apple"
```

There is deliberately no special case for a single array/container argument — spread it instead, which composes
with the same 0/1/2+ rule above so `min(arr...)` always agrees with `arr.min()`:

```go
min([3, 1, 2]...)   // 1, same as [3, 1, 2].min()
min([]...)          // undefined, same as [].min()
min([7]...)         // 7, same as [7].min()
```

This is a different, narrower contract than `math.min(x, y)`/`math.max(x, y)` (see [stdlib.md](stdlib.md)), which
are strictly 2-arg and `float`-only.

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

### Type predicates

`is_bool`, `is_byte`, `is_rune`, `is_int`, `is_float`, `is_decimal`, `is_string`, `is_runes`, `is_bytes`, `is_array`,
`is_record`, `is_dict`, `is_range`, `is_time`, `is_error`, `is_undefined`, `is_function`, `is_callable`,
`is_iterable`, `is_immutable`

Each `is_T` predicate (except `is_function`/`is_callable`/`is_iterable`/`is_immutable`) checks the value's *exact*
runtime type — no coercion, and no "is-a" relationship between related types (e.g. `is_int(byte(1))` is `false`).

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

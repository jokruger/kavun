# error

A failure as a first-class value.

## Overview

An `error` is a value that carries a payload describing a failure. Errors reach a script two ways:

- **constructed** — `error(payload)` wraps any value; the script raises it itself with `raise(...)`;
- **raised by the runtime** — division by zero, a bad index, a failed conversion, … — and caught with
  `recover()` inside a deferred function.

Every error is **truthy** — an error without a payload is still an error, and there is no zero error
(`error()` with no arguments is itself a runtime error: `expected 1 or 2 argument(s)`). Each error carries a
machine-readable `kind()` tag, a `value()` payload, and a **category** saying where it came from, read with
`is_runtime()` / `is_user()` / `is_requirement()`.

### Categories

Every kind belongs to exactly one category — the error's origin, an axis independent of what went wrong:

| category | predicate | raised by | reaches a script? |
| --- | --- | --- | --- |
| runtime | `is_runtime()` | a builtin, a member function or a stdlib module | yes |
| user | `is_user()` | the script's own `error(...)` / `raise(...)` | yes |
| requirement | `is_requirement()` | the script's own `require(cond, payload)` | yes |
| system | — | the VM itself: stack overflow, an internal invariant, a host defect | **never** |

A system error is always fatal, so it bypasses every `recover()` and only ever reaches the host — which is why
there is no `is_system()` to call. The host reads the category from `RuntimeError.Category` (see
[embedding](../embedding.md#handling-errors)).

## Construction

```go
error("boom")               // payload "boom", kind "user"
error(42)                   // any payload type
error({code: 404})          // including containers
error(undefined)            // legal: the payload-less error — still truthy
error(error("inner"))       // error("inner") — an error argument is never re-wrapped
error("boom", true)         // second argument: fatal flag — see below
```

`error(x)` **wraps** its argument — it is a constructor, not a conversion, so there is no `.error()` member
on other types and no conversion *from* `error` back into a payload type (see
[Exclusions](#excluded-members)).

**An argument that is already an error is answered as-is**, never wrapped a second time: `error(err)` is
`err`. A bare re-wrap only ever added an unlabelled layer, and it relabelled the payload's own diagnosis —
a caught runtime error came back out as kind `"user"` in the user category. The severity form keeps the
payload and kind and changes only the flag, which is exactly what `raise(err, fatal)` already did:

```go
e = error("boom")
error(e) == e               // true — the same error, not a nested one
error(e).value()            // "boom", a string — not an error
error(e, true)              // fatal; kind() and value() are still "user"/"boom"

// a runtime error keeps its own diagnosis through the constructor
c = /* a caught invalid_method error */
error(c).kind()             // "invalid_method" — not relabelled to "user"
error(c).is_runtime()       // true — the category travels with the error too
```

Deliberate nesting is spelled with a payload that **names** the cause, which is the annotated chain a bare
re-wrap could never express:

```go
error({msg: "loading config failed", cause: e})   // the chain, labelled
```

The optional second argument is a `bool` fatal flag. A fatal error is **not catchable**: `raise(error("x",
true))` unwinds past every `recover()` and terminates the script. Use it for states no handler should paper
over.

## Raising and catching

`raise(x)` raises. A non-error argument is wrapped first — `raise("text")` and `raise(42)` raise a kind
`"user"` error with that payload. `recover()`, called inside a deferred function, answers the in-flight error
and stops the unwinding; with nothing in flight it answers `undefined`.

```go
fmt := import("fmt")

safe_div := func(a, b) res {
    defer func() {
        e := recover()
        if e != undefined {
            fmt.println("caught: ", e.kind())   // "division_by_zero"
            res = 0
        }
    }()
    res = a / b
}

safe_div(10, 2)   // 5
safe_div(10, 0)   // prints "caught: division_by_zero", answers 0
```

Note the **named result** (`res`): a plain local assigned inside the deferred function would be lost — the
named result is how a handler substitutes a value. In a script's own body, which has no named result, a
top-level `defer` assigns a variable instead (see [`defer` at the top level](../language.md#defer-at-the-top-level-of-a-script)).

### `require(cond, payload)`

The input check a script opens with. When `cond` is true it answers `undefined` and the script carries on;
otherwise it raises a recoverable error of kind `requirement` carrying `payload` **untouched** — so a
structured rejection reaches the host as structure, not as a sentence.

```go
require(is_decimal(input.amount), {field: "amount", reason: "not a number"})
require(input.amount > 0d,        {field: "amount", reason: "must be positive"})
// the happy path follows
```

Truthiness is the same `is_true()` uses. Exactly two arguments; anything else is `wrong_num_arguments`.

### Selective re-raise

Handle the kinds you expect; re-raise everything else unchanged (`raise(e)` keeps the original kind and
payload):

```go
defer func() {
    e := recover()
    if e == undefined {
        return                          // nothing in flight
    }
    if e.kind() == "division_by_zero" {
        // handle this one
    } else {
        raise(e)                        // not ours — let it keep unwinding
    }
}()
```

A `division_by_zero` handled this way answers the substitute; an `index_out_of_bounds` raised in the same
position passes through to the outer handler with its kind intact.

## Operators

Errors **propagate by raising**: an error is not a number, a string, or a flag, so combining it with anything
raises `invalid_binary_operator` — whichever side it stands on. Only equality is defined, and it compares
**payloads** (the kind and flags do not participate):

```go
error("boom") == error("boom")   // true  — equal payloads
error("boom") == error("bam")    // false
error(1) == error(1)             // true
error(1) == 1                    // false — an error never equals a non-error
error("a") != error("b")         // true

error("x") + 1                   // runtime error: error + int
1 + error("x")                   // runtime error: int + error
error("x") + error("y")          // runtime error: error + error
error(1) < error(2)              // runtime error — no ordering
!error("x")                      // false — every error is truthy
```

## Member functions

#### `kind()`

The machine-readable tag, a `string`. Constructed and `raise`d errors are `"user"`; runtime errors carry the
tags in the [kind table](#common-error-kinds) below. This is the value a handler matches on.

```go
error("boom").kind()          // "user"
error("boom", true).kind()    // "user" — the fatal flag is not a kind
```

#### `value()`

The payload, exactly as wrapped — and the ONLY structured channel an error has. `kind()` is a tag; there is no
further field set on a runtime error.

| the error came from | `value()` is |
| --- | --- |
| the runtime (a builtin, member, module) | the message `string`, always non-empty: `1/0`'s is `"division by zero"` |
| `error(x)` / `raise(x)` | `x`, untouched — any type |
| `require(cond, x)` | `x`, untouched — any type |

```go
error(42).value()          // 42
error(undefined).value()   // undefined
```

So a script that wants to hand a structured rejection to its host raises one:

```go
require(amount > 0d, {field: "amount", reason: "must be positive"})
// host: re.Payload is that record, not a sentence to re-parse
```

#### `is_runtime()` / `is_user()` / `is_requirement()`

The [category](#categories) predicates — where the error came from. Exactly one is `true` for any error a
script can hold.

```go
error("boom").is_user()            // true
error("boom").is_runtime()         // false
error("boom").is_requirement()     // false

// a recovered 1/0 error
e.kind()                            // "division_by_zero"
e.is_runtime()                      // true

// a recovered require(false, {field: "amount"})
e.kind()                            // "requirement"
e.is_requirement()                  // true
e.value()                           // {field: "amount"} — the payload, untouched
```

There is no `is_system()`: a system error is always fatal and never reaches a script. There is no `is_fatal()`
either — the fatal flag's only reader is the host (`RuntimeError.Fatal`). A script *sets* it, with
`error(x, true)` or `raise(x, true)`, and can never read it back off a caught error, because a caught error
is recoverable by construction.

#### `string()` / `runes()`

The **payload's** render — the same path `format()` uses — as `string`/`runes`. This is a render of the
content, not a laundering conversion: the result describes the payload, whatever its type. The free
`string(e)` matches.

```go
error("boom").string()   // "boom"
error(42).string()       // "42"
error("boom").runes()    // u"boom"
string(error(42))        // "42"
```

#### `bool()` / `is_true()`

Both answer `true` for every error — truthiness is the single stated base case: an error is always truthy,
payload or no payload. `bool()` does **not** parse the payload.

```go
error(0).is_true()        // true
error(0).bool()           // true
error("false").bool()     // true — not a payload parse
error(undefined).is_true() // true
```

#### `copy()` / `freeze()`

Real deep operations on the payload — not header no-ops. `copy()` answers an error with a deep copy of the
payload; `freeze()` answers an error whose payload is deep-frozen (the receiver is unchanged).

```go
e := error([1, 2])
c := e.copy()
c.value().push_in_place(3)
format(c.value())              // "[1, 2, 3]"
format(e.value())              // "[1, 2]" — the original payload untouched

fe := e.freeze()
is_immutable(fe.value())       // true
is_immutable(e.value())        // false
```

#### `format([spec])`

The universal render. The default form renders the payload; `"v"` shows the constructor form.

```go
error("boom").format()      // "boom"
error("boom").format("v")   // "error(\"boom\")"
```

## Every error kind

Each kind has one severity and one [category](#categories), always. **Recoverable** kinds can reach a script's
`recover()`; **system** kinds are fatal and only ever reach the host.

### Script and script-data kinds — recoverable, category *runtime*

| kind | raised by | example trigger |
| --- | --- | --- |
| `division_by_zero` | `/` or `%` by zero, on `int` and `decimal` | `1 / 0`, `1d / 0d` |
| `invalid_argument_type` | an argument outside a member's accepted readings | `[1, 2].join(1.5)` |
| `invalid_value` | a value of the right type outside the valid domain — overflow, a declared limit, an empty sequence | `range(1, 4, 0)`; `"ab".repeat(1e12)`; `[].first()` |
| `wrong_num_arguments` | wrong argument count | `"a".upper(1)`, `[1, 2].chunk()` |
| `index_out_of_bounds` | index outside the sequence | `[1, 2][5]` |
| `invalid_index_type` | index of the wrong type | `[1, 2]["a"]` |
| `invalid_selector` | a selector a type does not answer | `{a: 1}.b.c` |
| `invalid_unpack_type` | destructuring a value that cannot be unpacked | `a, b := 5` |
| `not_mutable` | any mutating member on a frozen receiver | `freeze([1, 2]).push_in_place(3)` |
| `not_assignable` | an assignment statement into a frozen container | `a := freeze([1, 2]); a[0] = 9` |
| `not_accessible` / `not_appendable` / `not_deletable` / `not_iterable` / `not_sliceable` | an operation the type does not support | `range(1, 10)[1:3]` |
| `invalid_method` | calling a member the type does not have | `(5).len()` |
| `invalid_binary_operator` / `invalid_unary_operator` | an operator pair with no meaning | `error("x") + 1`; `-[1]` |
| `not_callable` | calling a non-callable value | `x := 5; x()` |
| `not_implemented` | a reading deferred to a type that does not exist yet | `range(1, 5).contains(1..3)` |
| `conversion` | a failed conversion or parse | `"abc".int()`, `decimal("abc")`, `(1e39).decimal()` |
| `unsupported_format_spec` | a malformed or unsupported format spec | `(5).format("{:,}")`, `format("{0:", [1])` |
| `formatting` | a value that cannot be rendered at all | — |
| `undefined_variable` | reading a name the runtime cannot resolve | — |
| `module_not_found` | `import` of a name that is not registered | `import("nope")` |
| `json_encoding` | a value with no JSON representation | `json.encode(func(){})` |
| `json_decoding` | malformed JSON input | `json.decode("{")` |
| `binary_encoding` | a value that cannot be binary-encoded, or corrupt input | — |
| `io` | the world refused: a file, a directory, an environment variable, a process | `os.remove("/nope")` |

### Script-authored kinds — recoverable

| kind | category | raised by | example trigger |
| --- | --- | --- | --- |
| `user` | user | `error(...)` construction, `raise(x)` | `raise("text")`, `raise({code: 42})` |
| `requirement` | requirement | `require(cond, payload)` with a falsy condition | `require(false, "bad input")` |

### System kinds — **fatal**, category *system*

These bypass every `recover()`, skip every remaining `defer`, and reach the host only. A script can never hold
one, which is why no `is_system()` predicate exists.

| kind | means |
| --- | --- |
| `stack_overflow` | the call-frame or operand stack ran out — usually unbounded recursion |
| `internal` | a VM invariant was violated, or a Go panic was contained at the run boundary |
| `host` | a host setup mistake: `SetValueType` on a builtin slot, an unregistered module ID |

`not_mutable` is uniform: **every** mutating member on an immutable receiver raises it, whatever the verb.
`not_assignable` is the assignment statement's own kind.

## Telling apart the kinds of wrong

A kind says what failed, not **whose fault it was** — and it cannot: `int(x)` does not know where `x` came
from. Three kinds carry that signal:

| signal | read it as |
| --- | --- |
| `requirement` | the script's own input check rejected the caller's data. The payload says which field and why. |
| `conversion`, `json_decoding`, `io` | the data or the world did not cooperate — a bad parse, malformed input, a missing file |
| anything else | most likely a defect in the script itself |

A host that wants to distinguish "the caller sent bad data" from "the rules have a bug" should branch on that,
not on the message text. See [embedding](../embedding.md#handling-errors).

## Migration notes

- `is_fatal()` is **removed**. A caught error is recoverable by construction, so the member could only ever
  read back a flag the script itself had set; the host reads severity from `RuntimeError.Fatal`.
- `is_runtime()` **changed meaning**. It used to answer `kind() != "user"` — true for everything a script did
  not construct. It now answers "this error's category is runtime", so it is `false` for a `requirement` error
  as well as for a `user` one.
- `is_user()` and `is_requirement()` are **new**.

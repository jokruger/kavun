# error

A failure as a first-class value.

## Overview

An `error` is a value that carries a payload describing a failure. Errors reach a script two ways:

- **constructed** — `error(payload)` wraps any value; the script raises it itself with `raise(...)`;
- **raised by the runtime** — division by zero, a bad index, a failed conversion, … — and caught with
  `recover()` inside a deferred function.

Every error is **truthy** — an error without a payload is still an error, and there is no zero error
(`error()` with no arguments is itself a runtime error: `expected 1 or 2 argument(s)`). Each error carries a
machine-readable `kind()` tag, a `value()` payload, and two classification flags, `is_fatal()` and
`is_runtime()`.

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
a caught runtime error came back out as kind `"user"` with `is_runtime()` false. The severity form keeps the
payload and kind and changes only the flag, which is exactly what `raise(err, fatal)` already did:

```go
e = error("boom")
error(e) == e               // true — the same error, not a nested one
error(e).value()            // "boom", a string — not an error
error(e, true).is_fatal()   // true; kind() and value() are still "user"/"boom"

// a runtime error keeps its own diagnosis through the constructor
c = /* a caught invalid_method error */
error(c).kind()             // "invalid_method" — not relabelled to "user"
error(c).is_runtime()       // true
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
named result is how a handler substitutes a value.

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

The payload, exactly as wrapped. For runtime errors the payload is the message `string` (occasionally empty —
`1/0`'s payload is `""`; its information is the kind).

```go
error(42).value()          // 42
error(undefined).value()   // undefined
```

#### `is_fatal()` / `is_runtime()`

Classification flags: was this error marked fatal at construction, and was it raised by the runtime (as
opposed to constructed/raised by the script)?

```go
error("boom").is_fatal()          // false
error("boom", true).is_fatal()    // true
error("boom").is_runtime()        // false
// a recovered 1/0 error: kind "division_by_zero", is_runtime() true, is_fatal() false
```

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

## Common error kinds

The tags a script can match on with `e.kind()`. Each row below is a verified trigger:

| kind | raised by | example trigger |
| --- | --- | --- |
| `user` | `error(...)` construction, `raise("text")`, `raise(42)` | `raise("text")` |
| `division_by_zero` | `/` or `%` by zero | `1 / 0`, `1 % 0` |
| `invalid_argument_type` | an argument outside a member's accepted readings | `[1, 2].join(1.5)` |
| `invalid_value` | a value of the right type outside the valid domain (overflow included) | `range(1, 4, 0)`; `9223372036854775807 + 1` |
| `wrong_num_arguments` | wrong argument count | `"a".upper(1)`, `[1, 2].chunk()` |
| `index_out_of_bounds` | index outside the sequence | `[1, 2][5]` |
| `invalid_index_type` | index of the wrong type | `[1, 2]["a"]` |
| `not_mutable` | any mutating member on a frozen receiver | `freeze([1, 2]).push_in_place(3)` |
| `not_assignable` | an assignment statement into a frozen container | `a := freeze([1, 2]); a[0] = 9` |
| `invalid_method` | calling a member the type does not have | `(5).len()` |
| `invalid_binary_operator` | an operator pair with no meaning | `error("x") + 1` |
| `not_implemented` | a reading deferred to a type that does not exist yet | `range(1, 5).contains(1..3)` |
| `not_callable` | calling a non-callable value | `x := 5; x()` |
| `not_sliceable` | slice syntax on a type without it | `range(1, 10)[1:3]` |
| `conversion` | a failed conversion/parse | `"abc".int()`, `int("abc")` |

`not_mutable` is uniform: **every** mutating member on an immutable receiver raises it, whatever the verb.
`not_assignable` is the assignment statement's own kind.

# module

A builtin module, as `import` answers it.

## Overview

`import("math")` answers a value of type `module`: the module's functions and constants are its **members**, it
is read-only, and it has no content of its own to convert. Every builtin module — the standard library ones and
any an embedder registers — is its own type with this one Kavun type name, so scripts see one shape:

```go
math := import("math")
type_name(math)        // "module"
math.cbrt(27)          // 3.0      — a member call
math.pi                // 3.141592653589793 — a constant member
[8, 27].map(math.cbrt) // [2.0, 3.0] — a function member read as a value
format(math)           // `import("math")` — it renders as the expression that answers it
```

A source module (`import("./helpers")`, or one added with `Script.AddCustomModule`) is not a `module` value: it
answers whatever it `export`s, frozen — a record keeps record rules. See
[language.md § Modules](../language.md#modules).

Which builtin modules a script may import is the host's decision. By default only the **pure** ones (they only
compute) are importable; a world-touching module (`clock`, `exec`, `fs`, `io`, `os`) needs a grant, checked at compile
time and again by the VM (`module_not_allowed`). See [embedding.md § The module grant](../embedding.md#the-module-grant).

## Operators

| operator | reading | example |
| --- | --- | --- |
| `==` / `!=` | the same module (for a builtin module: the same import) | `import("math") == import("math")` → `true` |
| `in` | a member name | `"pi" in math` → `true`; `1 in math` → `false` |
| `!` | truthiness — a module is always truthy | `!math` → `false` |
| everything else | raises `invalid_binary_operator` / `invalid_unary_operator` | `math + 1`, `math < math`, `-math` |

A module is not indexable (`math["pi"]` raises `not_accessible` — members are dot access only), not sliceable and
not callable.

## Members

A module answers its own functions and constants, plus the four universal members. It has **no** `string()`:
a module has no text content, so `string(math)` raises `conversion` and `math.string()` raises like any missing
member.

| member | answers |
| --- | --- |
| its functions | `m.f(args)` calls; `m.f` is the `builtin_function` value — the same value every time (`math.cbrt == math.cbrt`) |
| its constants | the constant's value |
| `is_true()` | `true` |
| `format([spec])` | the render, `import("math")`, with the generic spec (`math.format(">20")`); `{:T}` → `module` |
| `copy()` / `freeze()` | the module itself |

A name the module does not have raises, naming the module — as a read and as a call:

```go
math.pii       // invalid_selector: module 'math' has no member 'pii'
math.pii()     // invalid_method:   module 'math' has no member 'pii'
math.pi = 1    // not_assignable:   module 'math' is read-only: cannot assign member 'pi'
```

## Size and iteration

`len(m)` counts its functions and constants; `for k, v in m` visits them sorted by name, with functions as
`builtin_function` values:

```go
h := import("hex")
len(h)                                  // 2
names := []
for k, _ in h { names.push_in_place(k) }
names                                   // ["decode", "encode"]
```

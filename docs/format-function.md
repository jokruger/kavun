# `format(template, args)`

`format` is the runtime counterpart to [f-strings](f-strings.md). It takes a template string and a container of values
and produces a formatted string. The template uses the same `{...}` placeholder syntax as f-strings and the same
[Format Mini-Language](format-mini-language.md) for the optional `:fspec` suffix — but the template itself is a plain
runtime string, so it can be built, loaded, or chosen at run time.

```go
fmt = import("fmt")

fmt.println(format("hello {x} from {y}!", {x: "Kavun", y: "Kherson"}))
fmt.println(format("hello {0} from {1}!", ["Kavun", "Kherson"]))

fmt.println(format("pi = {x:.3f}",  {x: 3.14159}))        // pi = 3.142
fmt.println(format("n = {x:{fmt}}", {x: 42, fmt: "05d"})) // n = 00042
```

## Signature

```
format(template: string, args: array | dict | record) -> string
```

- `template` must be a `string`, `runes` or `bytes`.
- `args` must be either an `array` (for indexed placeholders) or a `dict` / `record` (for named placeholders).

The template's _mode_ (named vs. indexed) is locked by its first placeholder and every subsequent placeholder must use
the same mode. Mixing the two in a single template is an error.

## Placeholder syntax

```bnf
template     := { text_char | '{{' | '}}' | placeholder }
placeholder  := '{' name_or_index [ ':' spec_body ] '}'
spec_body    := fspec_text                       ; a literal Format Mini-Language spec
              | '{' name_or_index '}'            ; the entire spec supplied at run time
name_or_index := identifier | non-negative-integer
```

- An `identifier` follows the usual rule — ASCII letter or `_` followed by ASCII letters, digits, or `_`.
- A non-negative integer indexes into the `args` array (0-based).
- `{{` / `}}` produce a literal `{` / `}` in the output. A bare `}` is an error.
- `{}` with no name or index is an error — there is no auto-numbering.
- Expressions are **not** allowed inside `{...}` — only a single name or index.

### Format spec by reference

The spec part can either be a literal — passed verbatim to `fspec.Parse` — or a _single_ `{name_or_index}` whose value
is fetched from the same `args` container at run time and parsed as a spec:

```go
format("{x:{fmt}}", {x: 42, fmt: "05d"})  // "00042"
format("{0:{1}}",   [42, "05d"])          // "00042"
```

Restrictions:

- The reference must occupy the entire spec body — `"{x:>{w}}"` is not accepted (use a precomputed spec string instead).
- The referenced value must be a `string`.
- Only one level of nesting is allowed: the inner `{...}` may not itself contain `{...}`.

These restrictions are intentional — they keep the template parser small, fast, and unambiguous.

## Errors

`format` reports the following runtime errors:

| Condition                                                 | Error                        |
| --------------------------------------------------------- | ---------------------------- |
| Wrong number of arguments                                 | wrong number of arguments    |
| Non-string template                                       | invalid argument type        |
| `args` is not an array, dict, or record                   | invalid argument type        |
| Mixing named and indexed placeholders in one template     | logic error (template parse) |
| Template syntax error (bare `}`, empty `{}`, etc.)        | logic error (template parse) |
| Template uses indexed placeholders but `args` isn't array | logic error                  |
| Template uses named placeholders but `args` is array      | logic error                  |
| Index out of range                                        | logic error                  |
| Missing key                                               | logic error                  |
| Spec reference is not a string                            | logic error                  |
| Spec parsing failure                                      | logic error                  |
| Type's `Format` rejects the spec                          | unsupported format spec      |

## When to use `format` vs. an f-string

Use an **f-string** when the template is part of the source code and the values are local expressions. F-strings are
parsed at compile time, can embed arbitrary expressions, and produce more efficient bytecode.

Use **format** when the template comes from outside the program (a config file, a translation catalog, user input),
when the same template is selected among several at run time, or when the values arrive as a single container.

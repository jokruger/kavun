# F-Strings

Kavun supports f-strings — interpolated string literals that embed expressions and format them at compile-bound,
run-evaluated boundaries.

```go
name = "world"
n = 42
s = f"hello, {name}! n={n:5d}"   // "hello, world! n=   42"
```

## Syntax

```bnf
fstring   := 'f' '"' { fpart } '"'
fpart     := text_char
           | '{{' | '}}'                       ; literal '{' / '}'
           | '{' expr [ ':' fspec ] '}'
expr      := <any Kavun expression>
fspec     := <Format Mini-Language; literal text only — see format-mini-language.md>
text_char := <any rune except '"', '\\', '{', '}'>
           | '\\' <escape>                    ; standard string escape
```

An f-string starts with a lower-case `f` immediately before the opening `"` (no space). The literal `f` prefix is
required; there is no implicit prefix syntax.

The body is parsed once at **compile time**: literal segments are unescaped (the same escape set as for regular `"..."`
strings), each `{...}` is split into expression text and (optional) format spec, the expression is sub-parsed, and the
spec is parsed via the [Format Mini-Language](format-mini-language.md). Any error in any of these steps is reported as a
compile error with a source position.

At run time, each `{expr[:fspec]}` interpolation evaluates the expression and calls the value type's `Format` method
with the pre-parsed spec; the resulting fragments are concatenated in left-to-right order.

An f-string body may contain zero, one, or many `{...}` interpolations, mixed freely with literal text. There is no
fixed pattern. All of the following are valid:

```go
f""                       // empty string
f"hello"                  // pure literal -> a plain string constant
f"{x}"                    // single interpolation, no surrounding text
f"prefix {x}"             // leading literal
f"{x} suffix"             // trailing literal
f"{x}{y}"                 // adjacent interpolations, no separator
f"a={x} b={y} c={z}"      // many interpolations
f"<{a}{b}>{c}"            // any sequence
```

A pure-literal f-string compiles to a single string constant — no formatting opcode, no concatenation overhead.

### Escapes

Inside an f-string body:

- The standard string escape sequences are honored: `\n`, `\t`, `\r`, `\\`, `\"`, `\xHH`, `\uHHHH`, `\UHHHHHHHH`,
  octal `\NNN`. They have the same semantics as in regular `"..."` strings.
- A literal `{` is written `{{`; a literal `}` is written `}}`.
- A bare `}` is a compile error.
- An f-string cannot span multiple physical lines (matches the rule for regular `"..."` strings; multi-line f-strings
  will be added later).

Examples:

```go
f"path = \"{p}\""        // path = "..."
f"set = {{1, 2, 3}}"     // set = {1, 2, 3}
f"newline -> {x}\n"      // ends with a real newline
```

### Format specs

After the expression, an optional `:fspec` segment selects a per-type format spec from the
[Format Mini-Language](format-mini-language.md):

```go
f"{pi:.2f}"              // 3.14
f"{n:05d}"               // 00042
f"{n:>10,}"              // "     1,234"
f"{t:#date}"             // multi-character verb
```

The fspec is **literal text** — it is parsed at compile time and cannot reference variables (no nested interpolation).
Errors in the spec (unknown verb, malformed grammar, etc.) are reported at compile time:

```go
f"{x:zzz}" // Parse Error: f-string format spec "zzz": fspec: trailing characters "z" in "zzz"
```

### Empty format spec

Three forms are conceptually distinct, two of them are identical in practice:

| Form       | Spec text passed to fspec.Parse | Notes                            |
| ---------- | ------------------------------- | -------------------------------- |
| `f"{x}"`   | `""`                            | "no spec"                        |
| `f"{x:}"`  | `""`                            | explicit empty — same as no spec |
| `f"{x:v}"` | `"v"`                           | explicit "default verb" verb     |

`fspec.Parse("")` returns the zero `FormatSpec`. Each value type's `Format` method decides what an empty spec means for
that type — for many types it is equivalent to the type's default rendering, but the f-string machinery does **not**
rewrite empty specs to `v` or to anything else.

### What can go inside `{...}`

The expression portion is a full Kavun expression — anything you can put on the right-hand side of `=` is allowed:
identifiers, arithmetic, function calls, method chains, record literals, ternaries, indexing, etc.

```go
f"{x + y}"                              // arithmetic
f"{users[i].name}"                      // selection
f"{ dict({a: 1, b: 2}).values() :v}"    // dict literal in expression
f"{cond ? \"yes\" : \"no\"}"            // ternary
```

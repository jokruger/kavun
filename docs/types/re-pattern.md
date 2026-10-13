# re.pattern

A compiled regular expression, as `re.pattern(src)` answers it.

## Overview

`re.pattern` is owned by the [`re`](../stdlib.md#re) module and built only by its constructor. It is immutable,
its members are pure (the same pattern and text always answer the same result), and its text content is its
source — so unlike the world handles (`fs.file`, …) it has `string()`, and `string(p)` works too. It renders as the
expression that builds it.

```go
re := import("re")
p := re.pattern("([a-z]+)([0-9]+)")
type_name(p)                 // "re.pattern"
format(p)                    // `re.pattern("([a-z]+)([0-9]+)")`
string(p)                    // "([a-z]+)([0-9]+)"
p.match("ab12")              // true
p.replace("ab12 cd34", "$2$1")   // "12ab 34cd"
```

The syntax is Go's RE2: no backreferences or lookaround, and matching time is linear in the input — a pattern
cannot be made to run away on hostile text.

## Operators

| operator | reading |
| --- | --- |
| `==` / `!=` | the same source text (`re.pattern("a+") == re.pattern("a+")`) |
| `!` | a pattern is always truthy |
| everything else | raises |

## Members

| member | answers |
| --- | --- |
| `match(s)` | whether the expression matches anywhere in `s` |
| `find(s[, count])` | the first match, or up to `count` (`-1`: all); each match is an array of frozen `{text, begin, end}` group records, group 0 first; `[]` when nothing matches |
| `replace(s, repl)` | `s` with every match replaced; `$1`, `${name}` expand groups; a result past the sequence ceiling raises `invalid_value` |
| `split(s[, count])` | `s` cut at the matches (`count` limits the pieces; `-1` or absent: all) |
| `string([default])` | the source text |
| `is_true()`, `format([spec])`, `copy()`, `freeze()` | `true`; the render; the pattern itself |

```go
p.find("ab12 cd34")          // [[{begin: 0, end: 4, text: "ab12"}, {begin: 0, end: 2, text: "ab"}, {begin: 2, end: 4, text: "12"}]]
p.find("ab12 cd34", -1).len()    // 2
p.find("!!")                 // []
re.pattern(",").split("a,b,c", 2)    // ["a", "b,c"]
```

# Format Mini-Language

This document specifies the format mini-language used by Kavun for value formatting and f-strings. It is parsed at
compile time into a `FormatSpec` struct; at runtime each interpolation site invokes the value type's `Format` method
with the prebuilt spec.

For the surrounding f-string syntax (the part outside the spec), see [F-Strings](f-strings.md).

## Goals

- Familiar to users coming from Python / Go / Rust.
- Cheap at runtime: no spec parsing, single allocation per interpolation.
- Cleanly extensible: any type, including user-defined ones, can introduce its own verbs without touching the parser.
- Type-driven: each type owns the rendering of its values; a small library of helpers handles common chores
  (alignment, padding, grouping, sign).

## Grammar

```bnf
format_spec := generic ['#' tail] | '#' tail
generic     := [[fill] align] [sign] [width] [grouping] ['.' precision] flag* [verb]
fill        := <any char except '{' '}'>           ; only valid if followed by align
align       := '<' | '>' | '^' | '='
sign        := '+' | '-' | ' '
width       := digit+
grouping    := ',' | '_'
precision   := digit+
flag        := '~' | '!'                            ; each at most once, any order
verb        := <single ASCII letter> | '%'        ; type-defined; '' = default
tail        := <opaque to the parser; passed verbatim to the type>
```

A leading `0` in `width` (no explicit `align`) is treated as a shortcut for `fill='0', align='='` and sets the `ZeroPad`
flag.

A generic verb **may** be combined with a `'#' tail` (e.g. `x#a#b` parses as `Verb='x', Tail="a#b"`). The parser is
permissive here; whether the combination is meaningful is up to the type. Types that don't accept a tail with a
generic verb reject the spec via `FormatSpec.HasUnconsumedTail()`. The universal verbs `v` and `T` ignore everything
else, including any tail.

### The `#` separator

`#` is **not** the standard "alternate form" flag here. It is the optional **generic/tail separator**. Once the parser
sees a `#`, it stops consuming generic fields and stores the rest of the spec verbatim in `FormatSpec.Tail`.

When `#` is present, the parser sets `FormatSpec.Verb = '#'` (the literal hash byte). This gives a type's `Format`
method a single, cheap discriminator:

- `Verb == 0` — default rendering, no spec body.
- `Verb` is an ASCII letter — one of the type's generic verbs.
- `Verb == '#'` — tail form requested; the payload is in `Tail`. Types that don't support a tail can simply reject
  this case with a single comparison.

Use cases:

- Multi-character verbs: `f"{t:#date}"`, `f"{t:#%Y-%m-%d}"`.
- Verbs that start with characters reserved by the generic grammar (e.g. starts with a digit or `.`).
- User-defined types: `f"{u:#badge,large}"`.

The first `#` is the separator; any further `#` characters are part of the tail. If `#` is absent, parsing of `verb`
proceeds normally — a single ASCII letter at the end of `generic` becomes the verb, anything beyond it is a parse error.

## Generic fields

### Fill and alignment

If `align` is given it may be preceded by any single character used as the padding fill (defaults to space, or `'0'`
when `ZeroPad` is set).

Aligns:

- `<` — left-align (default for non-numeric types).
- `>` — right-align (default for numeric types).
- `^` — centered.
- `=` — pad between sign and digits (numerics only): `+0000123`.

Unless `width` is set, alignment has no effect.

### Sign

Numeric only:

- `+` — sign on both positive and negative.
- `-` — sign only on negative (default).
- ` ` — leading space on positive, `-` on negative.

### Width

Decimal integer ≥ 0. Minimum total field width including any prefix, sign, separators. If the rendered body already
meets `width`, no padding.

A leading `0` (without explicit `align`) enables sign-aware zero padding for numeric types: `{x:05d}` → `00042` /
`-0042`. Equivalent to setting `fill='0'` and `align='='`. Implemented by the type itself.

### Grouping

`,` or `_` inserts a separator every 3 digits in the integral part of integer / float / decimal. For integer verbs
`b`/`o`/`x`/`X`, `_` groups every 4 digits and `,` is a parse error. No grouping for the fractional part (intentional
simplification).

### Precision

Decimal integer ≥ 0, after a `.`. Meaning depends on the type/verb:

- `f`, `e`, `E`, `g`, `G`, `%` (float, decimal): digits after the point (or significant digits for `g`/`G`).
- `s` and string-like verbs: maximum number of characters (`runes`) used from the source value.
- Forbidden for integer verbs (parse error).

### Flags

A small set of single-character **symbol** flags may follow `precision` (and precede the `verb`). Each flag may appear
at most once; order within the flag set is not significant.

- `~` — for float / decimal: coerce negative zero to positive zero after rounding to the requested precision.
- `!` — for integer verbs that emit a conventional prefix (`b`, `o`, `x`, `X`): suppress the prefix, rendering the bare
  digits only. Example: `f"{0o755:!o}"` → `"755"`.

> **Design rule — character classes.** ASCII letters (and `%`) are reserved for **verbs**. `#` is the generic / tail
> separator. All other ASCII symbols and digits belong to the **grammar** (alignment, sign, width, precision,
> grouping, flags). New flags must always be non-letter symbols so that the verb namespace stays open for every type.

> **Gotcha — flag char as fill char.** Any character that immediately precedes an alignment char becomes the fill
> character (see [Fill and alignment](#fill-and-alignment)). This includes the flag chars `~` and `!`. So
> `f"{x:!^8o}"` parses as `fill='!', align='^', width=8, verb='o'` — the `!` is **fill**, not a flag, and no prefix is
> suppressed. To use a flag char as both fill and flag, write it twice in their respective positions:
> `f"{x:!^8!o}"` (fill `!`, align `^`, width 8, then the `!` flag, then verb `o`).

### Verb

A single ASCII letter at the end of `generic`. Type-defined; an empty verb selects the type's default. Verbs longer than
one letter, or that collide with grammar characters, must use the `#`-tail form.

## Tail

Anything after the (optional) first `#` is the **type tail**. The generic parser does not look at it. The type's
`Format` method receives it as a plain string in `FormatSpec.Tail` and may parse it however it wishes. This is the
extensibility hook for user-defined types and for types whose verbs are inherently multi-character.

A type that doesn't recognize its tail must return a format error.

## Universal verbs

Three forms are well-defined for **every** value, regardless of type:

| Verb      | Meaning                                                                    |
| --------- | -------------------------------------------------------------------------- |
| _(empty)_ | **Default** — human-friendly rendering.                                    |
| `v`       | **Kavun-source representation** — the canonical literal form of the value. |
| `T`       | **Type name** — the value's type, as reported by the type's `Name` hook.   |

Examples of default vs. `v`:

| Type      | Default     | `v`                                |
| --------- | ----------- | ---------------------------------- |
| `int`     | `42`        | `42`                               |
| `float`   | `1.5`       | `1.5`                              |
| `decimal` | `1.23`      | `1.23d`                            |
| `bool`    | `true`      | `true`                             |
| `string`  | `hello`     | `"hello"`                          |
| `runes`   | `hello`     | `u"hello"`                         |
| `bytes`   | `hello`     | `bytes([104, 101, 108, 108, 111])` |
| `rune`    | `A`         | `'A'`                              |
| `byte`    | `65`        | `byte(65)`                         |
| `time`    | RFC 3339    | `time("2026-…")`                   |
| `array`   | `[1, 2, 3]` | `[1, 2, 3]`                        |
| `dict`    | `{"a": 1}`  | `dict({"a": 1})`                   |
| `record`  | `{"a": 1}`  | `{"a": 1}`                         |
| `range`   | `[0..10)`   | `range(0, 10, 1)`                  |
| `error`   | message     | `error("…")`                       |

### `v` — Kavun-source representation

`v` ignores every other field of the spec — fill, align, width, sign, grouping, precision, zero-pad, the `~` and `!`
flags and any `#`-tail. They are silently discarded so `v` always renders the canonical Kavun-source form.

Each format-supporting type implements `v` in its own `Format` method, by convention as `case 'v': return
v.String(), nil`. This means that for every type that has a meaningful Kavun-source representation, the `String()`
method and `f"{x:v}"` produce the same string. Types whose `String()` is purely diagnostic (functions, iterators,
etc.) simply do not implement `v` — formatting such a value with `v` is a runtime error, even though `String()` is
still available for log output. User-defined types follow the same rule and decide whether to support `v`.

### `T` — type name

For any value, `f"{x:T}"` renders the value's type name (e.g. `int`, `float`, `bool`, `string`, `runes`, `bytes`,
`rune`, `byte`, `time`, `array`, `dict`, `record`, `range`, `error`). Generic fields (`fill`, `align`, `width`) apply
normally with left-alignment as the default; other modifiers (sign, grouping, precision, `~`, `!`, `#`-tail) are ignored.

## Per-type verb tables

### Numbers: `int`, `byte`

| Verb | Meaning                                               |
| ---- | ----------------------------------------------------- |
| `d`  | Decimal (default).                                    |
| `b`  | Binary, prefix `0b`.                                  |
| `o`  | Octal, prefix `0o`.                                   |
| `x`  | Hex lowercase digits, prefix `0x`.                    |
| `X`  | Hex uppercase digits, prefix `0x` (always lowercase). |
| `c`  | Code point (int) or ASCII byte (byte) as a character. |
| `q`  | Quoted character literal (e.g. `'A'`, `'\n'`).        |

Supports `sign`, `width`, `grouping`, `ZeroPad`. `precision` is a parse error. The `!` flag (suppress prefix) applies to
`b`, `o`, `x`, `X` and is a parse error on `d` / `c` / `q`.

Grouping `,` is decimal-only; use `_` with `b` / `o` / `x` / `X`.

### Numbers: `float`, `decimal`

| Verb | Meaning                                          |
| ---- | ------------------------------------------------ |
| `f`  | Fixed-point (default precision 6).               |
| `F`  | Same as `f`, uppercase `INF`/`NAN`.              |
| `e`  | Scientific lowercase.                            |
| `E`  | Scientific uppercase.                            |
| `g`  | Shortest of `f`/`e` (default verb).              |
| `G`  | Shortest of `F`/`E`.                             |
| `%`  | Multiply by 100, append `%`, otherwise like `f`. |

Supports `sign`, `width`, `grouping` (integral part only), `precision`, `ZeroPad`, `~`. The `!` flag is a parse error.

Decimal additionally accepts:

| Verb | Meaning                                            |
| ---- | -------------------------------------------------- |
| `s`  | Preserve source scale (no trim of trailing zeros). |

### `bool`

| Verb | Meaning                     |
| ---- | --------------------------- |
| `t`  | `true` / `false` (default). |
| `d`  | `1` / `0`.                  |

### `rune`

| Verb | Meaning                    |
| ---- | -------------------------- |
| `c`  | UTF-8 character (default). |
| `d`  | Code point as integer.     |
| `x`  | Code point lower hex.      |
| `X`  | Code point upper hex.      |
| `U`  | Unicode notation `U+%04X`. |
| `q`  | Quoted: `'A'`.             |

### Strings: `string`, `runes`, `bytes`

| Verb | Meaning                                                                   |
| ---- | ------------------------------------------------------------------------- |
| `s`  | Raw text (default).                                                       |
| `v`  | Source form (`"hello"` / `u"hello"` / `bytes("hello")`).                  |
| `q`  | Double-quoted with Kavun escapes.                                         |
| `b`  | Base64 standard.                                                          |
| `B`  | Base64 URL-safe, no padding.                                              |
| `x`  | Hex of UTF-8 bytes, lowercase.                                            |
| `X`  | Hex of UTF-8 bytes, uppercase.                                            |
| `u`  | Percent-encoded URL component (RFC 3986 unreserved set: `A-Za-z0-9-_.~`). |

`precision` truncates the _source_ before encoding: it counts runes for `string` / `runes` and bytes for `bytes`.

Width / fill / alignment apply to the rendered string; sign, grouping, zero-pad and the `~` / `!` flags are parse errors
unless the verb is `v` (in which case the runtime drops them along with the other generic fields).
Default alignment is left.

### `time`

Verbs are aliases; otherwise use `#`-tail.

| Form        | Meaning                                           |
| ----------- | ------------------------------------------------- |
| (empty)     | RFC 3339, seconds precision (default).            |
| `v`         | Source form: `time("2026-…")`.                    |
| `#iso`      | RFC 3339 explicit, seconds precision.             |
| `#isonano`  | RFC 3339 with sub-second component when non-zero. |
| `#date`     | `2006-01-02`.                                     |
| `#time`     | `15:04:05`.                                       |
| `#unix`     | Unix seconds.                                     |
| `#unixms`   | Unix milliseconds.                                |
| `#rfc822`   | RFC 822.                                          |
| `#<layout>` | Template layout (see directives below).           |

Any tail that doesn't match one of the named aliases above is treated as a template layout. The following
`%`-directives are recognized:

| Code | Meaning                              | Code | Meaning                              |
| ---- | ------------------------------------ | ---- | ------------------------------------ |
| `%Y` | 4-digit year                         | `%B` | full month name (`January`)          |
| `%y` | 2-digit year                         | `%b` | abbreviated month name (`Jan`)       |
| `%C` | century, `00`–`99`                   | `%A` | full weekday name (`Monday`)         |
| `%G` | ISO 8601 week-numbering year         | `%a` | abbreviated weekday (`Mon`)          |
| `%m` | month, zero-padded `01`–`12`         | `%u` | ISO 8601 weekday `1`–`7` (Mon=1)     |
| `%d` | day of month, zero-padded `01`–`31`  | `%w` | weekday `0`–`6` (Sun=0)              |
| `%e` | day of month, space-padded ` 1`–`31` | `%V` | ISO 8601 week of year, `01`–`53`     |
| `%j` | day of year, zero-padded `001`–`366` | `%p` | `AM` / `PM`                          |
| `%H` | hour 24h, `00`–`23`                  | `%P` | `am` / `pm`                          |
| `%I` | hour 12h, `01`–`12`                  | `%Z` | timezone abbreviation (`UTC`, `MST`) |
| `%M` | minute, `00`–`59`                    | `%z` | timezone offset (`-0700`)            |
| `%S` | second, `00`–`59`                    | `%s` | unix seconds                         |
| `%f` | microseconds, `000000`–`999999`      | `%n` | literal newline                      |
| `%t` | literal tab                          | `%%` | literal `%`                          |

Examples: `f"{t:#%Y-%m-%d %H:%M:%S}"`, `f"{t:#%Y-%j}"`, `f"{t:#%I:%M %p}"`. An unrecognized `%`-code is a runtime
formatting error.

Width / fill / alignment apply to the rendered string; sign, grouping, precision, zero-pad and the `~` / `!` flags are
parse errors for `time`.

### `error`

Default = message text. `v` = `error("…")` source form. No other verbs.

### Containers: `array`, `dict`, `record`, `range`

Containers accept only the empty verb (default human-friendly form) and `v` (Kavun source form). For the empty verb,
only width / fill / align are accepted; `precision`, `sign`, `grouping`, `ZeroPad`, `~` and `!` are parse errors. The `v`
verb additionally ignores width / fill / align (plus everything else) per the global rule above.

| Type     | Default                 | `v`                |
| -------- | ----------------------- | ------------------ |
| `array`  | `[1, 2, 3]`             | `[1, 2, 3]`        |
| `record` | `{"a": 1, "b": 2}`      | `{"a": 1, "b": 2}` |
| `dict`   | `{"a": 1}`              | `dict({"a": 1})`   |
| `range`  | `[0..10)` / `[0..10:2)` | `range(0, 10, 1)`  |

Element values are rendered using each value's Kavun-source form (i.e. nested strings appear quoted).
Default alignment is left.

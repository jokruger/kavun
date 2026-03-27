# Module `text`

```text
text := import("text")
```

## Functions

The function names mirror Go's `strings` and `regexp` packages but use snake_case identifiers. Unless stated otherwise, arguments and return values are strings.

- `re_match(pattern, text) => bool/error`
- `re_find(pattern, text, count) => [[{text: string, begin: int, end: int}]]/undefined`
- `re_replace(pattern, text, repl) => string/error`
- `re_split(pattern, text, count) => [string]/error`
- `re_compile(pattern) => regexp/error`
- `compare(a, b) => int`
- `contains(s, substr) => bool`
- `contains_any(s, chars) => bool`
- `count(s, substr) => int`
- `equal_fold(a, b) => bool`
- `fields(s) => [string]`
- `has_prefix(s, prefix) => bool`
- `has_suffix(s, suffix) => bool`
- `index(s, substr) => int`
- `index_any(s, chars) => int`
- `join(parts [string], sep string) => string`
- `last_index(s, substr) => int`
- `last_index_any(s, chars) => int`
- `repeat(s, count) => string`
- `replace(s, old, new, n) => string`
- `substr(s, lower, upper) => string`
- `split(s, sep) => [string]`
- `split_after(s, sep) => [string]`
- `split_after_n(s, sep, n) => [string]`
- `split_n(s, sep, n) => [string]`
- `title(s) => string`
- `to_lower(s) => string`
- `to_title(s) => string`
- `to_upper(s) => string`
- `pad_left(s, length, pad?) => string`
- `pad_right(s, length, pad?) => string`
- `trim(s, cutset?) => string`
- `trim_left(s, cutset) => string`
- `trim_prefix(s, prefix) => string`
- `trim_right(s, cutset) => string`
- `trim_space(s) => string`
- `trim_suffix(s, suffix) => string`
- `atoi(str) => int/error`
- `format_bool(b bool) => string`
- `format_float(f float, fmt string, prec int, bits int) => string`
- `format_int(i int, base int) => string`
- `itoa(i int) => string`
- `parse_bool(s) => bool/error`
- `parse_float(s, bits int) => float/error`
- `parse_int(s, base int, bits int) => int/error`
- `quote(s) => string`
- `unquote(s) => string/error`

## Regexp values

Objects returned by `re_compile` expose:

- `match(text) => bool`
- `find(text, count) => [[{text: string, begin: int, end: int}]]/undefined`
- `replace(text, repl) => string`
- `split(text, count) => [string]`

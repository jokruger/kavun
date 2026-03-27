# Module `json`

```text
json := import("json")
```

## Functions

- `decode(data string|bytes) => object/error`: parses JSON text into arrays, records, ints, floats, bools, and strings.
- `encode(value object) => bytes`: returns the JSON encoding of `value`. Output is UTF-8 and is not HTML-escaped.
- `indent(data string|bytes, prefix string, indent string) => bytes`: returns an indented version of the JSON input.
- `html_escape(data string|bytes) => bytes`: escapes HTML-sensitive characters inside a JSON string.

## Example

```text
json := import("json")
encoded := json.encode({a: 1, b: [2, 3, 4]})
pretty := json.indent(encoded, "", "  ")
safe := json.html_escape(encoded)
decoded := json.decode(encoded)
```

# Module `hex`

```text
hex := import("hex")
```

## Functions

- `encode(data bytes|array) => string`: hexadecimal encoding (lower-case).
- `decode(text string) => bytes/error`: decodes a hexadecimal string.

Encoders accept anything that can be converted to bytes. `decode` returns an `error(...)` value when the input is not valid hexadecimal.

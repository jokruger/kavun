# Module `base64`

```text
base64 := import("base64")
```

## Functions

- `encode(data bytes|array) => string`: standard base64 encoding.
- `decode(text string) => bytes/error`: decodes a standard base64 string.
- `raw_encode(data bytes|array) => string`: base64 encoding without padding.
- `raw_decode(text string) => bytes/error`: decodes a base64 string that omits padding.
- `url_encode(data bytes|array) => string`: URL-safe base64 encoding.
- `url_decode(text string) => bytes/error`: decodes URL-safe base64 input.
- `raw_url_encode(data bytes|array) => string`: URL-safe encoding without padding.
- `raw_url_decode(text string) => bytes/error`: decodes URL-safe base64 input that omits padding.

Encoding helpers expect something that can be converted to bytes (a `bytes` value, an array of integers, or a string). Decoders return `error(...)` when the input is invalid or would exceed the VM's byte-slice limits.

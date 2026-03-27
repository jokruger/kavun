# Standard Library

Import modules with `name := import("name")`. The CLI registers every module by default. When embedding GS you can register whichever modules you want via `stdlib.GetModuleMap`:

Imported modules are immutable records. Attempting to assign to a module field will raise an error.

## Module Catalog

- [`base64`](base64.md): base64 encoders/decoders.
- [`enum`](enum.md): enumeration utilities implemented in GS and shipped as a source module.
- [`fmt`](fmt.md): formatted printing helpers.
- [`hex`](hex.md): hexadecimal encoders/decoders.
- [`json`](json.md): JSON encoding/decoding helpers.
- [`math`](math.md): math constants and functions (snake_case names match the Go `math` package).
- [`os`](os.md): filesystem/process/environment helpers, plus `os.file`, `os.process`, and `os.command` records.
- [`rand`](rand.md): pseudorandom helpers and per-source generators.
- [`text`](text.md): regular expressions and advanced string helpers.
- [`times`](times.md): duration math, parsing, and constants.

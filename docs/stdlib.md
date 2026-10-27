# Standard Library

The builtin modules are listed, with their grants and summaries, in [reference/stdlib.md](reference/stdlib.md); each
has its own generated page under `reference/stdlib/`.

Notes:

- Each module is a value of type [`module`](reference/types/module.md): `import("math")` answers it, its functions and
  constants are read-only members, and a missing name raises.
- **Grant.** `base64`, `fin`, `hex`, `json`, `math`, `path`, `rand` and `re` are *pure* (they only compute) and
  importable by default. `clock`, `exec`, `fs`, `io` and `os` touch the world and must be granted by the
  host (`docs/embedding.md` § The module grant); the `kavun` CLI grants every module.
- Signatures on the module pages use Kavun-facing names and argument order.
- A module function either answers the documented value or **raises** — nothing answers an error value, and
  nothing answers a `true` that means "it worked". A function whose success has no result answers `undefined`.
  Each module page's **Failures** line names the kind it raises; catch one with `defer`/`recover()` (see
  [language.md § Errors and recovery](language.md#errors-and-recovery)) or let it reach the host. Every kind is
  listed in [reference/types/error.md § Every error kind](reference/types/error.md#every-error-kind).
- Some modules also export constants (for example `math`, `fs`, `os`).

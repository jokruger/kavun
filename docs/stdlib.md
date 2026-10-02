# Standard Library

This document covers the main builtin modules in Kavun stdlib:

- `base64`
- `fin`
- `fmt`
- `hex`
- `json`
- `math`
- `os`
- `rand`
- `regexp`
- `times`

Notes:

- Signatures below use Kavun-facing names and argument order.
- A module function either answers the documented value or **raises** — nothing answers an error value, and
  nothing answers a `true` that means "it worked". A function whose success has no result answers `undefined`.
  Each module's **Failures** line below names the kind it raises; catch one with `defer`/`recover()` (see
  [language.md § Errors and recovery](language.md#errors-and-recovery)) or let it reach the host. Every kind is
  listed in [types/error.md § Every error kind](types/error.md#every-error-kind).
- Some modules also export constants (for example `math`, `times`, `os`).

## base64

Example:

```go
base64 = import("base64")
base64.encode(bytes("hello"))
```

- `base64.encode(data bytes) -> string`: Standard Base64 encode.
- `base64.decode(s string) -> bytes`: Standard Base64 decode.
- `base64.raw_encode(data bytes) -> string`: Raw standard Base64 encode (no padding).
- `base64.raw_decode(s string) -> bytes`: Raw standard Base64 decode.
- `base64.url_encode(data bytes) -> string`: URL-safe Base64 encode.
- `base64.url_decode(s string) -> bytes`: URL-safe Base64 decode.
- `base64.raw_url_encode(data bytes) -> string`: Raw URL-safe Base64 encode (no padding).
- `base64.raw_url_decode(s string) -> bytes`: Raw URL-safe Base64 decode.

**Failures.** A malformed input raises kind `conversion`, with the function name in the message: `(base64.decode) illegal base64 data at input byte 0`.

## fin

Deterministic financial arithmetic — time value of money, cashflows, depreciation, rate conversion — backed by
[fin128](https://github.com/jokruger/fin128) over the same dec128 library `decimal` uses. Every function either
answers an exact `decimal` at the rounding you stated or raises; nothing is approximated, nothing defaults.

```go
fin := import("fin")
fin.payment(0.005d, 60, 25000, 0, "arrears", 2, "half_even")   // -483.32 — a 60-month loan at 0.5% a month
```

**The argument contract**, shared by every function:

- Money, rates and factors are `decimal` or `int`. A `float` **raises** (`argument rate expects type decimal or
  int, got float`) — binary floating point cannot spell most decimal rates, and an approximate rate is a wrong
  posting. Counts (`n`, `period`, `life`, `m`, `num`, `den`) are `int`.
- Every quantizing function **ends with `(scale, mode)`**, read exactly as `decimal`'s `*_round` members read
  them: scale 0…19, mode one of `ceiling`, `floor`, `down`, `up`, `half_down`, `half_up`, `half_even`. There is no
  default scale or mode.
- Enum arguments are **exact canonical names** (case-sensitive): `timing` is `"arrears"` (payments at period end)
  or `"advance"` (at period start); `frequency` is `"annual"`, `"semiannual"`, `"quarterly"`, `"monthly"`,
  `"weekly"` or `"daily"`. An unknown name raises and lists the accepted ones.
- Sign convention: money flowing **in** is positive, **out** negative — a positive `pv` (the amount borrowed)
  gives a negative `payment` (the amount paid back).
- The solver-driven functions (`rate`, `irr`) take an optional **solver spec** just before `(scale, mode)`: a
  record or dict with any of `lo`, `hi` (the search bracket), `tolerance` (in ulps at the output scale) and
  `max_iter`, laid over the frozen defaults `fin.default_solver()` answers (`{lo: -0.9999d, hi: 9, tolerance: 1,
  max_iter: 100}`). An unknown key raises. The solver is bisection, so its answer depends only on the bracket and
  the tolerance.

Functions (`s, m` is the trailing `(scale, mode)` pair):

| group | functions |
| --- | --- |
| time value of money | `payment(rate, n, pv, fv, timing, s, m)`, `present_value(rate, n, pmt, fv, timing, s, m)`, `future_value(rate, n, pmt, pv, timing, s, m)`, `periods(rate, pmt, pv, fv, timing, s, m)` → `[whole, remainder]`, `rate(n, pmt, pv, fv, timing, [solver,] s, m)`, `annuity_factor_pv(rate, n, timing, s, m)`, `annuity_factor_fv(rate, n, timing, s, m)` |
| schedule (period `1…n`) | `balance(rate, period, n, pv, fv, timing, s, m)` — the balance after the period; `charge_part(…)` / `principal_part(…)` — the period payment's interest and principal parts |
| cashflows | `npv(rate, cf, s, m)`, `irr(cf, [solver,] s, m)`, `mirr(cf, finance_rate, reinvest_rate, s, m)` — `cf` is an array of `decimal|int`, one per period |
| depreciation | `straight_line(cost, salvage, life, s, m)`, `sum_of_digits(cost, salvage, life, period, s, m)`, `declining_balance(book, salvage, rate, s, m)`, `declining_rate_from_factor(factor, life, s, m)`, `declining_rate_from_salvage(cost, salvage, life, s, m)` |
| rates and factors | `nominal_to_effective(nominal, m, s, mode)`, `effective_to_nominal(effective, m, s, mode)` (`m` compounding periods a year), `compound_factor(rate, n, s, m)`, `discount_factor(rate, n, s, m)`, `per_year(frequency)` → `int` |
| day count | `year_fraction(n1, d1[, n2, d2])` (the constructor), `year_fraction_between(start, end, convention)`, `year_fraction_between_final(…)`, `days_between(start, end, convention)`, `days_between_final(…)`, `days_in_year(on, convention)`, `conventions()`, `is_year_fraction(x)` — see [fin.year_fraction](#finyear_fraction) |
| interest over a fraction | `accrue_simple(principal, rate, f, s, m)`, `accrue_compound(principal, rate, f, per_year, s, m)`, `compound_factor_for(rate, f, s, m)`, `discount_factor_for(rate, f, s, m)` |
| discount instruments | `discount_price(redemption, rate, f, s, m)`, `discount_rate(price, redemption, f, s, m)`, `discount_yield(price, redemption, f, s, m)` |
| dated cashflows | `xnpv(rate, flows, convention, s, m)`, `xirr(flows, convention, [solver,] s, m)` — `flows` is an array of `{date, amount}`, ascending by date |
| exact helpers | `apply_rate(amount, rate, s, m)` (amount × rate, rounded once), `exact_product(a, b)` (exact, or raises), `mul_div_round(a, b, num, den, s, m)` (a × b × num / den, rounded once) |
| units (exact) | `to_percent(v)`, `from_percent(v)`, `to_basis_points(v)`, `from_basis_points(v)` |
| solver | `default_solver()` → record |

```go
fin.present_value(0.005d, 60, -483.32d, 0, "arrears", 2, "half_even")   // 25000.00
n, rest := fin.periods(0.005d, -500, 25000, 0, "arrears", 6, "half_even")   // 57, 0.679593d
fin.rate(60, -483.32d, 25000, 0, "arrears", 10, "half_even")             // 0.0049999973
fin.rate(60, -483.32d, 25000, 0, "arrears", {hi: 1}, 10, "half_even")    // a narrower bracket, same answer
fin.balance(0.005d, 12, 60, 25000, 0, "arrears", 2, "half_even")         // 20579.92 — owed after a year
fin.npv(0.1d, [-1000, 300, 400, 500], 8, "half_even")                    // -21.03681443
fin.irr([-1000, 300, 400, 500], 10, "half_even")                         // 0.0889633947
fin.straight_line(10000, 1000, 9, 2, "half_even")                        // 1000.00 a year
fin.nominal_to_effective(0.12d, 12, 10, "half_even")                     // 0.1268250301
```

Spreadsheet equivalents, for migrations: `payment` ≈ `PMT`, `present_value` ≈ `PV`, `future_value` ≈ `FV`,
`periods` ≈ `NPER` (split into whole periods and a remainder), `rate` ≈ `RATE`, `charge_part` ≈ `IPMT`,
`principal_part` ≈ `PPMT`, `straight_line` ≈ `SLN`, `sum_of_digits` ≈ `SYD`. **`npv` is not the spreadsheet's
`NPV`**: here `cf[0]` sits at t = 0 and is not discounted (the definition, as numpy-financial has it); a
spreadsheet discounts its first value by one period. Pass the initial outlay as `cf[0]`, or shift the array.

Quantize then recompute: a function's result is already rounded to the scale you asked for, and each function
rounds its own result once. So two results rounded independently — `balance` and `principal_part` of the same
period — can disagree by a cent with their difference. Build a schedule from **one** rounded series and derive the
rest from it (see [the loan example](examples.md#a-loan-schedule-with-fin)).

### fin.year_fraction

A length of time **in years**, as an exact ratio: the days a day-count convention counts over the year length it
takes. It is the `f` the interest functions take, kept exact so an accrual never rounds the period before it
multiplies. The type is named after its constructor (`type_name(f)` is `"fin.year_fraction"`) because it exists
only for this module.

```go
f := fin.year_fraction_between(d"2026-01-01", d"2026-02-01", "ACT/365F")   // 31/365
fin.year_fraction_between(d"2026-02-01", d"2026-03-01", "30E/360")        // 30/360 — every month is 30 days
fin.year_fraction_between(d"2027-07-01", d"2028-07-01", "ACT/ACT")        // 184/365+182/366 — split at the year end
fin.year_fraction(31, 365)                 // stated directly: 31 counted days over a 365-day year
fin.year_fraction()                        // 0/1, the default — falsy
fin.accrue_simple(10000, 0.05d, f, 2, "half_even")   // 42.47 — interest for January
```

| member / operator | result |
| --- | --- |
| `f.value(scale, mode)` | the decimal reading, at a stated rounding — the only bridge to `decimal` |
| `f.terms()` | `[n1, d1]` or `[n1, d1, n2, d2]`; `fin.year_fraction(f.terms()...) == f` always holds |
| `f.string()`, `f"{f}"` | `"31/365"`, `"184/365+182/366"` |
| `f.is_zero()`, `f.is_negative()`, `f.is_positive()` | a period running backwards is negative |
| `f + g` | the exact sum |
| `f * n`, `n * f` (`n` an `int`) | scaled — e.g. a year fraction as compounding periods |
| `f < g` (`<=` `>` `>=`), `f == g` | exact, numeric: `fin.year_fraction(31, 365) == fin.year_fraction(62, 730)` |
| `f - g`, `f / n`, `f * f`, `f ± decimal`, `f < 1` | raise — use `f.value(…)` to leave the type |

`fin.year_fraction(n1, d1[, n2, d2])` takes `int`s: numerators within `int32`, any sign; denominators **1…65535**
(fin128's own output stays within that, so every value can be rebuilt from its terms). A sum or scale that would
leave those bounds raises `invalid_value`. JSON is the terms array, `[184, 365, 182, 366]`; `json.decode` gives
an array back, which `fin.year_fraction(v...)` rebuilds.

**Conventions** are named by their market spelling and read through fin128's alias table, so `"ACT/365F"`,
`"30/360 US"` or `"Actual/360"` work as products store them; `fin.conventions()` lists the canonical names
(`30B/360`, `30E-ISDA/360`, `30E/360`, `30E3/360`, `30U/360`, `ACT/360`, `ACT/365`, `ACT/366`, `ACT/ACT`,
`NL/365`). An unknown name raises and lists them. The `_final` variants take `end` as the contract's termination
date — only `30E-ISDA/360` distinguishes it, when the termination falls at the end of February.

`days_between` and `days_in_year` answer the **convention's** count, not the calendar's:
`fin.days_between(d"2026-02-01", d"2026-03-01", "30E/360")` is `30` (the calendar's `d2 - d1` is `28`), and
`fin.days_in_year(d"2028-03-01", "ACT/360")` is `360` (the calendar's `times.days_in_year(2028)` is `366`).

**Dated cashflows** discount each flow by the year fraction from the **first** flow's date, so the flows must be in
ascending date order — an unsorted array raises rather than being sorted for you:

```go
flows := [{date: d"2026-01-01", amount: -10000}, {date: d"2026-06-30", amount: 2750},
          {date: d"2027-01-01", amount: 4250}, {date: d"2027-06-30", amount: 3250},
          {date: d"2028-01-01", amount: 2750}]
fin.xnpv(0.09d, flows, "ACT/365F", 2, "half_even")   // 1706.87
fin.xirr(flows, "ACT/365F", 10, "half_even")         // 0.2449124168
```

Each flow is a record or dict with exactly `date` (a `date`) and `amount` (`decimal|int`).

Not yet available: conventions outside the named set (a custom year length, a host-defined name table, a
script-defined convention) — a stated fraction, `fin.year_fraction(n, d)`, covers the functions that take `f`.

**Failures.** Data the formula cannot use raises `invalid_value` with fin128's reason —
`(fin.payment) the number of periods must be positive`, `(fin.irr) the bracket does not contain a root`,
`(fin.straight_line) the salvage value is above the cost`. A wrong argument type raises `invalid_argument_type`,
a wrong count `wrong_num_arguments`, a division by zero `division_by_zero`.

## fmt

Example:

```go
fmt = import("fmt")
fmt.println("sum:", 20 + 22)
```

- `fmt.print(values...) -> undefined`: Print values without newline.
- `fmt.println(values...) -> undefined`: Print values with newline.

## hex

Example:

```go
hex = import("hex")
hex.encode(bytes("ok"))
```

- `hex.encode(data bytes) -> string`: Hex-encode bytes.
- `hex.decode(s string) -> bytes`: Hex-decode string.

**Failures.** A malformed input raises kind `conversion`: `(hex.decode) encoding/hex: invalid byte …`.

## json

Example:

```go
json = import("json")
json.encode({"a": 1, "b": true})
```

- `json.decode(data bytes|string) -> value`: Decode JSON bytes into Kavun value.
- `json.encode(value) -> bytes`: Encode Kavun value into JSON bytes.
- `json.indent(data bytes|string, prefix string, indent string) -> bytes`: Pretty-format JSON bytes.
- `json.html_escape(data bytes|string) -> bytes`: Escape JSON for safe HTML embedding.

**Failures.** `decode` and `indent` raise kind `json_decoding` on malformed input. `encode` raises kind
`json_encoding` on a value with no JSON representation, naming the path to it once — `.items[0].price: value type
<compiled-function/0> does not support JSON encoding` — and on text holding octets that are not symbols (JSON is
UTF-8 by definition; encode such text as `bytes`, which goes as base64).

## math

`min`/`max` are not module functions: selection over arguments is the free variadic `min(a, b, ...)` /
`max(a, b, ...)`, and aggregation over elements is the member (`arr.min()`, `arr.max()`).

Example:

```go
math = import("math")
math.sqrt(144)
```

Constants:

- Core numeric constants: `e`, `pi`, `phi`, `sqrt2`, `sqrt_e`, `sqrt_pi`, `sqrt_phi`, `ln2`, `log2e`, `ln10`, `log10e`.
- Float bounds: `max_float32`, `smallest_nonzero_float32`, `max_float64`, `smallest_nonzero_float64`.
- Integer bounds: `max_int`, `min_int`, `max_int8`, `min_int8`, `max_int16`, `min_int16`, `max_int32`, `min_int32`, `max_int64`, `min_int64`.

- `math.abs(x float) -> float`: Absolute value.
- `math.acos(x float) -> float`: Arc cosine.
- `math.acosh(x float) -> float`: Inverse hyperbolic cosine.
- `math.asin(x float) -> float`: Arc sine.
- `math.asinh(x float) -> float`: Inverse hyperbolic sine.
- `math.atan(x float) -> float`: Arc tangent.
- `math.atan2(y float, x float) -> float`: Arc tangent of y/x with quadrant.
- `math.atanh(x float) -> float`: Inverse hyperbolic tangent.
- `math.cbrt(x float) -> float`: Cube root.
- `math.ceil(x float) -> float`: Smallest integer value >= x.
- `math.copy_sign(f float, sign float) -> float`: Magnitude of f with sign of sign.
- `math.cos(x float) -> float`: Cosine.
- `math.cosh(x float) -> float`: Hyperbolic cosine.
- `math.dim(x float, y float) -> float`: Max(x-y, 0).
- `math.erf(x float) -> float`: Error function.
- `math.erfc(x float) -> float`: Complementary error function.
- `math.exp(x float) -> float`: e\*\*x.
- `math.exp2(x float) -> float`: 2\*\*x.
- `math.expm1(x float) -> float`: e\*\*x - 1 with precision for small x.
- `math.floor(x float) -> float`: Greatest integer value <= x.
- `math.gamma(x float) -> float`: Gamma function.
- `math.hypot(p float, q float) -> float`: sqrt(p*p + q*q).
- `math.ilogb(x float) -> int`: Binary exponent as integer.
- `math.inf(sign int) -> float`: +/- infinity by sign.
- `math.is_inf(x float, sign int) -> bool`: Infinity check with sign filter.
- `math.is_nan(x float) -> bool`: NaN check.
- `math.j0(x float) -> float`: Bessel J0.
- `math.j1(x float) -> float`: Bessel J1.
- `math.jn(n int, x float) -> float`: Bessel Jn.
- `math.ldexp(frac float, exp int) -> float`: frac \* 2\*\*exp.
- `math.log(x float) -> float`: Natural logarithm.
- `math.log10(x float) -> float`: Base-10 logarithm.
- `math.log1p(x float) -> float`: log(1+x) with precision for small x.
- `math.log2(x float) -> float`: Base-2 logarithm.
- `math.logb(x float) -> float`: Binary exponent as float.
- `math.mod(x float, y float) -> float`: Floating-point remainder.
- `math.nan() -> float`: NaN value.
- `math.next_after(x float, y float) -> float`: Next representable float from x toward y.
- `math.pow(x float, y float) -> float`: x\*\*y.
- `math.pow10(n int) -> float`: 10\*\*n.
- `math.remainder(x float, y float) -> float`: IEEE 754 remainder.
- `math.signbit(x float) -> bool`: True if sign bit is set.
- `math.sin(x float) -> float`: Sine.
- `math.sinh(x float) -> float`: Hyperbolic sine.
- `math.sqrt(x float) -> float`: Square root.
- `math.tan(x float) -> float`: Tangent.
- `math.tanh(x float) -> float`: Hyperbolic tangent.
- `math.trunc(x float) -> float`: Integer part toward zero.
- `math.y0(x float) -> float`: Bessel Y0.
- `math.y1(x float) -> float`: Bessel Y1.
- `math.yn(n int, x float) -> float`: Bessel Yn.

## os

Example:

```go
os = import("os")
os.read_file("./README.md")
```

Constants:

- Platform/path: `platform`, `arch`, `dev_null`, `path_separator`, `path_list_separator`.
- Open flags: `o_rd`, `o_wr`, `o_rdwr`, `o_append`, `o_create`, `o_excl`, `o_sync`, `o_trunc`.
- File mode bits: `mode_dir`, `mode_append`, `mode_exclusive`, `mode_temporary`, `mode_symlink`, `mode_device`, `mode_named_pipe`, `mode_socket`, `mode_set_uid`, `mode_set_gui`, `mode_char_device`, `mode_sticky`, `mode_type`, `mode_perm`.
- Seek modes: `seek_set`, `seek_cur`, `seek_end`.

- `os.args() -> [string]`: Command-line arguments.
- `os.chdir(dir string) -> undefined`: Change current working directory.
- `os.chmod(path string, mode int) -> undefined`: Change file mode bits.
- `os.chown(path string, uid int, gid int) -> undefined`: Change owner and group.
- `os.clear_env() -> undefined`: Clear all environment variables.
- `os.environ() -> [string]`: Environment as `KEY=VALUE` strings.
- `os.exit(code int) -> undefined`: Exit process with code.
- `os.expand_env(s string) -> string`: Expand `$VAR` references.
- `os.get_egid() -> int`: Effective GID.
- `os.get_env(key string) -> string`: Environment value (empty if missing).
- `os.get_euid() -> int`: Effective UID.
- `os.get_gid() -> int`: Real GID.
- `os.get_groups() -> [int]`: Supplementary group IDs.
- `os.get_page_size() -> int`: Memory page size.
- `os.get_pid() -> int`: Current process ID.
- `os.get_ppid() -> int`: Parent process ID.
- `os.get_uid() -> int`: Real UID.
- `os.get_wd() -> string`: Current working directory.
- `os.hostname() -> string`: Hostname.
- `os.lchown(path string, uid int, gid int) -> undefined`: Change owner/group of symlink target entry.
- `os.link(old_path string, new_path string) -> undefined`: Create hard link.
- `os.lookup_env(key string) -> string | false`: Lookup env var with presence flag.
- `os.mkdir(path string, perm int) -> undefined`: Create directory.
- `os.mkdir_all(path string, perm int) -> undefined`: Create directory tree.
- `os.read_link(path string) -> string`: Read symlink target.
- `os.remove(path string) -> undefined`: Remove file or empty directory.
- `os.remove_all(path string) -> undefined`: Remove path recursively.
- `os.rename(old_path string, new_path string) -> undefined`: Rename/move path.
- `os.set_env(key string, value string) -> undefined`: Set environment variable.
- `os.symlink(old_path string, new_path string) -> undefined`: Create symbolic link.
- `os.temp_dir() -> string`: System temporary directory.
- `os.truncate(path string, size int) -> undefined`: Truncate file.
- `os.unset_env(key string) -> undefined`: Unset environment variable.
- `os.create(path string) -> file`: Create file, returns file record.
- `os.open(path string) -> file`: Open file (read-only), returns file record.
- `os.open_file(path string, flag int, perm int) -> file`: Open file with flags/mode, returns file record.
- `os.find_process(pid int) -> process`: Find process by PID.
- `os.start_process(name string, argv [string], dir string, env [string]) -> process`: Start process.
- `os.exec_look_path(file string) -> string`: Search executable in PATH.
- `os.exec(name string, args...) -> command`: Build exec command record.
- `os.stat(path string) -> fileinfo`: File metadata record.
- `os.read_file(path string) -> bytes`: Read file contents.

**Failures.** Everything the world can refuse — a missing path, a permission, a failed exec — raises kind
`io`, with the operation named in the message: `(os.remove) remove /nope: no such file or directory`. There is
nothing to check after a call; a function whose success has no result answers `undefined`. The same holds for the
`file`, `process` and `exec` objects these functions answer.

### os returned records

- `file` record methods:
  - `chdir()`, `chown(uid, gid)`, `close()`, `name()`, `read_dir_names(n)`, `sync()`, `write(bytes)`, `write_string(string)`, `read(bytes)`, `chmod(mode)`, `seek(offset, whence)`, `stat()`.
- `process` record methods:
  - `kill()`, `release()`, `signal(sig)`, `wait() -> process_state`.
- `process_state` record methods:
  - `exited()`, `pid()`, `string()`, `success()`.
- `command` record methods (`os.exec(...)`):
  - `combined_output()`, `output()`, `run()`, `start()`, `wait()`, `set_path(path)`, `set_dir(dir)`, `set_env(env)`, `process()`.

## rand

Example:

```go
rand = import("rand")
rand.int_n(100)
```

- `rand.int() -> int`: Random 63-bit integer.
- `rand.float() -> float`: Random float in `[0.0, 1.0)`.
- `rand.int_n(n int) -> int`: Random integer in `[0, n)`.
- `rand.exp_float() -> float`: Exponential distribution sample.
- `rand.norm_float() -> float`: Normal distribution sample.
- `rand.perm(n int) -> [int]`: Random permutation of `[0..n)`.
- `rand.seed(seed int) -> undefined`: Seed global generator.
- `rand.read(buf bytes) -> int`: Fill byte buffer with random data, return bytes written.
- `rand.rand(seed int) -> rng`: Create independent RNG record.

**Failures.** A failure of the entropy source raises kind `io`: `(rand.read) …`.

### rand rng record

`rng` has the same callable methods as module-level random generator:

- `int()`, `float()`, `int_n(n)`, `exp_float()`, `norm_float()`, `perm(n)`, `seed(seed)`, `read(buf)`.

## regexp

What remains of the former `text` module: everything string-shaped became member functions on
`string`/`runes`/`bytes` (see the per-type pages), and only the five regex functions stay module-shaped.

Example:

```go
re = import("regexp")
re.re_match("[0-9]+", "abc123")   // true
```

- `regexp.re_match(pattern string, s string) -> bool`: Regex full/partial match check.
- `regexp.re_find(pattern string, s string, count? int) -> [match] | undefined`: Regex find with optional limit; each match is a list of `{text, begin, end}` group records.
- `regexp.re_replace(pattern string, s string, repl string) -> string`: Regex replace all (`$1` group references).
- `regexp.re_split(pattern string, s string, count? int) -> [string]`: Regex split with optional limit.
- `regexp.re_compile(pattern string) -> regexp`: Compile once into a reusable object with `match(s)`, `find(s [,count])`, `replace(s, repl)`, `split(s [,count])` methods.

**Failures.** An invalid pattern raises kind `invalid_value` — a bad pattern is a bad argument, not an
exhausted resource: `(regexp.re_compile) error parsing regexp: missing closing ): `(``.

Where the old `text` functions went:

| old | now |
| --- | --- |
| `text.contains/count/index/last_index/has_prefix/has_suffix/replace/repeat/split/fields/join/title/pad_*/trim*` | member functions: `s.contains(x)`, `s.count(x)`, `s.index(x)`, `s.index_last(x)`, `s.has_prefix(p)`, `s.replace(old, new)`, `s.repeat(n)`, `s.split(...seps)`, `arr.join(sep)`, `s.title_case()`, `s.pad_start/pad_end(n [,fill])`, `s.trim/trim_start/trim_end(...set)`, `s.remove_prefix/remove_suffix(run)` |
| `text.to_lower/to_upper` | `s.lower()`, `s.upper()` |
| `text.equal_fold(a, b)` | `a.case_fold() == b.case_fold()` |
| `text.atoi/itoa/parse_*` | conversions: `s.int()`, `i.string()`, `s.bool()`, `s.float()` — each takes an optional trailing default |
| `text.format_bool/format_float/format_int` | `format()` / `x.format(spec)` (arbitrary radix beyond the format verbs is not currently expressible) |
| `text.compare(a, b)` | the comparison operators (`<`, `<=`, `>`, `>=`) |
| `text.substr(s, i, j)` | `s.slice(i, j)` / `s[i:j]` |
| `text.contains_any/index_any/last_index_any` | `s.contains(a, b, ...)` (variadic set); the locator forms are a predicate: `s.index(func(c) { return c == 'x' \|\| c == 'y' })` |
| `text.split_n/split_after/split_after_n` | `partition(...seps)` covers the split-once use; keep-separator and n-way splits have no member form |
| `text.quote/unquote` | removed with no successor — `format()` is the render surface, `json.encode` the interop one |

## times

Example:

```go
times = import("times")
times.now().in_zone("Europe/Kyiv").format("#datetime")
```

The module holds what needs **no time value**: the clock, durations, the unit-named epoch constructors,
constants, and calendar facts about plain numbers. Construction, parsing, zones and calendar arithmetic are on
the type itself — `time(…)`, `t.in_zone(z)`, `t.add_months(n, eom)`; see [time](types/time.md).

Constants:

- Duration units (nanoseconds): `nanosecond`, `microsecond`, `millisecond`, `second`, `minute`, `hour`.
- Months: `january`, `february`, `march`, `april`, `may`, `june`, `july`, `august`, `september`, `october`, `november`, `december`.

Every `int` in this module is one of three things, and the function name says which: a **duration in
nanoseconds** (`sleep`, `parse_duration`, `since`, `until`, `duration_*`), a **unix timestamp** in the
encoding the name states (`unix`, `from_unix_*`), or a **calendar number** — a year, a month
(`is_leap_year`, `days_in_year`, `days_in_month`). This mirrors the operator/conversion split on the `time` type
itself — see [time](types/time.md#overview).

- `times.sleep(duration int) -> undefined`: Sleep for duration (nanoseconds).
- `times.parse_duration(s string) -> int`: Parse duration string to nanoseconds.
- `times.since(t time) -> int`: Elapsed duration since time (nanoseconds).
- `times.until(t time) -> int`: Duration until time (nanoseconds).
- `times.duration_hours(d int) -> float`: Duration to hours.
- `times.duration_minutes(d int) -> float`: Duration to minutes.
- `times.duration_nanoseconds(d int) -> int`: Duration to nanoseconds.
- `times.duration_seconds(d int) -> float`: Duration to seconds.
- `times.duration_string(d int) -> string`: Duration text format.
- `times.now() -> time`: The current instant, in **UTC** — the host's zone is never visible; name a zone with
  `times.now().in_zone(z)`.
- `times.unix(sec int, nsec int) -> time`: Unix seconds + nanoseconds to time (UTC).
- `times.from_unix_ms(msec int) -> time`: Unix milliseconds to time (UTC).
- `times.from_unix_micro(usec int) -> time`: Unix microseconds to time (UTC).
- `times.from_unix_nano(nsec int) -> time`: Unix nanoseconds to time (UTC). Seconds are the conversion `(n).time()`.
- `times.is_leap_year(year int) -> bool`: Whether the year is a leap year.
- `times.days_in_year(year int) -> int`: 365 or 366.
- `times.days_in_month(year int, month int) -> int`: 28…31 — a month has no length without its year.

The calendar facts take **ints only**; a `date`/`time` answers the same questions as members
(`t.is_leap_year()`, `t.days_in_year()`, `t.days_in_month()`), and passing one here raises:

```go
times.days_in_month(2027, 2)               // 28
times.is_leap_year(2000)                   // true
times.is_leap_year(t"2028-01-01T00:00:00Z")  // raises: argument first expects type int, got time
```

**Failures.** An unparsable duration raises kind `conversion`
(`(times.parse_duration) time: invalid duration "x"`); a year outside 1…9999 or a month outside 1…12 raises
`invalid_value`.

Where the removed functions went:

| removed | now |
| --- | --- |
| `times.date(y, mo, d, h, mi, s, ns[, zone])` | `time({year, month, day, hour, minute, second, nanosecond, zone})` — no normalization; a DST-gap or overlap wall clock raises |
| `times.parse(layout, s)` | `time(s, layout)` / `s.time(layout[, default])` with `%` directives |
| `times.format_*` (Go reference layouts) | the format aliases (`#iso`, `#datetime`, …) and `%` templates |
| `times.add_date(t, y, m, d)` | `t.add_years(y, eom)`, `t.add_months(m, eom)`, `t.add_days(d)` |
| `times.in_location(t, z)` | `t.in_zone(z)` |
| `times.from_unix(n)` | `(n).time()` |

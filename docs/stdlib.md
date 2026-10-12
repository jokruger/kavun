# Standard Library

This document covers the main builtin modules in Kavun stdlib:

- `base64`
- `clock`
- `exec`
- `fin`
- `fs`
- `hex`
- `io`
- `json`
- `math`
- `os`
- `path`
- `rand`
- `re`

Notes:

- Each module is a value of type [`module`](types/module.md): `import("math")` answers it, its functions and
  constants are read-only members, and a missing name raises.
- **Grant.** `base64`, `fin`, `hex`, `json`, `math`, `path`, `rand` and `re` are *pure* (they only compute) and
  importable by default. `clock`, `exec`, `fs`, `io` and `os` touch the world and must be granted by the
  host (`docs/embedding.md` § The module grant); the `kavun` CLI grants every module.
- Signatures below use Kavun-facing names and argument order.
- A module function either answers the documented value or **raises** — nothing answers an error value, and
  nothing answers a `true` that means "it worked". A function whose success has no result answers `undefined`.
  Each module's **Failures** line below names the kind it raises; catch one with `defer`/`recover()` (see
  [language.md § Errors and recovery](language.md#errors-and-recovery)) or let it reach the host. Every kind is
  listed in [types/error.md § Every error kind](types/error.md#every-error-kind).
- Some modules also export constants (for example `math`, `fs`, `os`).

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

## clock

The world's time: the current instant, the distance to one, and waiting. It is the only place a script reads the
wall clock, so it needs a grant.

```go
clock := import("clock")
clock.now().in_zone("Europe/Kyiv").format("#datetime")
start := clock.now()
clock.sleep(250.milliseconds())
clock.since(start).in_seconds()     // ≈ 0.25
```

- `clock.now() -> time`: The current instant, in UTC.
- `clock.since(t) -> int`: The duration from `t` to now, in nanoseconds.
- `clock.until(t) -> int`: The duration from now to `t`, in nanoseconds.
- `clock.sleep(d) -> undefined`: Wait `d` nanoseconds (`clock.sleep(2.seconds())`).

Durations are **int members**, not module functions: `30.seconds()`, `1.5.hours()`, `d.in_minutes()`,
`d.duration_string()`, `"1h30m".nanoseconds()` (see [types/int.md § Durations](types/int.md#durations)).
Construction, parsing, zones, calendar arithmetic and calendar facts are on the `time` and `date` types —
`time(…)`, `t.in_zone(z)`, `t.add_months(n, eom)`, `t.is_leap_year()`.

**Migration:** the module was `times`, and kept only these four functions. `30 * times.second` → `30.seconds()`;
`times.parse_duration(s)` → `s.nanoseconds()`; `times.duration_hours(d)` (and the other units) → `d.in_hours()`;
`times.duration_string(d)` → `d.duration_string()`; `times.unix(sec, nsec)` → `sec.time() + nsec`;
`times.from_unix_ms(n)` → `n.time_ms()` (likewise `_micro`, `_nano`); `times.is_leap_year(y)`, `days_in_year`,
`days_in_month` → the `date` members; the month constants → integer literals.

## exec

Run external programs. World access: a host must grant it.

```go
exec := import("exec")
string(exec.exec("echo", "hi").output())   // "hi\n" — output() answers bytes
p := exec.start_process("/bin/sleep", ["sleep", "1"], "", [])
p.wait().success                        // true
```

- `exec.exec(name string, args...) -> exec.command`: Prepare a command; configure it, then run it once.
- `exec.exec_look_path(file string) -> string`: Search an executable in `PATH`.
- `exec.start_process(name string, argv [string], dir string, env [string]) -> exec.process`: Start a process.
- `exec.find_process(pid int) -> exec.process`: The process with that pid.

The two types are documented on their own pages: [`exec.command`](types/exec-command.md) (`run`, `start`, `wait`,
`output`, `combined_output`, `set_path`, `set_dir`, `set_env`, `process`) and
[`exec.process`](types/exec-process.md) (`pid`, `kill`, `signal`, `wait`, `release`). `process.wait()` answers a
frozen record of facts: `{pid, exited, success, status}`.

**Failures.** A program that cannot be found or run, or ends badly for `run`/`output`, raises `io` naming the call
(`(exec.command.run) exit status 1`).

**Migration:** these were `os.exec`, `os.exec_look_path`, `os.start_process` and `os.find_process`. The command and
process are types instead of records of closures; the wait result is a record of VALUES — `st.success`, not
`st.success()` — and its text field is `status` (was `string()`).

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
| rate and charge tables | `tiered_rates(text \| bands)`, `tiered_charges(text \| bands, min, max)`, `dated_rates(text \| bands)`, `dated_charges(text \| bands)`, and `is_tiered_rates(x)` … — see [Rate and charge tables](#rate-and-charge-tables) |
| exact helpers | `apply_rate(amount, rate, s, m)` (amount × rate, rounded once), `exact_product(a, b)` (exact, or raises), `mul_div_round(a, b, num, den, s, m)` (a × b × num / den, rounded once) |
| units (exact) | `to_percent(v)`, `from_percent(v)`, `to_basis_points(v)`, `from_basis_points(v)` |
| solver | `root(f, [solver,] s, m)` — x where `f(x) = 0`; `brackets(f[, solver])` → `bool`; `default_solver()` → record |

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

### The solver: `root` and `brackets`

`fin.root(f, [solver,] scale, mode)` finds the `x` where a script function crosses zero, by bisection over the
solver bracket — the same solver `rate`, `irr` and `xirr` use, so its answer depends only on the bracket and the
tolerance. `fin.brackets(f[, solver])` answers whether `f` changes sign across the bracket at all: it tells a
bracket that merely missed (widen `hi`) from a function with no root.

```go
fin.root(x => x * x - 2, 10, "half_even")                     // 1.4142135624
fin.root(r => fin.npv(r, flows, 12, "half_even"), 10, "half_even")   // irr, spelled out
fin.brackets(x => x - 20)                                     // false — the default bracket stops at 9
fin.root(x => x - 20, {hi: 50}, 10, "half_even")              // 20.0000000000
```

The callback contract:

- `f` takes **exactly one** argument, a `decimal`, and returns a `decimal` or `int` — anything else raises
  `invalid_argument_type`.
- A raise inside `f` stops the search and surfaces **as itself** — the script sees its own error, never "the bracket
  does not contain a root". After the first raise `f` is not called again. A fatal error (`stack_overflow`) stays
  fatal.
- `f` is called at most `max_iter + 2` times (102 with the defaults).
- `f` may itself call `fin.root`. The result is deterministic only if `f` is: a callback that reads the clock or
  other host state makes the answer depend on it. Calls that run a callback are never folded at compile time.

### Rate and charge tables

Four immutable, validated table types for product parameters — a savings ladder, a fee schedule, a teaser rate
that reverts, a standing charge that changes on a date. Each is built once, from a one-line **text form** (what a
product configuration stores) or from **band records** (when a script computes the table), and then asked
questions. Like `fin.year_fraction`, each type is named after its constructor.

| type | text form | band record |
| --- | --- | --- |
| `fin.tiered_rates` | `"0:0.005, 1000:0.007, 10000:0.009"` — from 0, open at the top | `{from, rate}` |
| `fin.tiered_charges` | `"0:0.015+2.00, 1000:0.01; min=25, max=500"` — a rate plus a fixed amount per band, the charge clamped | `{from, rate, fixed}`, with `(bands, min, max)`; `max` 0 = no cap |
| `fin.dated_rates` | `"2026-01-01:0.04, 2026-07-01:0.02"` — each band from a date to the next | `{from: date, rate}` |
| `fin.dated_charges` | `"2024-01-01:25.00, 2025-01-01:30.00"` | `{from: date, amount}` |

A rate is written as a rate, never a percent (`0.007` is 0.7%), and the digits are kept as written (`"0:0.50"`
stays `"0:0.50"`). Values are `decimal|int` — a `float` raises. Every ordering and anchoring rule (ascending
bands, the first tiered band at 0, `max` not below `min`) is fin128's, and raises the same way from text or
records. The empty text raises: a table always has a band.

| member | `tiered_rates` | `tiered_charges` | `dated_rates` | `dated_charges` |
| --- | --- | --- | --- | --- |
| `at(x)` | the band `amount` falls in → `{from, rate}` | → `{from, rate, fixed}` | the rate in force `on` (`at(on, fallback)`: the fallback answers **only** a date before the first band) | the amount in force `on` |
| `charge(amount, rule, s, m)` | ✓ | ✓ (clamped) | — | — |
| `charge_parts(amount, rule, s, m)` | each band's slice → `[{from, to, rate, fixed, amount}]` | ✓ | — | — |
| `rate(amount, rule, s, m)` | the applied rate (blended under `"marginal"`) — for statements, not postings | — | — | — |
| `accrue(…)` | `(principal, f, rule, s, m)` over a year fraction | — | `(principal, start, end, convention, s, m)`, the rate changing at band dates | — |
| `accrue_parts(…)` | — | — | each stretch → `[{start, end, rate, amount}]` | — |
| `apply(amount, on, [fallback,] s, m)` | — | — | amount × the rate in force, rounded once | — |
| `bounds()` | — | `{min, max}` | — | — |
| `bands()`, `string()` | ✓ | ✓ | ✓ | ✓ |

`rule` is `"whole"` (the band the amount lands in applies to all of it) or `"marginal"` (each slice at its own
band's rate). A member a table does not have does not exist: `tc.rate(…)` is `invalid_method` (a fixed charge is
money, not a rate), as is `dc.apply(…)`.

```go
tr := fin.tiered_rates("0:0.005, 1000:0.007, 10000:0.009")
tr.charge(15000, "whole", 2, "half_even")       // 135.00 — all of it at the 10000 band's rate
tr.charge(15000, "marginal", 2, "half_even")    // 113.00 — 5.00 + 63.00 + 45.00
tc := fin.tiered_charges("0:0.015+2.00, 1000:0.01; min=25, max=500")
tc.charge(500, "whole", 2, "half_even")         // 25.00 — the minimum
dr := fin.dated_rates("2026-01-01:0.04, 2026-07-01:0.02")
dr.accrue(10000, d"2026-01-01", d"2027-01-01", "ACT/365F", 2, "half_even")   // 299.18 — 198.36 + 100.82
dr.at(d"2025-12-31")                            // raises: (at) no band covers the argument
dr.at(d"2025-12-31", 0.05d)                     // 0.05 — the fallback, before the first band only
```

**Values, not containers.** A table has no `len()`, iteration, indexing or `in`; `for b in t.bands()` reads it.
`bands()` answers the records in constructor shape, so `fin.tiered_rates(t.bands()) == t`. Two tables are equal
when they have the same bands, compared numerically (`"0:0.50"` equals `"0:0.5"`; for `tiered_charges` the bounds
too) — but `"0:0, 1000:0"` is **not** `"0:0"`: under `"marginal"` an extra band changes the parts. There is no
ordering. Each type's default (`fin.tiered_rates()` …) is the zero-rate table (`"0:0"`, or `"0001-01-01:0"` for the
dated ones), the one falsy table.

**In configuration**, store the text form: `json.encode` writes a table as its text, and `json.decode` gives the
string back for `fin.tiered_rates(cfg.rates)`. (JSON numbers decode as `float`, which the record form refuses.)

Not yet available: conventions outside the named set (a custom year length, a host-defined name table, a
script-defined convention) — a stated fraction, `fin.year_fraction(n, d)`, covers the functions that take `f`.

**Failures.** Data the formula cannot use raises `invalid_value` with fin128's reason —
`(fin.payment) the number of periods must be positive`, `(fin.irr) the bracket does not contain a root`,
`(fin.straight_line) the salvage value is above the cost`. A wrong argument type raises `invalid_argument_type`,
a wrong count `wrong_num_arguments`, a division by zero `division_by_zero`.

## fs

The file system: files, directories, links, permissions. World access: a host must grant it.

```go
fs := import("fs")
f := fs.create("./out.txt")
f.write_string("hello")
f.close()
string(fs.read_file("./out.txt"))   // "hello" — read_file answers bytes
fs.stat("./out.txt").size           // 5
```

Constants:

- Open flags: `o_rd`, `o_wr`, `o_rdwr`, `o_append`, `o_create`, `o_excl`, `o_sync`, `o_trunc`.
- Seek modes: `seek_set`, `seek_cur`, `seek_end`.
- File mode bits: `mode_dir`, `mode_append`, `mode_exclusive`, `mode_temporary`, `mode_symlink`, `mode_device`,
  `mode_named_pipe`, `mode_socket`, `mode_set_uid`, `mode_set_gui`, `mode_char_device`, `mode_sticky`,
  `mode_type`, `mode_perm`.

- `fs.open(path string) -> fs.file`: Open a file for reading.
- `fs.open_file(path string, flag int, perm int) -> fs.file`: Open with flags and permission bits.
- `fs.create(path string) -> fs.file`: Create (or truncate) a file.
- `fs.read_file(path string) -> bytes`: Read a whole file.
- `fs.stat(path string) -> record`: A frozen record `{name, mtime, size, mode, directory}`.
- `fs.mkdir(path string, perm int) -> undefined`, `fs.mkdir_all(path string, perm int) -> undefined`: Create a
  directory, or a whole tree.
- `fs.remove(path string) -> undefined`, `fs.remove_all(path string) -> undefined`: Remove a file or empty
  directory, or a tree.
- `fs.rename(old_path string, new_path string) -> undefined`: Rename or move.
- `fs.truncate(path string, size int) -> undefined`: Truncate a file.
- `fs.chmod(path string, mode int) -> undefined`, `fs.chown(path string, uid int, gid int) -> undefined`,
  `fs.lchown(path string, uid int, gid int) -> undefined`: Permission bits and ownership.
- `fs.link(old_path string, new_path string) -> undefined`, `fs.symlink(old_path string, new_path string) ->
  undefined`, `fs.read_link(path string) -> string`: Hard and symbolic links.
- `fs.temp_dir() -> string`: The system temporary directory.

An open file is an [`fs.file`](types/fs-file.md): `name`, `read(buffer)`, `write(bytes)`, `write_string(s)`,
`seek(offset, whence)`, `stat`, `sync`, `chmod`, `chown`, `chdir`, `read_dir_names(n)`, `close`.

**Failures.** Everything the world can refuse raises kind `io`, with the operation named in the message:
`(fs.remove) remove /nope: no such file or directory`. A function whose success has no result answers `undefined`.

**Migration:** these were `os` functions and constants of the same names (`os.read_file` → `fs.read_file`). The
file is a type instead of a record of closures.

## hex

Example:

```go
hex = import("hex")
hex.encode(bytes("ok"))
```

- `hex.encode(data bytes) -> string`: Hex-encode bytes.
- `hex.decode(s string) -> bytes`: Hex-decode string.

**Failures.** A malformed input raises kind `conversion`: `(hex.decode) encoding/hex: invalid byte …`.

## io

The script's console. It writes to the VM's stdout and reads from its stdin — the process streams unless the host
redirects them (`VM.SetStdout`, `VM.SetStdin`, see [embedding.md § Standard streams](embedding.md#standard-streams)).
`io` touches the world, so a host must grant it; the `kavun` CLI does.

| | line-oriented | raw stream |
| --- | --- | --- |
| output | `print(values...)` | `write(values...)` |
| input | `read_line([default])` | `read([n])` |

Example:

```go
io := import("io")
io.print("sum:", 20 + 22)            // sum: 42
io.write("a", "b"); io.write("c\n")  // abc
name := io.read_line("anonymous")    // one line of input, or "anonymous" at end of input
head := io.read(4)                   // the next 4 symbols ("" at end of input)
rest := io.read()                    // everything after them ("" when nothing is left)
```

- `io.print(values...) -> undefined`: Print values separated by spaces, then a newline. Text prints as its content,
  everything else as its render.
- `io.write(values...) -> undefined`: Write values concatenated — no separator, no newline. The building block for
  output assembled piece by piece.
- `io.read_line([default]) -> string`: One line of input without its terminator (`\n` or `\r\n`); a last line
  with no terminator is still a line. At end of input it raises `io` (`(io.read_line) end of input`), unless a
  default is given — then the default is answered as-is. (`""` is a valid line, so end of input cannot answer it.)
- `io.read([n]) -> string`: Without `n`, the rest of the input. With `n`, the next `n` symbols — the unit of a
  string's `len()` and indexing — waiting for them unless the input ends first, so it answers fewer only at end of
  input. `""` only at end of input (and for `n == 0`), so a loop `for s := io.read(64); s != ""; s = io.read(64)`
  ends cleanly. A negative `n` raises `invalid_value`; `n` bounds the read, it is not an allocation.

Both readers share one input buffer, so `read_line` followed by `read` loses nothing. Octets that are not valid
UTF-8 come through unchanged; each counts as one symbol, as in every text conversion.

Failures: `io` (an output or input stream the world refused, or end of input for `read_line` without a default);
`invalid_value` (a negative count); `invalid_argument_type`; `wrong_num_arguments`.

**Migration:** the module was `fmt`. `import("fmt")` → `import("io")`. `println(...)` → `print(...)`, and the old
`print` (no newline, no separators) → `write(...)`. Output goes to the VM's stdout instead of the process's.
`read_line` and `read` are new.

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

**Failures.** `decode` and `indent` raise kind `json_decoding` on malformed input. `decode` reads a number without
a fraction or exponent as `int` and any other as `float`, and raises `json_decoding` when it does not fit — an
integer outside int64 (`(json.decode) number 9223372036854775808 is out of int range`) or a float that would be
±Inf (`1e999`); it never wraps or clamps. `encode` raises kind
`json_encoding` on a value with no JSON representation, naming the path to it once — `.items[0].price: value type
<compiled-function/0> does not support JSON encoding` — and on text holding octets that are not symbols (JSON is
UTF-8 by definition; encode such text as `bytes`, which goes as base64).

## math

Constants, trigonometry and the special functions. A numeric operation that `decimal` has is a **float member**
instead — `x.sqrt()`, `x.pow(y)`, `x.exp()`, `x.ln()`, `x.log2()`, `x.log10()`, `x.round(n, mode)` and its twins,
`x.copy_sign(y)`, `x.quo_rem(y)`, … (see [types/float.md § Arithmetic members](types/float.md#arithmetic-members)) —
so no function exists both ways. `min`/`max` are not module functions either: selection over arguments is the free
variadic `min(a, b, ...)` / `max(a, b, ...)`, and aggregation over elements is the member (`arr.min()`).

Example:

```go
math = import("math")
math.hypot(3, 4)       // 5.0
math.cbrt(27)          // 3.0
math.pi                // 3.141592653589793
```

Constants:

- Core numeric constants: `e`, `pi`, `phi`, `sqrt2`, `sqrt_e`, `sqrt_pi`, `sqrt_phi`, `ln2`, `log2e`, `ln10`, `log10e`.
- Float bounds: `max_float32`, `smallest_nonzero_float32`, `max_float64`, `smallest_nonzero_float64`.
- Integer bounds: `max_int`, `min_int`, `max_int8`, `min_int8`, `max_int16`, `min_int16`, `max_int32`, `min_int32`, `max_int64`, `min_int64`.

- `math.acos(x float) -> float`: Arc cosine.
- `math.acosh(x float) -> float`: Inverse hyperbolic cosine.
- `math.asin(x float) -> float`: Arc sine.
- `math.asinh(x float) -> float`: Inverse hyperbolic sine.
- `math.atan(x float) -> float`: Arc tangent.
- `math.atan2(y float, x float) -> float`: Arc tangent of y/x with quadrant.
- `math.atanh(x float) -> float`: Inverse hyperbolic tangent.
- `math.cbrt(x float) -> float`: Cube root.
- `math.cos(x float) -> float`: Cosine.
- `math.cosh(x float) -> float`: Hyperbolic cosine.
- `math.dim(x float, y float) -> float`: Max(x-y, 0).
- `math.erf(x float) -> float`: Error function.
- `math.erfc(x float) -> float`: Complementary error function.
- `math.exp2(x float) -> float`: 2\*\*x.
- `math.expm1(x float) -> float`: e\*\*x - 1 with precision for small x.
- `math.gamma(x float) -> float`: Gamma function.
- `math.hypot(p float, q float) -> float`: sqrt(p*p + q*q).
- `math.ilogb(x float) -> int`: Binary exponent as integer.
- `math.inf(sign int) -> float`: +/- infinity by sign.
- `math.j0(x float) -> float`: Bessel J0.
- `math.j1(x float) -> float`: Bessel J1.
- `math.jn(n int, x float) -> float`: Bessel Jn.
- `math.ldexp(frac float, exp int) -> float`: frac \* 2\*\*exp.
- `math.log1p(x float) -> float`: log(1+x) with precision for small x.
- `math.logb(x float) -> float`: Binary exponent as float.
- `math.nan() -> float`: NaN value.
- `math.next_after(x float, y float) -> float`: Next representable float from x toward y.
- `math.pow10(n int) -> float`: 10\*\*n.
- `math.remainder(x float, y float) -> float`: IEEE 754 remainder.
- `math.sin(x float) -> float`: Sine.
- `math.sinh(x float) -> float`: Hyperbolic sine.
- `math.tan(x float) -> float`: Tangent.
- `math.tanh(x float) -> float`: Hyperbolic tangent.
- `math.y0(x float) -> float`: Bessel Y0.
- `math.y1(x float) -> float`: Bessel Y1.
- `math.yn(n int, x float) -> float`: Bessel Yn.

**Migration:** `abs`, `is_nan`, `is_inf`, `signbit`, `sqrt`, `pow`, `exp`, `log`, `log2`, `log10`, `floor`, `ceil`,
`trunc`, `copy_sign` and `mod` left the module; each is a float member now (`math.log(x)` → `x.ln()`,
`math.mod(x, y)` → `x.quo_rem(y)[1]`, `math.floor(x)` → `x.round_floor(0)`; full list in
[types/float.md § Migration notes](types/float.md#migration-notes)). Reading one raises
`module 'math' has no member 'sqrt'`.

## os

The process and its environment: arguments, exit, environment variables, identity, the working directory. Files are
[`fs`](#fs)'s, other processes [`exec`](#exec)'s, path text [`path`](#path)'s. World access: a host must grant it.

```go
os := import("os")
os.get_env("HOME")
os.platform              // "linux", "darwin", …
```

Constants: `platform`, `arch`, `dev_null`, `path_separator`, `path_list_separator`.

- `os.args() -> [string]`: Command-line arguments.
- `os.exit(code int) -> undefined`: Exit process with code.
- `os.environ() -> [string]`: Environment as `KEY=VALUE` strings.
- `os.get_env(key string) -> string`: Environment value (empty if missing).
- `os.set_env(key string, value string) -> undefined`: Set environment variable.
- `os.unset_env(key string) -> undefined`: Unset environment variable.
- `os.lookup_env(key string) -> string | false`: Lookup env var with presence flag.
- `os.expand_env(s string) -> string`: Expand `$VAR` references.
- `os.clear_env() -> undefined`: Clear all environment variables.
- `os.hostname() -> string`: Hostname.
- `os.get_wd() -> string`: Current working directory.
- `os.chdir(dir string) -> undefined`: Change current working directory.
- `os.get_pid() -> int`, `os.get_ppid() -> int`: Current and parent process ID.
- `os.get_uid() -> int`, `os.get_euid() -> int`, `os.get_gid() -> int`, `os.get_egid() -> int`: Real and effective
  user and group IDs.
- `os.get_groups() -> [int]`: Supplementary group IDs.
- `os.get_page_size() -> int`: Memory page size.
- `os.rand_generator() -> rand.generator`: A [`rand.generator`](types/rand-generator.md) over `crypto/rand` — the
  one entropy source (`os.rand_generator().int(1, 7)`).

**Failures.** Everything the world can refuse raises kind `io`, with the operation named in the message:
`(os.chdir) chdir /nope: no such file or directory`. A function whose success has no result answers `undefined`.

**Migration:** `os` was also the file system and process module. Each function and constant kept its name and
moved: files, directories, links, permissions and `temp_dir` to [`fs`](#fs) (`os.read_file` → `fs.read_file`,
`os.o_create` → `fs.o_create`), `exec`, `exec_look_path`, `start_process` and `find_process` to [`exec`](#exec).
The objects they answer are types now (`fs.file`, `exec.command`, `exec.process`), not records of closures.

## path

Path TEXT with the host's separator — joining, splitting, cleaning. It never touches the file system (that is
[`fs`](#fs)), so it is pure and importable by default.

```go
path := import("path")
path.join("a", "b", "../c")     // "a/c"
path.split("/x/y.txt")          // ["/x/", "y.txt"]
```

- `path.join(parts...) -> string`: Join with the separator and clean; `""` for no parts.
- `path.base(p) -> string`: The last element (`"."` for `""`).
- `path.dir(p) -> string`: Everything but the last element, cleaned.
- `path.ext(p) -> string`: The extension of the last element, dot included (`""` when none).
- `path.clean(p) -> string`: The shortest equivalent path.
- `path.split(p) -> [dir, file]`: `dir` keeps its trailing separator, so `dir + file == p`.
- `path.rel(base, target) -> string`: `target` relative to `base`; raises `invalid_value` when there is none.
- `path.is_abs(p) -> bool`: Whether `p` is absolute.
- `path.match(pattern, name) -> bool`: Shell pattern match (`*`, `?`, `[...]`); a malformed pattern raises
  `invalid_value`.

Failures: `invalid_value` (as above), `invalid_argument_type` (a non-string), `wrong_num_arguments`.

## rand

Pseudo-random numbers as a value. The module's one function is the constructor of the type it owns:
`rand.generator(seed)` answers a [`rand.generator`](types/rand-generator.md) whose draws are its own state. Pure —
it reads nothing from the world, so it is importable by default; entropy comes from `os.rand_generator()`, which
answers the same type over `crypto/rand`.

```go
rand := import("rand")
g := rand.generator(1)
g.int(1, 7)                 // a die roll, [1, 7) — the same on every run with seed 1
g.float()                   // [0, 1)
[1, 2, 3, 4, 5].shuffle(g)  // a new array, shuffled with g's draws
os := import("os")          // os needs a grant
d := os.rand_generator()    // the same type, unpredictable
```

- `rand.generator(seed int) -> rand.generator`: A seeded generator. The same seed gives the same sequence on every
  build: the source is PCG-DXSM (a specified algorithm) and every draw is derived from its raw output by Kavun's own
  code. A reproducible run seeds from a literal; a seed from entropy is `rand.generator(os.rand_generator().int())`.

The generator's members (see [types/rand-generator.md](types/rand-generator.md)): `int([lo,] hi)`,
`float([lo,] hi)`, `bytes(n)`, `perm(n)`, `norm_float()`, `exp_float()`, `copy()`. Ranges are half-open, like
`range(lo, hi)`.

**Failures.** A bound that leaves no value (`g.int(5, 5)`) or a bad count (`g.bytes(0)`) raises `invalid_value`.

**Migration:** the module-level draws and the process-global generator are gone — they shared state between every
script in the process. `rand.int_n(n)` → `g.int(n)`, `rand.int()` → `g.int()`, `rand.float()` → `g.float()`,
`rand.perm(n)` → `g.perm(n)`, `rand.read(buf)` → `g.bytes(n)` (answers new bytes), `rand.seed(s)` / `rand.rand(s)` →
`rand.generator(s)`; for unpredictable draws use `os.rand_generator()`. `rand` no longer needs a grant.

## re

Regular expressions. The module's one function is the constructor of the type it owns: `re.pattern(src)` compiles
the expression once, and the [`re.pattern`](types/re-pattern.md) answers `match`, `find`, `replace` and `split`.
Pure: importable by default.

```go
re := import("re")
digits := re.pattern("[0-9]+")
digits.match("abc123")              // true
digits.find("a1b22")                // [[{begin: 1, end: 2, text: "1"}]] — the first match, group 0 first
digits.find("a1b22", -1).len()      // 2 — every match
digits.find("abc")                  // [] — no match is an empty array, never undefined
digits.replace("a1b22", "#")        // "a#b#"
re.pattern(",\\s*").split("a, b,c")  // ["a", "b", "c"]
```

- `re.pattern(src string) -> re.pattern`: Compile; a malformed expression raises `invalid_value`.

The pattern's members (see [types/re-pattern.md](types/re-pattern.md)):

- `p.match(s) -> bool`: Whether the expression matches anywhere in `s`.
- `p.find(s, count?) -> [match]`: The first match (or up to `count`; `-1` for all), each an array of frozen
  `{text, begin, end}` group records, group 0 first; `[]` when nothing matches.
- `p.replace(s, repl) -> string`: Every match replaced; `$1` / `${name}` expand groups. A replacement that would
  grow past the sequence ceiling raises `invalid_value`.
- `p.split(s, count?) -> [string]`: `s` cut at the matches.
- `p.string() -> string`: The source text (the pattern IS text); `format(p)` renders `re.pattern("…")`.

**Failures.** An invalid expression raises kind `invalid_value` — a bad pattern is a bad argument:
`(re.pattern) error parsing regexp: missing closing ): `(``.

**Migration:** the module was `regexp`. `re_compile(p)` → `re.pattern(p)`; the one-shot functions became the
pattern's members: `re_match(p, s)` → `re.pattern(p).match(s)`, likewise `find`, `replace`, `split`. `find` answers
`[]` on no match (was `undefined`), and every match record is frozen. Hoist a pattern used in a loop: it is compiled
where `re.pattern(…)` runs.

Where the old `text` module's functions went (the regex ones are above):

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


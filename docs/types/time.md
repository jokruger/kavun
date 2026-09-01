# time

An instant in time, with nanosecond precision and a time zone.

## Overview

`time` is a domain scalar: an absolute instant plus the zone it is *viewed* in. The zone affects the
calendar accessors (`hour()`, `day()`, …) and the rendered text; it never affects identity — two values of
the same instant in different zones are equal, and all arithmetic and ordering work on the instant.

Key rules, each expanded below:

- Next to a `time`, a bare `int` **operand** is a duration in **nanoseconds**; a bare `int` in a
  **conversion** is a unix timestamp. Operator = duration, conversion = timestamp.
- `time` and `int` deliberately do **not** compare: an instant and a bare number have no common order.
- One render surface: `format(spec)` with the format mini-language. One canonical text form: RFC3339 with
  the fraction the instant carries (RFC3339Nano), used by `.string()`, f-strings, and the default
  `format()`.

`time` values are immutable.

## Construction

### The `times` module

```go
times := import("times")

times.now()                                     // the current instant
times.date(2026, 8, 29, 15, 4, 5, 123456789)    // time("2026-08-29T15:04:05.123456789Z") — UTC by default
times.date(2026, 8, 29, 15, 4, 5, 0, "Europe/Kyiv")  // time("2026-08-29T15:04:05+03:00") — optional IANA zone
times.parse("2006-01-02", "2026-08-29")         // time("2026-08-29T00:00:00Z") — Go reference layout
times.unix(1700000000, 500)                     // time("2023-11-14T22:13:20.0000005Z") — (sec, nsec)
times.from_unix(1700000000)                     // time("2023-11-14T22:13:20Z")
times.from_unix_ms(1700000000123)               // time("2023-11-14T22:13:20.123Z")
times.from_unix_micro(1700000000123456)         // …
times.from_unix_nano(1700000000123456789)       // …
```

The `from_unix*` family and the numeric conversions below always answer in **UTC** — an integer timestamp
is host-independent, so its wall-clock reading must be too.

`times.date` normalizes out-of-range components the calendar way rather than raising:

```go
times.date(2026, 1, 32, 0, 0, 0, 0)   // time("2026-02-01T00:00:00Z")
times.date(2026, 13, 1, 0, 0, 0, 0)   // time("2027-01-01T00:00:00Z")
```

Note: on bad input the module constructors return an `error` **value** (e.g. an unknown zone name in
`times.date`, an unparsable string in `times.parse`) — test the result or let the downstream use raise.
The conversion members below raise directly (or answer their `[default]`).

The module also carries duration constants for readable operator arithmetic —
`times.nanosecond` / `microsecond` / `millisecond` / `second` / `minute` / `hour` — plus calendar helpers
(`times.add_date`, `times.in_location`, `times.since`, `times.until`, …); see [stdlib](../stdlib.md).

### Conversions into `time`

An `int` in conversion position is a **unix timestamp**; the member name picks the encoding:

```go
(0).time()                          // time("1970-01-01T00:00:00Z") — seconds
(1700000000).time()                 // time("2023-11-14T22:13:20Z")
(1700000000123).time_ms()           // time("2023-11-14T22:13:20.123Z") — milliseconds
(1700000000123456).time_micro()     // microseconds
(1700000000123456789).time_nano()   // nanoseconds
```

`float` and `decimal` read as fractional unix seconds:

```go
(1700000000.5).time()               // time("2023-11-14T22:13:20.5Z")
decimal("1700000000.25").time()     // time("2023-11-14T22:13:20.25Z")
```

`string`/`runes` **parse**, accepting the common textual forms; a zoned text keeps its stated offset, a
zoneless one reads as UTC, and a bare digit string reads as a unix timestamp (normalized to UTC):

```go
"2026-08-29T12:00:00Z".time()       // time("2026-08-29T12:00:00Z")
"2026-08-29".time()                 // time("2026-08-29T00:00:00Z")
"2026-08-29 15:04:05".time()        // time("2026-08-29T15:04:05Z")
"1700000000".time()                 // time("2023-11-14T22:13:20Z")
"not a time".time()                 // raises: cannot convert string to time
"not a time".time(time())           // time("0001-01-01T00:00:00Z") — the [default] slot
```

A `dict` converts through the **components** reading — the same shape `components()` answers; every key is
optional, an unknown key raises:

```go
dict({year: 2026, month: 8, day: 29}).time()   // time("2026-08-29T00:00:00Z")
dict({yr: 2026}).time()                        // raises: (time) unknown component "yr"
dict({}).time()                                // time("0001-01-01T00:00:00Z") — all defaults
```

The free constructor takes the same conversions, plus the zero form; maybe-missing data is the member's
default:

```go
time()                     // time("0001-01-01T00:00:00Z") — the zero time
time(1700000000)           // ≡ (1700000000).time()
time("2026-08-29")         // ≡ "2026-08-29".time()
undefined.time(time())     // time("0001-01-01T00:00:00Z") — the maybe-missing form
```

## Operators

| expression | result | meaning |
| --- | --- | --- |
| `t + n` / `n + t` | `time` | `n` is a duration in **nanoseconds** |
| `t - n` | `time` | back by `n` nanoseconds |
| `t2 - t1` | `int` | the elapsed nanoseconds |
| `t1 < t2` (`<=` `>` `>=`) | `bool` | ordering of instants |
| `t == x` / `t != x` | `bool` | instant equality; `false`/`true` against any non-time |
| `n - t` | raises | an int minus an instant has no meaning |
| `t < n` / `n < t` | raises | deliberate — see below |
| `t + t2`, `t * n`, … | raises | no instant+instant, no scaling |

```go
t := times.date(2026, 8, 29, 15, 4, 5, 123456789)
t2 := times.date(2026, 8, 29, 16, 4, 5, 123456789)

t + 1000000000            // time("2026-08-29T15:04:06.123456789Z") — one second later
t - 1000000000            // time("2026-08-29T15:04:04.123456789Z")
t + 30 * times.second     // readable duration spelling
t2 - t                    // 3600000000000 — one hour, in nanoseconds
t < t2                    // true
t == t                    // true
```

### Reading a duration

A duration is nothing special — a plain `int` of nanoseconds. Divide by the module constants, or use the
`times.duration_*` helpers:

```go
elapsed := t2 - t                  // 3600000000000
elapsed / times.minute             // 60 — int division
times.duration_seconds(elapsed)    // 3600 — seconds, as a float
times.duration_minutes(elapsed)    // 60
times.duration_hours(elapsed)      // 1
times.duration_string(elapsed)     // "1h0m0s"
```

### The missing comparisons

**`time` vs `int` comparison is deliberately absent** — an instant and a bare number have no common order
(nanoseconds since when? seconds or millis?), so both directions raise instead of guessing:

```go
t < 1700000000            // raises: time < int
1700000000 < t            // raises: int < time
t == 1788015845           // false — equality just answers "not the same value"
```

Equality and ordering are zone-insensitive — they compare instants:

```go
kyiv := times.date(2026, 8, 29, 15, 4, 5, 0, "Europe/Kyiv")
kyiv == kyiv.utc()        // true — same instant, different view
```

## Members

### Calendar accessors

All zero-argument, answered in the value's own zone:

```go
t := times.date(2026, 8, 29, 15, 4, 5, 123456789)

t.year()             // 2026
t.month()            // 8 — January is 1
t.day()              // 29
t.hour()             // 15
t.minute()           // 4
t.second()           // 5
t.nanosecond()       // 123456789
t.month_name()       // "August"
t.week_day()         // 6 — Sunday is 0 … Saturday is 6
t.week_day_name()    // "Saturday"
t.year_day()         // 241 — 1-based day of the year
```

### Epoch accessors

```go
t.unix()             // 1788015845 — seconds since 1970-01-01T00:00:00Z
t.unix_ms()          // 1788015845123
t.unix_micro()       // 1788015845123456
t.unix_nano()        // 1788015845123456789
```

Every integral reading **floors** — the containing interval is the meaning, not a discarded remainder, so
half a second *before* the epoch is second `-1`, not `0`:

```go
tn := times.unix(0, 0) - 500000000   // 0.5s before the epoch
tn.unix()                            // -1
tn.unix_ms()                         // -500
```

### Zone accessors and moves

```go
kyiv := times.date(2026, 8, 29, 15, 4, 5, 0, "Europe/Kyiv")

kyiv.zone_name()      // "EEST"
kyiv.zone_offset()    // 10800 — seconds east of UTC
kyiv.utc()            // time("2026-08-29T12:04:05Z") — same instant, viewed in UTC
kyiv.utc().hour()     // 12
t.local()             // same instant in the host's local zone — host-dependent, the one impure member
```

`utc()`/`local()` move the *view*, never the instant. Zone survives arithmetic: `(kyiv + n)` keeps the
Kyiv view. For arbitrary zones use `times.in_location(t, "Europe/Kyiv")`.

### `components()` — the constitutive parts

Answers a record of exactly the parts the instant can be rebuilt from; the computed accessors
(`week_day`, `month_name`, `zone_name`) stay their own members:

```go
t.components()
// {"year": 2026, "month": 8, "day": 29, "hour": 15, "minute": 4,
//  "second": 5, "nanosecond": 123456789, "zone_offset": 0}   (key order not significant)

time(t.components()) == t            // true — the way back
dict(t.components()).time() == t     // true — the conversion spelling
```

### Render — `format([spec])`, the one surface

`format` with the [format mini-language](../format-mini-language.md) is the *only* member that renders a
`time` — there is no `iso()`/`date()`/`format_date()` family. `#`-specs name the layouts; anything after
`#` that is not a named layout is a strftime-style pattern:

```go
t.format()                  // "2026-08-29T15:04:05.123456789Z" — RFC3339, precision-preserving
t.format("#iso")            // "2026-08-29T15:04:05Z" — the explicitly seconds-truncating spec
t.format("#isonano")        // "2026-08-29T15:04:05.123456789Z"
t.format("#date")           // "2026-08-29"
t.format("#time")           // "15:04:05"
t.format("#datetime")       // "2026-08-29 15:04:05"
t.format("#unix")           // "1788015845"       (#unixms, #unixmicro, #unixnano likewise)
t.format("#rfc822")         // "29 Aug 26 15:04 UTC"
t.format("#%d.%m.%Y %H:%M") // "29.08.2026 15:04" — strftime directives
t.format("#%A, %B %e")      // "Saturday, August 29"
t.format("#%I:%M %p")       // "03:04 PM"
```

Supported strftime directives: `%Y %y %C %m %d %e %H %I %M %S %p %P %B %b %A %a %u %w %V %G %j %s %f
%Z %z %n %t %%` — an unknown directive raises.

### Conversions out

Every conversion is `x.T([default])`. An instant **as a number is its unix timestamp** — `int()` is a
deliberate synonym of `unix()` (the type name answers "as a number", the unit name "in which unit"):

```go
t.int()                    // 1788015845 — ≡ t.unix(), floors
t.float()                  // 1788015845.1234567 — fractional seconds, approximate (~100ns today)
t.decimal()                // 1788015845.123456789d — exact to the nanosecond
t.string()                 // "2026-08-29T15:04:05.123456789Z" — the ONE text form, RFC3339Nano
t.runes()                  // u"2026-08-29T15:04:05.123456789Z"
t.time()                   // identity
t.string().time() == t     // true — the text form round-trips
```

There is deliberately no `bool()` (truthiness is `!!t` / `t.is_true()`), no `byte()`/`rune()`/`bytes()`
(no octet meaning), and no container targets or `len()` — an instant has no elements.

### Truthiness

The **zero time** (`0001-01-01T00:00:00Z`) is falsy; every other instant — the unix epoch included — is
truthy. There is no `is_zero()`: `!t.is_true()` is exactly that test.

```go
time().is_true()                       // false — the zero time
times.date(1, 1, 1, 0, 0, 0, 0).is_true()  // false — the same instant
(0).time().is_true()                   // true — unix 0 is 1970, a real instant
!time()                                // true
```

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar — kept so generic code never type-errors:

```go
t.copy() == t     // true
t.freeze() == t   // true
```

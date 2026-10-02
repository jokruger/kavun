# time

An instant in time, with nanosecond precision and a time zone.

## Overview

`time` is a domain scalar: an absolute instant plus the zone it is *viewed* in. The zone affects the
calendar accessors (`hour()`, `day()`, …), calendar arithmetic and the rendered text; it never affects
identity — two values of the same instant in different zones are equal, and ordering and duration
arithmetic work on the instant.

Key rules, each expanded below:

- Next to a `time`, a bare `int` **operand** is a duration in **nanoseconds**; a bare `int` in a
  **conversion** is a unix timestamp. Operator = duration, conversion = timestamp.
- `time` and `int` deliberately do **not** compare: an instant and a bare number have no common order.
- **One strict text grammar** (below) for `t"…"`, `time(s)` and the canonical layouts — nothing is guessed.
  Any other text shape names its **layout**: `time(s, layout)`, `s.time(layout[, default])`.
- **A wall clock that does not name exactly one instant raises** — a time a daylight-saving gap skips, or one
  an overlap repeats — whether it is constructed or computed. Nothing normalizes.
- A zone is an IANA name (`"Europe/Kyiv"`) or `"UTC"`. The host's own zone is never visible: `"Local"` and
  `""` are not zone names, and `times.now()` answers in UTC.
- One render surface: `format(spec)` with the format mini-language. One canonical text form: RFC3339 with
  the fraction the instant carries (RFC3339Nano), used by `.string()`, f-strings, and the default
  `format()`.

`time` values are immutable. The range is years 1…9999.

## Construction

### Text: the canonical grammar

`t"…"` literals, `time(s)` and the layouts `iso`/`isonano`/`date`/`datetime` read exactly this grammar:

| form | reads as |
| --- | --- |
| `YYYY-MM-DD` | midnight UTC |
| `YYYY-MM-DD` + `T` **or a space** + `HH:MM[:SS[.f]]` (1–9 fraction digits) | that wall clock, UTC |
| any of the above + `Z` / `±HH:MM` / `±HHMM` | that wall clock at the stated offset (`Z` and `+00:00` are UTC) |

```go
t"2026-08-29T15:04:05Z"             // a literal: a compile-time constant
time("2026-08-29")                  // time("2026-08-29T00:00:00Z")
time("2026-08-29 15:04:05")         // time("2026-08-29T15:04:05Z") — the form #datetime writes
time("2026-08-29T15:04+03:00")      // time("2026-08-29T15:04:00+03:00")
time("9999-12-31T23:59:59.9999999") // the far sentinel many backends use reads fine
```

Every field is fixed width and every value must exist. These raise `conversion` (a `t"…"` literal that does not
match is a compile error): unpadded fields (`"2026-8-9"`), slashes (`"12/01/2026"`), month names and free text,
bare digits (`"20260829"`, `"1700000000"` — name the unit with a layout), lowercase `t`/`z`, and impossible
values (`"2026-02-30"`, `"2026-08-29T24:00"`).

```go
time("12/01/2026")                  // raises: cannot convert string to time: expected a 4-digit year at offset 0
time("2026-02-30")                  // raises: ...: day 30 out of range 1..28
```

### Text: layouts

Any other shape names its layout. **On a text receiver the layout is required**, and the conversion's
`[default]` comes after it — so a layout can never be mistaken for a default:

```go
time("29/08/2026", "%d/%m/%Y")              // free form: the second argument is the layout
"29/08/2026".time("%d/%m/%Y")               // member form: layout required
"bad".time("%d/%m/%Y", undefined)           // undefined — the default, after the layout
"2026-08-29".time("iso")                    // the canonical grammar, spelled explicitly
"2026-08-29".time()                         // raises: wrong_num_arguments — text needs a layout
time("29/08/2026 03:04 PM", "%d/%m/%Y %I:%M %p")   // time("2026-08-29T15:04:00Z")
time("1700000000", "unix")                  // time("2023-11-14T22:13:20Z") — the layout names the unit
```

| layout | reads |
| --- | --- |
| `iso`, `isonano`, `date`, `datetime` | the canonical grammar above |
| `unix`, `unixms`, `unixmicro`, `unixnano` | an integer, in exactly that unit |
| `time`, `rfc822` | refused — no date / a 2-digit year and a zone abbreviation |
| a `%`-template (anything containing `%`) | the directives below; every other byte matches literally |

Readable directives — the inverse of the render directives, over the ones that read back unambiguously:
`%Y %m %d %e %j %H %I %M %S %f %p %P %z %s` (fixed width, exactly as rendered), `%B %b %A %a` (English
names, exact case; a weekday that contradicts the date raises), `%n %t %%`. Refused on input: `%y` (a century
pivot is a guess), `%Z` (an abbreviation names several zones), and the week-based `%G %V %u %w %C`. Unset fields
default to the zero value's (`1970-01-01T00:00:00Z`). `%I` needs `%p`; `%s` stands alone.

A wrong **layout** is a mistake in the script, not a miss in the data, so it raises `invalid_value` even when a
default is given:

```go
"x".time("%y", undefined)           // raises: (time) directive %y cannot be parsed (ambiguous on input)
"x".time("%d/%m/%Y", undefined)     // undefined — the data missed, the layout is fine
```

### Components

A `dict` or `record` converts through the **components** reading — the same shape `components()` answers.
Every key is optional and defaults to the zero value's part; an unknown key raises; nothing normalizes:

```go
time({year: 2026, month: 8, day: 29})                       // time("2026-08-29T00:00:00Z")
time({year: 2026, month: 3, day: 29, hour: 15, zone: "Europe/Kyiv"})   // time("2026-03-29T15:00:00+03:00")
time({year: 2026, hour: 1, zone_offset: 3600})              // time("2026-01-01T01:00:00+01:00") — a fixed offset
time({})                                                    // time("1970-01-01T00:00:00Z") — the zero time
time({yr: 2026})                                            // raises: (time) unknown component "yr"
time({year: 2026, month: 1, day: 32})                       // raises: (time) day 32 out of range 1..31
```

`zone` makes the parts a wall clock in that zone, and the **one-instant rule** applies:

```go
time({year: 2026, month: 3, day: 29, hour: 3, minute: 30, zone: "Europe/Kyiv"})
// raises: 2026-03-29 03:30:00 does not exist in zone Europe/Kyiv (a daylight-saving gap)
time({year: 2026, month: 10, day: 25, hour: 3, minute: 30, zone: "Europe/Kyiv"})
// raises: ... occurs twice in zone Europe/Kyiv (a daylight-saving overlap): give zone_offset to pick one
time({year: 2026, month: 10, day: 25, hour: 3, minute: 30, zone: "Europe/Kyiv", zone_offset: 7200})
// time("2026-10-25T03:30:00+02:00") — the second 03:30
```

`zone` and `zone_offset` together must agree; `"Local"`, `""` and unknown names raise.

### Numbers

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

Every numeric conversion answers in **UTC** — an integer timestamp is host-independent, so its wall-clock
reading must be too.

### The free constructor and the `times` module

```go
time()                     // time("1970-01-01T00:00:00Z") — the zero time, unix 0
time(1700000000)           // ≡ (1700000000).time()
time("2026-08-29")         // the canonical grammar
time(s, layout)            // a layout (the free form has no default slot; the member does)
undefined.time(time())     // time("1970-01-01T00:00:00Z") — the maybe-missing form

times := import("times")
times.now()                                     // the current instant, in UTC
times.unix(1700000000, 500)                     // time("2023-11-14T22:13:20.0000005Z") — (sec, nsec)
times.from_unix_ms(1700000000123)               // time("2023-11-14T22:13:20.123Z")
times.from_unix_micro(1700000000123456)         // …
times.from_unix_nano(1700000000123456789)       // …
```

On bad input every constructor **raises** a catchable error — none answers an `error` value. The module also
carries duration constants for readable operator arithmetic (`times.nanosecond` / `microsecond` /
`millisecond` / `second` / `minute` / `hour`), `times.since`/`until`, and the calendar facts about plain
numbers (`times.is_leap_year(y)`, …); see [stdlib](../stdlib.md).

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
times := import("times")
t := t"2026-08-29T15:04:05.123456789Z"
t2 := t"2026-08-29T16:04:05.123456789Z"

t + 1000000000            // time("2026-08-29T15:04:06.123456789Z") — one second later
t - 1000000000            // time("2026-08-29T15:04:04.123456789Z")
t + 30 * times.second     // readable duration spelling
t2 - t                    // 3600000000000 — one hour, in nanoseconds
t < t2                    // true
t == t                    // true
```

A duration is exact elapsed time; to move by **calendar** days or months, use the arithmetic members below.

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
kyiv := t.in_zone("Europe/Kyiv")
kyiv == t                 // true — same instant, different view
```

## Members

### Calendar accessors

All zero-argument, answered in the value's own zone:

```go
t := t"2026-08-29T15:04:05.123456789Z"

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
t.is_leap_year()     // false
t.days_in_year()     // 365
t.days_in_month()    // 31
```

### Calendar arithmetic

`add_days`, `add_months` and `add_years` move the **calendar** day in the value's own zone and keep the wall
clock. The end-of-month rule is required — `"clamp"` or `"last_day"`, with no default — and only matters when
the start is a month end:

```go
t"2026-01-31T10:00:00Z".add_months(1, "clamp")      // time("2026-02-28T10:00:00Z")
t"2026-04-30T10:00:00Z".add_months(1, "clamp")      // time("2026-05-30T10:00:00Z")
t"2026-04-30T10:00:00Z".add_months(1, "last_day")   // time("2026-05-31T10:00:00Z") — month end stays month end
t"2023-02-28T00:00:00Z".add_years(1, "last_day")    // time("2024-02-29T00:00:00Z")
t"2026-12-31T23:59:59Z".add_days(1)                 // time("2027-01-01T23:59:59Z")
t"2026-01-31T10:00:00Z".add_months(1)               // raises: wrong_num_arguments — state the rule
```

The kept wall clock follows the one-instant rule: landing in a daylight-saving gap or overlap **raises**.
UTC and fixed-offset times never do, so moving the arithmetic there is the explicit escape:

```go
a := time({year: 2026, month: 3, day: 28, hour: 3, minute: 30, zone: "Europe/Kyiv"})
a.add_days(1)          // raises: 2026-03-29 03:30:00 does not exist in zone Europe/Kyiv (a daylight-saving gap)
a.add_days(2)          // time("2026-03-30T03:30:00+03:00")
a.utc().add_days(1)    // time("2026-03-29T01:30:00Z") — exactly 24h later, in UTC
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
kyiv := t"2026-08-29T12:04:05Z".in_zone("Europe/Kyiv")

kyiv                  // time("2026-08-29T15:04:05+03:00") — same instant, viewed in Kyiv
kyiv.zone_name()      // "EEST"
kyiv.zone_offset()    // 10800 — seconds east of UTC
kyiv.utc()            // time("2026-08-29T12:04:05Z") — same instant, viewed in UTC
kyiv.utc().hour()     // 12
kyiv.in_zone("")      // raises: (in_zone) a zone name is required
```

`in_zone(name)`/`utc()` move the *view*, never the instant. Zone survives arithmetic: `(kyiv + n)` and
`kyiv.add_days(n)` keep the Kyiv view.

### `components()` — the constitutive parts

Answers a record of exactly the parts the instant can be rebuilt from; the computed accessors
(`week_day`, `month_name`, `zone_name`) stay their own members. A value viewed in a named zone also carries
`zone`; UTC and fixed offsets are fully described by `zone_offset`:

```go
t.components()
// {"year": 2026, "month": 8, "day": 29, "hour": 15, "minute": 4,
//  "second": 5, "nanosecond": 123456789, "zone_offset": 0}   (key order not significant)
kyiv.components().zone               // "Europe/Kyiv"

time(t.components()) == t            // true — the way back
dict(kyiv.components()).time() == kyiv   // true — the conversion spelling, zone included
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
%Z %z %n %t %%` — an unknown directive raises. The same names read back as layouts (see
[Text: layouts](#text-layouts) for which do).

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
time(t.string()) == t      // true — the text form round-trips through the canonical grammar
```

There is deliberately no `bool()` (truthiness is `!!t` / `t.is_true()`), no `byte()`/`rune()`/`bytes()`
(no octet meaning), and no container targets or `len()` — an instant has no elements.

### Truthiness

Derived like every type's: a time is falsy exactly when it is the zero value `time()`, the unix epoch
`1970-01-01T00:00:00Z` (in any zone). Every other instant is truthy — including `0001-01-01T00:00:00Z`, the
"unset" sentinel some backends (.NET's `default(DateTime)`) send. A host that means "missing" by such a
sentinel should map it to `undefined` before it reaches a script.

```go
time().is_true()                       // false — the zero time
(0).time().is_true()                   // false — the same instant
t"0001-01-01T00:00:00Z".is_true()      // true — a real instant
!time()                                // true
```

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar — kept so generic code never type-errors:

```go
t.copy() == t     // true
t.freeze() == t   // true
```

## Zone data

Zone names resolve through the host's IANA tzdata (Go's lookup order: `ZONEINFO`, the system copy,
`$GOROOT/lib/time/zoneinfo.zip`, and the embedded copy the `kavun` CLI links in). Zone rules change, so a
named-zone result can differ between hosts with different tzdata versions; UTC and fixed offsets never do.
See [embedding](../embedding.md#time-zone-data) for pinning tzdata and replaying past results.

## Migration notes

- **Text parsing is strict.** `time(s)`, `t"…"` and the canonical layouts read only the grammar above. Slashes,
  month names, free text, unpadded fields, lowercase `t`/`z` and bare digit strings now raise (a `t"…"`
  literal, at compile time). A digit string was a unix timestamp whose unit was guessed from its length; name
  the unit: `time(s, "unix")`. The `github.com/araddon/dateparse` dependency is gone.
- **Text → time takes a layout in member form.** `s.time()` → `s.time("iso")` (or `time(s)`); `s.time(default)`
  → `s.time("iso", default)`. A non-string first argument raises, so a default can no longer be taken for a
  layout (or the reverse).
- **`times.parse(layout, s)` is removed** → `time(s, layout)` with `%` directives (`"2006-01-02"` →
  `"%Y-%m-%d"`). The `times.format_*` Go-layout constants went with it; the format aliases replace them.
- **`times.date(y, mo, d, h, mi, s, ns[, zone])` is removed** → `time({year, month, day, hour, minute, second,
  nanosecond, zone})`. Components no longer normalize (`day: 32` raises), and a wall clock in a DST gap or
  overlap raises.
- **`times.add_date(t, y, m, d)` is removed** → `t.add_years(y, eom)`, `t.add_months(m, eom)`, `t.add_days(d)`.
  Go's normalization (31 Jan + 1 month = 3 March) is gone: the end-of-month rule is stated.
- **`times.in_location(t, z)` is removed** → `t.in_zone(z)`. **`t.local()` is removed**, and `"Local"`/`""` are
  not zone names: name the zone. **`times.now()` answers in UTC**, not the host's zone.
- **`times.from_unix(n)` is removed** → `(n).time()`.
- **The zero value is unix 0.** `time()` was `0001-01-01T00:00:00Z`; it is now `1970-01-01T00:00:00Z`, the same
  instant `(0).time()` and `date()` name. Truthiness moves with it: the epoch is falsy, `0001-01-01` truthy.

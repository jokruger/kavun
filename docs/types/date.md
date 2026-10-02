# date

A civil calendar day — `0001-01-01` … `9999-12-31` — with no clock and no zone.

## Overview

`date` is a domain scalar for the questions that are about *days*, not instants: a due date, a value date, a
birthday, the day a posting is booked. It has no time of day and no zone, so two dates compare and subtract
exactly, and nothing about them depends on where the script runs. When a day has to meet the clock, the
conversion names the zone (see [the time bridge](#the-time-bridge)).

Key rules, each expanded below:

- Next to a `date`, a bare `int` **operand** is a number of **days**; a bare `int` in a **conversion** is a day
  count since `1970-01-01` (epoch days). Operator = days, conversion = epoch days.
- One strict text form, `YYYY-MM-DD`, for `d"…"`, `date(s)`, `.string()`, f-strings and JSON. Any other text
  shape names its layout: `date(s, layout)`, `s.date(layout[, default])`.
- Month and year arithmetic states its end-of-month rule — `"clamp"` or `"last_day"` — with no default.
- `date` and `time` / `int` deliberately do **not** order against each other; `==` across them is `false`.

`date` values are immutable. A date is a primitive: it never allocates, and a `d"…"` literal is a compile-time
constant.

## Construction

```go
d"2026-01-31"                       // literal: a compile-time constant; a bad literal is a compile error
date("2026-01-31")                  // the canonical text
date("31/01/2026", "%d/%m/%Y")      // a layout — the free form's second argument
"31/01/2026".date("%d/%m/%Y")       // member form on text: the layout is required
"bad".date("date", undefined)       // undefined — the default comes after the layout
date({year: 2026, month: 1, day: 31})   // components (every key optional, defaulting to date()'s part)
(20484).date()                      // epoch days: d"2026-01-31"
date(t"2026-03-31T23:30:00Z")       // a time's civil day, in the time's own zone
date()                              // d"1970-01-01" — the zero value
undefined.date(d"2026-01-01")       // d"2026-01-01" — the maybe-missing form
```

The text form is exactly `YYYY-MM-DD`: fixed width, a day that exists, nothing after it. Everything else raises
`conversion`; nothing normalizes:

```go
date("2026-1-31")                   // raises: cannot convert string to date: expected a 2-digit month at offset 5
date("2026-02-30")                  // raises: ...: day 30 out of range 1..28
date("2026-01-31T00:00:00Z")        // raises: ...: unexpected text (a date has no clock or zone) at offset 10
date({year: 2026, month: 2, day: 29})   // raises: (date) day 29 out of range 1..28
date(3652059)                       // raises: cannot convert int to date — one day past 9999-12-31
```

**Layouts** for text are the `date` alias (the canonical form) or a `%`-template of **date directives**:
`%Y %m %d %e %j %B %b %A %a %n %t %%` (fixed width; English names, exact case; a weekday that contradicts the
date raises). Clock and zone directives (`%H %M %S %z` …) raise on a date, and so do the ambiguous `%y %Z` and
the week-based `%G %V %u %w %C`. A wrong layout raises `invalid_value` even when a default is given — it is a
mistake in the script, not a miss in the data:

```go
date("Saturday 31 January 2026", "%A %d %B %Y")   // d"2026-01-31"
date("2026-031", "%Y-%j")                         // d"2026-01-31" — day of year
"x".date("%H", undefined)                         // raises: directive %H is not available on a date
"x".date("%d/%m/%Y", undefined)                   // undefined — the data missed, the layout is fine
```

## Operators

| expression | result | meaning |
| --- | --- | --- |
| `d + n` / `n + d` | `date` | `n` days later |
| `d - n` | `date` | `n` days earlier |
| `d2 - d1` | `int` | the days between (negative when `d2` is earlier) |
| `d1 < d2` (`<=` `>` `>=`) | `bool` | calendar order |
| `d == x` / `d != x` | `bool` | the same day; `false`/`true` against any non-date |
| a result outside `0001-01-01` … `9999-12-31` | raises | `invalid_value` |
| `n - d`, `d + d`, `d * n`, `d / n`, `d ± float` | raises | no meaning for a day |
| `d < n`, `d < t` (and the reverse) | raises | a day has no common order with a number or an instant |
| `d + t` | raises | a day is not a duration |

```go
d := d"2026-01-31"
d + 1                       // d"2026-02-01"
d - 31                      // d"2025-12-31"
d"2026-03-01" - d"2026-02-01"   // 28
d < d"2026-02-01"           // true
d == 20484                  // false — use d.int() == 20484
d < 5                       // raises: date < int
min(d, d"2026-02-01")       // d"2026-01-31" — the global min/max work, the type has no min member
```

## Members

### Calendar accessors

```go
d := d"2026-01-31"
d.year()             // 2026
d.month()            // 1 — January is 1
d.day()              // 31
d.week_day()         // 6 — Sunday is 0 … Saturday is 6 (as time)
d.week_day_name()    // "Saturday"
d.month_name()       // "January"
d.year_day()         // 31 — 1-based day of the year
```

### Month shape and calendar facts

```go
f := d"2028-02-10"
f.start_of_month()   // d"2028-02-01"
f.end_of_month()     // d"2028-02-29"
f.is_end_of_month()  // false
f.is_leap_year()     // true
f.days_in_year()     // 366
f.days_in_month()    // 29
```

For plain numbers — "how long is February 2027?" — use `times.days_in_month(2027, 2)`; see [stdlib](../stdlib.md#times).

### Arithmetic

`add_days(n)` is `d + n`. Month and year steps take a **required** end-of-month rule, which only matters when
the start is a month end:

| rule | meaning |
| --- | --- |
| `"clamp"` | keep the day of the month, clamped to the target month's length |
| `"last_day"` | a month-end start stays a month end |

```go
d"2026-01-31".add_months(1, "clamp")      // d"2026-02-28"
d"2026-04-30".add_months(1, "clamp")      // d"2026-05-30"
d"2026-04-30".add_months(1, "last_day")   // d"2026-05-31"
d"2024-02-29".add_years(1, "clamp")       // d"2025-02-28"
d"2023-02-28".add_years(1, "last_day")    // d"2024-02-29"
d"2026-01-31".add_months(1)               // raises: wrong_num_arguments — state the rule
```

`months_since(other, eom)` is their inverse: `[months, days]` — the whole months from `other` to the receiver
and the days left over, so that `other.add_months(months, eom) + days` is the receiver. Both are negative when
the receiver is earlier:

```go
d"2026-05-31".months_since(d"2026-01-31", "clamp")     // [4, 0]
d"2026-05-30".months_since(d"2026-04-30", "last_day")  // [0, 30] — the month step would land on 31 May
d"2026-01-01".months_since(d"2026-03-15", "clamp")     // [-2, -14]
m, days := d"2026-07-20".months_since(d"2026-01-31", "clamp")   // destructures positionally
```

### The time bridge

A `date` has no zone, so every move to or from a `time` names one:

```go
t := t"2026-03-31T23:30:00Z"
t.date()                       // d"2026-03-31" — the civil day in the time's own zone
t.date_in("Europe/Kyiv")       // d"2026-04-01" — the same instant is already April 1st in Kyiv
t.in_zone("Europe/Kyiv").date()  // d"2026-04-01" — the same thing

d := d"2026-01-31"
d.time()                       // time("2026-01-31T00:00:00Z") — midnight UTC; ≡ time(d)
d.time_in("Europe/Kyiv")       // time("2026-01-31T00:00:00+02:00") — the start of the day in Kyiv
```

**Which day an instant falls on depends on the zone** — the first line above is a real hazard for bookings:
near midnight, UTC and local days differ. Name the zone the business day belongs to.

`time_in(zone)` answers the **first instant whose local date is the day**: midnight, or — on a day whose
midnight a daylight-saving change skips — the instant the gap ends. So `d.time_in(z).date_in(z) == d` always
holds:

```go
d"2026-09-06".time_in("America/Santiago")   // time("2026-09-06T01:00:00-03:00") — Chile skips 00:00–01:00
```

`"Local"` and `""` are not zone names; an unknown zone raises.

### Render — `format([spec])`

The default render and `#date`/`#iso` are `YYYY-MM-DD`; a `%`-template takes the date directives
(`%Y %y %C %m %d %e %j %B %b %A %a %u %w %V %G %n %t %%`). Clock and zone directives, and the clock-bearing
aliases (`#datetime`, `#time`, `#unix`, …), raise:

```go
d := d"2026-01-31"
f"{d}"                      // "2026-01-31"
f"{d:#%d/%m/%Y}"            // "31/01/2026"
f"{d:#%A %e %B %Y}"         // "Saturday 31 January 2026"
f"{d:v}"                    // `date("2026-01-31")` — the source form
d.format("#%H")             // raises: directive %H is not available on a date
```

### Conversions out

Every conversion is `x.T([default])`:

```go
d.string()                  // "2026-01-31"
d.runes()                   // u"2026-01-31"
d.int()                     // 20484 — epoch days; (20484).date() is the way back
d.time()                    // midnight UTC
d.components()              // {"year": 2026, "month": 1, "day": 31}
date(d.components()) == d   // true
```

`json.encode(d)` is `"2026-01-31"`; `json.decode` never produces a date (JSON has no date type), so a config
reads it back with `date(cfg.due)`. The binary codec is 4 bytes and re-validates the range on decode.

### Truthiness

Derived like every type's: `date()` (`1970-01-01`) is falsy, every other day is truthy — including `0001-01-01`
and `9999-12-31`, the sentinels other systems use for "unset" and "no end". A host that means "missing" by one
should map it to `undefined`.

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar, kept so generic code never type-errors.

## For hosts

`kavun.ValueOf` accepts fin128's `civil.Date` directly, and a date's `Interface()` is a `civil.Date`.

## Migration notes

- **New type.** `date` and the `d"…"` literal are new. Before it, a day was a `time` at midnight UTC; such
  scripts keep working, and `t.date()` / `d.time()` convert between the two.
- New builtins: `date(...)`, `is_date(x)`. New members on other types: `t.date([default])`, `t.date_in(zone)`,
  `(n).date([default])` (epoch days), `s.date(layout[, default])` on `string`/`runes`, `dict.date([default])`
  (components), and `undefined.date(default)`.

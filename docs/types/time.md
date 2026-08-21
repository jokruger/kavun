# time

Date and time values (timestamp or calendar values).

## Overview

The `time` type represents an instant in time. Time values are typically created using the `time()` function with
various input formats. They store a precise moment and provide methods for querying and formatting.

## Declaration and Usage

### Construction via Function

```go
t = time("2024-01-01")                   // ISO 8601 date
t2 = time("2024-01-01T12:30:00Z")        // ISO 8601 datetime in UTC
t3 = time("2024-01-01T12:30:00+05:30")   // ISO 8601 with timezone
t4 = time(1704067200)                    // Unix timestamp, seconds (int)
t5 = time(1704067200.5)                  // Unix timestamp, sec.frac (float -- lossy)
t6 = time(1704067200.123456789d)         // Unix timestamp, sec.frac (decimal -- exact)
t7 = t"2024-01-01T12:30:00Z"            // static time literal
```

`t"..."` uses the same parsing logic as `time("...")` for string inputs and is resolved at compile time using
`dateparse.ParseAny`.

### Input Formats

Time constructor automatically detects various formats and parses them accordingly.

| input | reading |
|---|---|
| `string` / `runes` | text: ISO 8601 and many other layouts; a bare numeric string is a unix timestamp whose unit is inferred from its digit count |
| `int` | unix timestamp in **seconds** (`n.time_ms()`/`time_micro()`/`time_nano()` for the other encodings) |
| `float` | unix timestamp as **sec.frac** — integer part seconds, fraction sub-second. **Lossy**: float64 has ~15–16 significant digits and a present-day timestamp spends 10 on the seconds, so `time(1704067200.123)` lands on `…00.122999907`. Use `decimal` when the sub-second part must be right |
| `decimal` | unix timestamp as **sec.frac**, exact — `time(1704067200.123456789d)` keeps every digit, since dec128 is base 10. Finer than nanoseconds truncates |
| anything else | declines: `time(x)` is `undefined`, `time(x, fallback)` is `fallback` |

`NaN`, `Inf`, and values outside `int64` seconds also decline the same way.

**Every construction path produces UTC**, so wall-clock accessors (`hour()`, `day()`, `year()`, …) never
depend on the host machine's timezone — the same script gives the same answer everywhere. The one thing that
*is* preserved is an explicit zone written in the input: `time("2024-01-01T12:30:00+05:30").hour()` is `12`,
because that offset is data the caller supplied, not a default to normalize away. `times.date(...)` without
its optional location argument builds in UTC for the same reason, and `times.now()` is the deliberate
exception (it is host-local and marked impure).

## Arithmetic and Comparison

`time` combines with `int` and with itself only — there is no arithmetic or ordering with any other type.

```go
t = time("2024-01-01T00:00:00Z")

t + 1000000000     // 2024-01-01 00:00:01 +0000 UTC -- add 1 second (int is nanoseconds, not seconds)
t - 1000000000      // 2023-12-31 23:59:59 +0000 UTC -- subtract 1 second

t2 = time("2024-01-01T00:00:01Z")
t2 - t              // 1000000000 -- int, the duration between them in nanoseconds

t < t2               // true  -- chronological ordering
t2 > t               // true
t2 >= t2             // true
```

### What an `int` means next to a `time`

An `int` plays two different roles around `time`, and **the position decides which one**:

| position | role | encoding |
|---|---|---|
| **operator** — `t + n`, `t - n`, `t2 - t` | a **duration** | nanoseconds |
| **conversion** — `time(n)`, `n.time()`, `t.int()`, `t.unix()` | an **instant** | unix timestamp |

Neither is a unit "choice" that could be made consistent with the other: nanoseconds is what a duration
is throughout the language (`times.parse_duration("1h")` is `3600000000000`, `times.since`/`until`,
`times.sleep`, every `times.duration_*`), and a unix timestamp is how the outside world encodes an
instant in an integer. They are two different concepts that happen to share one type, so each keeps the
encoding conventional for *it*. No occurrence is ever in both positions, so no occurrence is ambiguous.

**Operator position — nanoseconds.** `time + int`/`time - int` adds/subtracts that many nanoseconds, and
`time - time` yields the nanosecond duration between the two instants. There is no unary `-` for `time`
(a duration has no natural "negative instant" reading).

**Conversion position — unix timestamp.** Seconds by default (`time(n)`, `n.time()`, `t.int()`,
`t.unix()`); every other encoding is named by the function that reads or writes it — `n.time_ms()`,
`n.time_micro()`, `n.time_nano()` in, `t.unix_ms()`, `t.unix_micro()`, `t.unix_nano()` out, plus
`times.from_unix*` and `times.time_unix*`. Every conversion produces UTC, so it never depends on the
host's timezone.

Because the encodings differ, only the matching pair round-trips exactly:

```go
t = time("2024-01-01T00:00:00.123456789Z")

t.unix_nano().time_nano() == t    // true  -- matching encoding, lossless
t.int().time() == t               // false -- the seconds encoding truncates sub-second precision
```

**There is deliberately no ordering or equality between `time` and a bare `int`.** `t < 5` would have to
pick one of the two roles, and either reading makes a real mistake silent: under the duration reading it
compares an instant against a length of time, and under the timestamp reading `t < t2 - t` silently
compares against 1970 instead of erroring. Equality has a second problem — since `int == string` is true
for the canonical text form, admitting `time == int` would make `t == 1704067200` and
`1704067200 == "1704067200"` both true while `t == "1704067200"` stays false, i.e. non-transitive
equality. Convert explicitly instead, which also documents which role you meant:

```go
t < time(1704067200)                        // instant vs instant
t < times.from_unix_ms(1704067200000)       // ... from a millisecond timestamp
t.unix() < 1704067200                       // int vs int
```

There is likewise no arithmetic or ordering against `float`, `string`, or any other type — construct
another `time` first (e.g. via `time(...)`) if you need to compare against one.

## Member Functions

### General Functions

#### `copy()`

Returns the value itself.

**Arguments:** None

**Returns:** `time`

**Description:** Provided for symmetry with the builtin `copy(x)` function. Since `time` is immutable, this method
returns the receiver unchanged.

```go
t = time("2024-01-01")
t.copy()    // same time value
```

#### `format([spec])`

Renders the value as a string using the [Format Mini-Language](../format-mini-language.md).

**Arguments:**

- `spec` (optional, `string`) - format mini-language spec. Defaults to `""`.

**Returns:** `string`

**Description:** Equivalent to using the value as the operand of an f-string interpolation, e.g.
`f"{x:<spec>}"` - except the spec is parsed on each call rather than at compile time. With no argument or with an empty
string the type's default rendering is returned. The set of accepted verbs and modifiers is type-specific;
see [Format Mini-Language](../format-mini-language.md) for the full grammar.

```go
t = time("2024-01-02T03:04:05Z")
t.format()                   // "2024-01-02T03:04:05Z"
t.format("#date")            // "2024-01-02"
t.format("#%Y-%m-%d %H:%M:%S") // "2024-01-02 03:04:05"
```

### Conversion Functions

#### `time()`

Converts to time.

**Arguments:** None

**Returns:** `time`

**Description:** Returns the same time value.

```go
time("2024-01-01").time()    // 2024-01-01
```

#### `bool()`

Converts to boolean.

**Arguments:** None

**Returns:** `bool`

**Description:** Returns `true` for all valid time values.

```go
time("2024-01-01").bool()    // true
```

#### `int()`

Converts to integer.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp (seconds since epoch) — the default int encoding of an
instant, same as `unix()`. Sub-second precision is truncated; use `unix_ms()`/`unix_micro()`/
`unix_nano()` when it matters. This is a *conversion*, so the int is a timestamp, not a duration — see
"What an `int` means next to a `time`" above.

```go
time("1970-01-01T00:00:00Z").int()   // 0
time("2024-01-01T00:00:00Z").int()   // 1704067200
```

#### `string()`

Converts to string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the time in ISO 8601 format (RFC 3339).

```go
time("2024-01-01").string()            // "2024-01-01T00:00:00Z"
time("2024-01-01T12:30:45Z").string()  // "2024-01-01T12:30:45Z"
```

### Date and Time Field Functions

#### `year()`

Gets the year.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the year component (e.g., 2024).

```go
time("2024-06-15").year()    // 2024
```

#### `month()`

Gets the month.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the month (1-12).

```go
time("2024-06-15").month()   // 6
```

#### `day()`

Gets the day of month.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the day of the month (1-31).

```go
time("2024-06-15").day()     // 15
```

#### `hour()`

Gets the hour.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the hour (0-23).

```go
time("2024-01-01T14:30:00Z").hour()  // 14
```

#### `minute()`

Gets the minute.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the minute (0-59).

```go
time("2024-01-01T14:30:45Z").minute()  // 30
```

#### `second()`

Gets the second.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the second (0-59).

```go
time("2024-01-01T14:30:45Z").second()  // 45
```

#### `nanosecond()`

Gets the nanosecond.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the nanosecond component (0-999999999).

```go
time("2024-01-01T00:00:00.123456789Z").nanosecond()  // 123456789
```

### Epoch and Calendar Metadata Functions

#### `unix()`

Gets Unix timestamp in seconds.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp (seconds since epoch).

```go
time("1970-01-01T00:00:00Z").unix()    // 0
time("2024-01-01T00:00:00Z").unix()    // 1704067200
```

#### `unix_ms()`

Gets Unix timestamp in milliseconds.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp in milliseconds. The inverse of `int.time_ms()` and of
`times.from_unix_ms()`.

```go
time("1970-01-01T00:00:00Z").unix_ms()              // 0
time("2024-01-01T00:00:00.123Z").unix_ms()          // 1704067200123
```

#### `unix_micro()`

Gets Unix timestamp in microseconds.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp in microseconds. The inverse of `int.time_micro()` and of
`times.from_unix_micro()`.

```go
time("1970-01-01T00:00:00Z").unix_micro()              // 0
time("2024-01-01T00:00:00.123456Z").unix_micro()       // 1704067200123456
```

#### `unix_nano()`

Gets Unix timestamp in nanoseconds.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp in nanoseconds. The inverse of `int.time_nano()` and of
`times.from_unix_nano()`, and the only encoding that round-trips a sub-second instant exactly.

```go
time("1970-01-01T00:00:00Z").unix_nano()                    // 0
time("2024-01-01T00:00:00.123456789Z").unix_nano()          // 1704067200123456789
```

#### `week_day()`

Gets day of week.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the day of the week (0=Sunday, 1=Monday, ..., 6=Saturday).

```go
time("2024-01-01").week_day()  // 1 (Monday, January 1, 2024)
```

#### `year_day()`

Gets day of year.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the day of the year (1-366).

```go
time("2024-01-01").year_day()   // 1
time("2024-12-31").year_day()   // 366 (leap year)
```

#### `month_name()`

Gets month name.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the full month name in English.

```go
time("2024-06-15").month_name()  // "June"
```

#### `week_day_name()`

Gets day of week name.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the full day name in English.

```go
time("2024-01-01").week_day_name()  // "Monday"
```

### Timezone and Formatting Functions

#### `utc()`

Converts to UTC.

**Arguments:** None

**Returns:** `time`

**Description:** Returns the time in UTC timezone.

```go
t = time("2024-01-01T12:30:00+05:30")
utc_t = t.utc()  // 2024-01-01T07:00:00Z
```

#### `local()`

Converts to local timezone.

**Arguments:** None

**Returns:** `time`

**Description:** Returns the time in the local timezone.

```go
t = time("2024-01-01T00:00:00Z")
local_t = t.local()  // Converts to local time
```

#### `format_date()`

Formats as date string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the date in YYYY-MM-DD format.

```go
time("2024-06-15T14:30:00Z").format_date()  // "2024-06-15"
```

#### `format_time()`

Formats as time string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the time in HH:MM:SS format.

```go
time("2024-06-15T14:30:45Z").format_time()  // "14:30:45"
```

#### `format_datetime()`

Formats as datetime string.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the full datetime in a human-readable format.

```go
time("2024-06-15T14:30:45Z").format_datetime()
// "2024-06-15 14:30:45"
```

#### `zone_offset()`

Gets timezone offset.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the timezone offset in seconds from UTC.

```go
time("2024-01-01T00:00:00Z").zone_offset()          // 0
time("2024-01-01T00:00:00+05:30").zone_offset()     // 19800 (5.5 hours)
```

#### `zone_name()`

Gets timezone name.

**Arguments:** None

**Returns:** `string`

**Description:** Returns the timezone abbreviation or name.

```go
time("2024-01-01T00:00:00Z").zone_name()   // "UTC"
```

### Sequence Functions

#### `repeat(n)`

Repeats the time `n` times into an array.

**Arguments:**

- `n` (int): Non-negative repeat count.

**Returns:** `array`

**Description:** Returns a new array of length `n` where every element equals the receiver. Errors when `n < 0`.

```go
t := time(0)
t.repeat(3).len()   // 3
t.repeat(0)         // []
```

## Examples

### Time Formatting

```go
fmt = import("fmt")

// Format times for display
meeting = time("2024-06-15T14:30:00Z")

date_str = meeting.format_date()      // "2024-06-15"
time_str = meeting.format_time()      // "14:30:00"
datetime_str = meeting.format_datetime()  // "2024-06-15 14:30:00"

message = "Meeting on " + date_str + " at " + time_str
fmt.println(message)  // "Meeting on 2024-06-15 at 14:30:00"
```

### Timezone Handling

```go
fmt = import("fmt")

// Handle different timezones
utc_time = time("2024-01-01T00:00:00Z")
offset_time = time("2024-01-01T00:00:00+05:30")

fmt.println("UTC: " + utc_time.string())
fmt.println("Offset: " + offset_time.string())

// Convert to UTC
normalized = offset_time.utc()
fmt.println("Normalized: " + normalized.string())
```

### Day-of-Week Operations

```go
fmt = import("fmt")

// Check day of week
dates = [
    time("2024-01-01"),
    time("2024-01-09"),
    time("2024-01-17")
]

for date in dates {
    day_name = date.week_day_name()
    fmt.println(date.format_date() + " is a " + day_name)
}
```

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
t4 = time(1704067200)                    // Unix timestamp (int)
```

### Input Formats

Time constructor automatically detects various formats and parses them accordingly.

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

**Description:** Returns the Unix timestamp (seconds since epoch).

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

#### `unix_nano()`

Gets Unix timestamp in nanoseconds.

**Arguments:** None

**Returns:** `int`

**Description:** Returns the Unix timestamp in nanoseconds.

```go
time("1970-01-01T00:00:00Z").unix_nano()    // 0
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

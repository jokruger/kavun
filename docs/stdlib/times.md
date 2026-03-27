# Module `times`

```text
times := import("times")
```

## Constants

Formats: `format_ansic`, `format_unix_date`, `format_ruby_date`, `format_rfc822`, `format_rfc822z`, `format_rfc850`, `format_rfc1123`, `format_rfc1123z`, `format_rfc3339`, `format_rfc3339_nano`, `format_kitchen`, `format_stamp`, `format_stamp_milli`, `format_stamp_micro`, `format_stamp_nano`.

Durations: `nanosecond`, `microsecond`, `millisecond`, `second`, `minute`, `hour`.

Months: `january` through `december`.

## Functions

- `sleep(duration int)`
- `parse_duration(value string) => int`
- `since(t time) => int`
- `until(t time) => int`
- `duration_hours(duration int) => float`
- `duration_minutes(duration int) => float`
- `duration_nanoseconds(duration int) => int`
- `duration_seconds(duration int) => float`
- `duration_string(duration int) => string`
- `month_string(month int) => string`
- `date(year, month, day, hour, min, sec, nsec, loc string) => time`
- `now() => time`
- `parse(format string, value string) => time`
- `unix(sec int, nsec int) => time`
- `add(t time, duration int) => time`
- `add_date(t time, years int, months int, days int) => time`
- `sub(t time, u time) => int`
- `after(t time, u time) => bool`
- `before(t time, u time) => bool`
- `time_year(t time) => int`
- `time_month(t time) => int`
- `time_day(t time) => int`
- `time_weekday(t time) => int`
- `time_hour(t time) => int`
- `time_minute(t time) => int`
- `time_second(t time) => int`
- `time_nanosecond(t time) => int`
- `time_unix(t time) => int`
- `time_unix_nano(t time) => int`
- `time_format(t time, format string) => string`
- `time_location(t time) => string`
- `time_string(t time) => string`
- `is_zero(t time) => bool`
- `to_local(t time) => time`
- `to_utc(t time) => time`
- `in_location(t time, loc string) => time`

# Standard Library

This document covers the main builtin modules in Kavun stdlib:

- `base64`
- `fmt`
- `hex`
- `json`
- `math`
- `os`
- `rand`
- `text`
- `times`

Notes:

- Signatures below use Kavun-facing names and argument order.
- `error` in return descriptions means the function returns an error value on failure.
- Some modules also export constants (for example `math`, `times`, `os`).

## base64

Example:

```go
base64 = import("base64")
base64.encode(bytes("hello"))
```

- `base64.encode(data bytes) -> string`: Standard Base64 encode.
- `base64.decode(s string) -> bytes | error`: Standard Base64 decode.
- `base64.raw_encode(data bytes) -> string`: Raw standard Base64 encode (no padding).
- `base64.raw_decode(s string) -> bytes | error`: Raw standard Base64 decode.
- `base64.url_encode(data bytes) -> string`: URL-safe Base64 encode.
- `base64.url_decode(s string) -> bytes | error`: URL-safe Base64 decode.
- `base64.raw_url_encode(data bytes) -> string`: Raw URL-safe Base64 encode (no padding).
- `base64.raw_url_decode(s string) -> bytes | error`: Raw URL-safe Base64 decode.

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
- `hex.decode(s string) -> bytes | error`: Hex-decode string.

## json

Example:

```go
json = import("json")
json.encode({"a": 1, "b": true})
```

- `json.decode(data bytes|string) -> value | error`: Decode JSON bytes into Kavun value.
- `json.encode(value) -> bytes | error`: Encode Kavun value into JSON bytes.
- `json.indent(data bytes|string, prefix string, indent string) -> bytes | error`: Pretty-format JSON bytes.
- `json.html_escape(data bytes|string) -> bytes | error`: Escape JSON for safe HTML embedding.

## math

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
- `math.max(x float, y float) -> float`: Larger value.
- `math.min(x float, y float) -> float`: Smaller value.
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
- `os.chdir(dir string) -> error`: Change current working directory.
- `os.chmod(path string, mode int) -> error`: Change file mode bits.
- `os.chown(path string, uid int, gid int) -> error`: Change owner and group.
- `os.clear_env() -> undefined`: Clear all environment variables.
- `os.environ() -> [string]`: Environment as `KEY=VALUE` strings.
- `os.exit(code int) -> undefined`: Exit process with code.
- `os.expand_env(s string) -> string`: Expand `$VAR` references.
- `os.get_egid() -> int`: Effective GID.
- `os.get_env(key string) -> string`: Environment value (empty if missing).
- `os.get_euid() -> int`: Effective UID.
- `os.get_gid() -> int`: Real GID.
- `os.get_groups() -> [int] | error`: Supplementary group IDs.
- `os.get_page_size() -> int`: Memory page size.
- `os.get_pid() -> int`: Current process ID.
- `os.get_ppid() -> int`: Parent process ID.
- `os.get_uid() -> int`: Real UID.
- `os.get_wd() -> string | error`: Current working directory.
- `os.hostname() -> string | error`: Hostname.
- `os.lchown(path string, uid int, gid int) -> error`: Change owner/group of symlink target entry.
- `os.link(old_path string, new_path string) -> error`: Create hard link.
- `os.lookup_env(key string) -> string | false`: Lookup env var with presence flag.
- `os.mkdir(path string, perm int) -> error`: Create directory.
- `os.mkdir_all(path string, perm int) -> error`: Create directory tree.
- `os.read_link(path string) -> string | error`: Read symlink target.
- `os.remove(path string) -> error`: Remove file or empty directory.
- `os.remove_all(path string) -> error`: Remove path recursively.
- `os.rename(old_path string, new_path string) -> error`: Rename/move path.
- `os.set_env(key string, value string) -> error`: Set environment variable.
- `os.symlink(old_path string, new_path string) -> error`: Create symbolic link.
- `os.temp_dir() -> string`: System temporary directory.
- `os.truncate(path string, size int) -> error`: Truncate file.
- `os.unset_env(key string) -> error`: Unset environment variable.
- `os.create(path string) -> file | error`: Create file, returns file record.
- `os.open(path string) -> file | error`: Open file (read-only), returns file record.
- `os.open_file(path string, flag int, perm int) -> file | error`: Open file with flags/mode, returns file record.
- `os.find_process(pid int) -> process | error`: Find process by PID.
- `os.start_process(name string, argv [string], dir string, env [string]) -> process | error`: Start process.
- `os.exec_look_path(file string) -> string | error`: Search executable in PATH.
- `os.exec(name string, args...) -> command`: Build exec command record.
- `os.stat(path string) -> fileinfo | error`: File metadata record.
- `os.read_file(path string) -> bytes | error`: Read file contents.

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
- `rand.read(buf bytes) -> int | error`: Fill byte buffer with random data, return bytes written.
- `rand.rand(seed int) -> rng`: Create independent RNG record.

### rand rng record

`rng` has the same callable methods as module-level random generator:

- `int()`, `float()`, `int_n(n)`, `exp_float()`, `norm_float()`, `perm(n)`, `seed(seed)`, `read(buf)`.

## text

Example:

```go
text = import("text")
text.trim_space("  hello  ")
```

- `text.re_match(pattern string, s string) -> bool | error`: Regex full/partial match check.
- `text.re_find(pattern string, s string, count? int) -> [match] | undefined | error`: Regex find with optional limit.
- `text.re_replace(pattern string, s string, repl string) -> string | error`: Regex replace all.
- `text.re_split(pattern string, s string, count? int) -> [string] | error`: Regex split with optional limit.
- `text.re_compile(pattern string) -> regexp | error`: Compile regex into reusable object.
- `text.compare(a string, b string) -> int`: Lexicographic compare.
- `text.contains(s string, substr string) -> bool`: Substring test.
- `text.contains_any(s string, chars string) -> bool`: Any-char containment test.
- `text.count(s string, substr string) -> int`: Substring occurrence count.
- `text.equal_fold(a string, b string) -> bool`: Case-insensitive Unicode compare.
- `text.fields(s string) -> [string]`: Split by Unicode whitespace.
- `text.has_prefix(s string, prefix string) -> bool`: Prefix test.
- `text.has_suffix(s string, suffix string) -> bool`: Suffix test.
- `text.index(s string, substr string) -> int`: First substring index or `-1`.
- `text.index_any(s string, chars string) -> int`: First index of any char or `-1`.
- `text.join(parts [string], sep string) -> string`: Join strings.
- `text.last_index(s string, substr string) -> int`: Last substring index or `-1`.
- `text.last_index_any(s string, chars string) -> int`: Last index of any char or `-1`.
- `text.repeat(s string, count int) -> string`: Repeat string count times.
- `text.replace(s string, old string, new string, n int) -> string`: Replace up to n occurrences (`n < 0` for all).
- `text.substr(s string, lower int, upper? int) -> string`: Slice by rune index.
- `text.split(s string, sep string) -> [string]`: Split by separator.
- `text.split_after(s string, sep string) -> [string]`: Split and keep separator suffix.
- `text.split_after_n(s string, sep string, n int) -> [string]`: Split-after with limit.
- `text.split_n(s string, sep string, n int) -> [string]`: Split with limit.
- `text.title(s string) -> string`: Title-case string.
- `text.to_lower(s string) -> string`: Lowercase transform.
- `text.to_title(s string) -> string`: Titlecase transform.
- `text.to_upper(s string) -> string`: Uppercase transform.
- `text.pad_left(s string, width int, pad? string) -> string`: Left-pad string.
- `text.pad_right(s string, width int, pad? string) -> string`: Right-pad string.
- `text.trim(s string, cutset string) -> string`: Trim both sides by cutset.
- `text.trim_left(s string, cutset string) -> string`: Trim left by cutset.
- `text.trim_prefix(s string, prefix string) -> string`: Remove prefix if present.
- `text.trim_right(s string, cutset string) -> string`: Trim right by cutset.
- `text.trim_space(s string) -> string`: Trim Unicode whitespace.
- `text.trim_suffix(s string, suffix string) -> string`: Remove suffix if present.
- `text.atoi(s string) -> int | error`: Parse base-10 integer.
- `text.format_bool(v bool) -> string`: Format boolean.
- `text.format_float(f float, fmt char|string, prec int, bits int) -> string`: Format float.
- `text.format_int(i int, base int) -> string`: Format integer in base.
- `text.itoa(i int) -> string`: Integer to decimal string.
- `text.parse_bool(s string) -> bool | error`: Parse boolean text.
- `text.parse_float(s string, bits int) -> float | error`: Parse float.
- `text.parse_int(s string, base int, bits int) -> int | error`: Parse integer.
- `text.quote(s string) -> string`: Go-style quoted literal.
- `text.unquote(s string) -> string | error`: Unquote Go-style literal.

## times

Example:

```go
times = import("times")
times.time_format(times.now(), times.format_rfc3339)
```

Constants:

- Time format layouts: `format_ansic`, `format_unix_date`, `format_ruby_date`, `format_rfc822`, `format_rfc822z`, `format_rfc850`, `format_rfc1123`, `format_rfc1123z`, `format_rfc3339`, `format_rfc3339_nano`, `format_kitchen`, `format_stamp`, `format_stamp_milli`, `format_stamp_micro`, `format_stamp_nano`.
- Duration units (nanoseconds): `nanosecond`, `microsecond`, `millisecond`, `second`, `minute`, `hour`.
- Months: `january`, `february`, `march`, `april`, `may`, `june`, `july`, `august`, `september`, `october`, `november`, `december`.

- `times.sleep(duration int) -> undefined`: Sleep for duration (nanoseconds).
- `times.parse_duration(s string) -> int | error`: Parse duration string to nanoseconds.
- `times.since(t time) -> int`: Elapsed duration since time (nanoseconds).
- `times.until(t time) -> int`: Duration until time (nanoseconds).
- `times.duration_hours(d int) -> float`: Duration to hours.
- `times.duration_minutes(d int) -> float`: Duration to minutes.
- `times.duration_nanoseconds(d int) -> int`: Duration to nanoseconds.
- `times.duration_seconds(d int) -> float`: Duration to seconds.
- `times.duration_string(d int) -> string`: Duration text format.
- `times.month_string(month int) -> string`: Month name.
- `times.date(year int, month int, day int, hour int, min int, sec int, nsec int, location? string) -> time`: Build time value.
- `times.now() -> time`: Current local time.
- `times.parse(layout string, value string) -> time | error`: Parse with layout.
- `times.unix(sec int, nsec int) -> time`: Unix timestamp to time.
- `times.add(t time, d int) -> time`: Add duration to time.
- `times.add_date(t time, years int, months int, days int) -> time`: Add calendar date components.
- `times.sub(t time, u time) -> int`: Difference `t-u` in nanoseconds.
- `times.after(t time, u time) -> bool`: Whether `t` is after `u`.
- `times.before(t time, u time) -> bool`: Whether `t` is before `u`.
- `times.time_year(t time) -> int`: Year component.
- `times.time_month(t time) -> int`: Month component.
- `times.time_day(t time) -> int`: Day of month.
- `times.time_weekday(t time) -> int`: Weekday index.
- `times.time_hour(t time) -> int`: Hour component.
- `times.time_minute(t time) -> int`: Minute component.
- `times.time_second(t time) -> int`: Second component.
- `times.time_nanosecond(t time) -> int`: Nanosecond component.
- `times.time_unix(t time) -> int`: Unix seconds.
- `times.time_unix_nano(t time) -> int`: Unix nanoseconds.
- `times.time_format(t time, layout string) -> string`: Format time.
- `times.time_location(t time) -> string`: Location name.
- `times.time_string(t time) -> string`: Default string format.
- `times.is_zero(t time) -> bool`: Zero-time check.
- `times.to_local(t time) -> time`: Convert to local timezone.
- `times.to_utc(t time) -> time`: Convert to UTC.
- `times.in_location(t time, location string) -> time | error`: Convert to named location.

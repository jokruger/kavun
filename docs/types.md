# Type Reference

This section describes each builtin type in detail.

## undefined

Represents absence of value.

- Field/index access on `undefined` returns `undefined`.
- It is falsy.
- Many conversion builtins return `undefined` on conversion failure unless a fallback is provided.

```go
u := undefined
u.anything        // undefined
u[0]              // undefined
```

## bool

Boolean values are `true` and `false`.

- Used directly in control flow (`if`, `for condition`).
- Participates in coercive equality/comparisons where appropriate.

```go
ok := true
ok && false   // false
```

Bool member functions:

- Conversion: `to_bool()`, `to_int()`, `to_string()`

## int

Signed integer type.

Declaration and usage:

```go
i := 42
i2 := 0x2a
```

Int member functions:

- Conversion: `to_int()`, `to_float()`, `to_decimal()`, `to_bool()`, `to_char()`, `to_string()`, `to_time()`
- Numeric utilities: `sign()`, `abs()`

## float

Floating-point type.

Declaration and usage:

```go
f := 3.14
g := 1e3
h := 1f
```

Float member functions:

- Conversion: `to_float()`, `to_decimal()`, `to_int()`, `to_string()`
- Numeric utilities: `sign()`

## decimal

`decimal` is an exact decimal type.

Declaration and construction:

```go
a := 1.23d
b := 123d

a2 := decimal(123)       // int -> decimal
b2 := decimal(1.23)      // float -> decimal
c2 := decimal("1.23")    // string -> decimal

d := (123).to_decimal()
e := (1.23).to_decimal()
f := "1.23".to_decimal()
```

`decimal(x)` conversion rules:

- `decimal()` returns `decimal(0)`.
- `decimal(decimalValue)` returns the same decimal value.
- `decimal(x)` attempts runtime conversion via `AsDecimal`.
- `decimal(x, fallback)` returns `fallback` when conversion fails.

`to_decimal()` member methods exist on:

- `int.to_decimal()`
- `float.to_decimal()`
- `string.to_decimal()`
- `decimal.to_decimal()`

Notable edge cases from implementation:

- `float.to_decimal()` returns decimal `NaN` for `NaN` and infinities.
- `string.to_decimal()` returns decimal `NaN` for invalid input.
- `decimal("bad")` returns `undefined` (or fallback when provided), because conversion failure is tracked separately from the returned decimal value.

Automatic conversions in mixed operations involving `decimal`:

- `decimal op x` converts `x` to decimal (if possible); arithmetic result is decimal.
- `int op decimal` promotes `int` to decimal; arithmetic result is decimal.
- `float op decimal` keeps float semantics (decimal converted to float); arithmetic result is float.
- `string + decimal` is valid only when string is on the left; decimal is converted to string.

Examples:

```go
decimal(1) + 2         // decimal(3)
1 + decimal(2)         // decimal(3)
decimal(1) + 2.0       // decimal(3)
1.0 + decimal(2)       // 3.0 (float)
"value=" + decimal(2)  // "value=2"
```

Decimal member functions:

- Conversion: `to_decimal()`, `to_float()`, `to_int()`, `to_string()`
- Classification: `is_zero()`, `is_negative()`, `is_positive()`, `is_nan()`
- Metadata: `sign()`, `scale()`, `error_details()`
- Scale/normalization: `to_scale(scale)`, `canonical()`, `trunc(scale)`
- Neighbor/value transforms: `next_up()`, `next_down()`, `abs()`, `negate()`, `sqrt()`
- Rounding: `round_down(scale)`, `round_up(scale)`, `round_toward_zero(scale)`, `round_away_from_zero(scale)`, `round_half_toward_zero(scale)`, `round_half_away_from_zero(scale)`, `round_bank(scale)`

For decimal methods that accept `scale`, the argument must be an `int` in the implementation-defined decimal scale range; otherwise a runtime error is raised.

## char

`char` is a single Unicode rune.

Declaration and usage:

```go
c := 'A'
'A' + 1   // 66 (int)
'9' - '0' // 9 (int)
```

Char member functions:

- Conversion: `to_char()`, `to_bool()`, `to_int()`, `to_string()`

## string

Strings are immutable and indexed by Unicode rune (not byte).

Declaration and usage:

```go
s := "ウクライナ"
s[0]         // char 'ウ'
s[0:2]       // "ウク"
len(s)       // 5 (rune count)
```

String member functions:

- Conversion: `to_string()`, `to_array()`, `to_bool()`, `to_bytes()`, `to_char()`, `to_float()`, `to_int()`, `to_decimal()`, `to_time()`, `to_record()`
- Queries and accessors: `is_empty()`, `len()`, `first()`, `last()`, `contains(x)`
- Transformations: `lower()`, `upper()`, `trim([cutset])`

## bytes

Bytes are mutable byte arrays. Indexing returns an `int` in `0..255`.

Declaration and usage:

```go
b := bytes("abc")
b[0]                            // 97
b[0:2]                          // bytes slice
bytes("abc") + bytes("def")     // concatenation
```

Bytes member functions:

- Conversion: `to_bytes()`, `to_array()`, `to_record()`, `to_string()`
- Queries and accessors: `is_empty()`, `len()`, `first()`, `last()`, `contains(x)`

## time

Time values are builtin scalar values, usually created via `time(...)`.

```go
t := time("2024-01-01")
```

Time member functions:

- Conversion: `to_time()`, `to_bool()`, `to_int()`, `to_string()`
- Date and time fields: `year()`, `month()`, `day()`, `hour()`, `minute()`, `second()`, `nanosecond()`
- Epoch and calendar metadata: `unix()`, `unix_nano()`, `week_day()`, `year_day()`, `month_name()`, `week_day_name()`
- Timezone and formatting: `to_utc()`, `to_local()`, `format_date()`, `format_time()`, `format_datetime()`, `zone_offset()`, `zone_name()`

## error

Error values carry payload and are first-class values.

```go
e := error("something went wrong")
e.value()      // "something went wrong"
is_error(e)    // true
```

Error member functions:

- Accessors: `value()`
- Conversion: `to_string()`

## array

Arrays are mutable and reference-typed (`a := b` makes both variables point at the same array).

Declaration and usage:

```go
a := [1, 2, 3]
b := a
a[0] = 99
// b[0] == 99
```

Array member functions:

- Conversion: `to_array()`, `to_bytes()`, `to_string()`, `to_record()`
- Transformations and filtering: `sort()`, `filter(fn)`, `map(fn)`
- Predicates and matching: `all(fn)`, `any(fn)`, `contains(x)`
- Aggregations: `count(fn)`, `reduce(init, fn)`, `min()`, `max()`, `sum()`, `avg()`
- Queries and accessors: `is_empty()`, `len()`, `first()`, `last()`

Lambda callbacks for `filter`/`map`/etc. can accept one argument (value) or two (index, value):

```go
[1, 2, 3, 4].filter(x => x % 2 == 0)          // [2, 4]
[1, 2, 3].map((i, v) => i * v)                // [0, 2, 6]
[1, 2, 3].reduce(0, (acc, v) => acc + v)      // 6
```

## record

Records are the primary object type. Keys are strings. Both dot notation and index notation work.

Declaration and usage:

```go
r := {name: "Alice", age: 30}
r.name       // "Alice"
r["age"]     // 30
r.missing    // undefined

r.city = "Berlin"   // add new key
delete(r, "age")    // remove key

"name" in r  // true - key existence check
```

Records are reference-typed.

Records do not expose member functions. Selector access is for fields.

## map

`map` is similar to a record but only supports index access for elements; selector access is reserved for map member functions.

Declaration and usage:

```go
m := map({a: 1, b: 2})
m["a"]       // 1
m.a          // runtime error - dot access not allowed on map elements

m.keys()     // array of keys (unsorted)
m.values()   // array of values
```

Record and map relationship:

Records and maps represent the same logical structure: a string-keyed dictionary with values of any type. Converting a record with `map(record)` does not copy data; the resulting map points to the same underlying structure.

Access rules:

- Record: selector and index access read/write elements (`r.name`, `r["name"]`); no member functions.
- Map: index access reads/writes elements (`m["name"]`); selector access is for map member functions (for example `m.len()`, `m.filter(...)`, `m.keys()`).

Map member functions:

- Conversion: `to_record()`
- Queries and accessors: `is_empty()`, `len()`, `keys()`, `values()`, `contains(x)`
- Filtering and predicates: `filter(fn)`, `count(fn)`, `all(fn)`, `any(fn)`

## range

Ranges are lazy sequences.

Declaration and usage:

```go
range(0, 5).to_array()       // [0, 1, 2, 3, 4]
range(5, 0, 1).to_array()    // [5, 4, 3, 2, 1]
range(0, 10, 2).contains(4)  // true

for v in range(1, 4) { }     // v = 1, 2, 3
```

Range member functions:

- Conversion: `to_array()`, `to_bytes()`, `to_string()`, `to_record()`
- Queries and accessors: `is_empty()`, `len()`, `contains(x)`

## immutable wrappers

`immutable(x)` makes a container immutable at container level. Mutating an immutable container raises a runtime error.

`copy()` always returns a mutable deep copy, even from an immutable value:

```go
a := immutable([1, 2, 3])
a[0] = 9       // runtime error
type_name(a)   // "immutable-array"

b := copy(a)   // mutable copy
b[0] = 9       // ok
```

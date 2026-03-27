# GS Type System

The GS runtime exposes a compact set of value types. Everything implements the `core.Object` interface and therefore shares a small surface area (conversion helpers, selector access, truthiness). The behavior documented here is enforced by `tests/unit` in the repository.

## Value Categories

| Category | Types |
| --- | --- |
| Scalars | `int`, `float`, `bool`, `char` |
| Text & Binary | `string`, `bytes` |
| Collections | `array`, `map`, `record` |
| Temporal | `time` |
| Executable | `function`, `lambda`, `builtin` |
| Special | `error`, `undefined`, immutable forms of the above |

## Mutability

Containers are mutable by default but can be frozen via `immutable(expr)`. The outer container becomes `immutable-array`, `immutable-record`, or `immutable-map`; nested values are left untouched so tests can mutate inner structures unless those are explicitly frozen too.

```text
nums := immutable([1, 2, [3, 4]])
nums[0] = 5         # runtime error
nums[2][0] = 9      # allowed (inner array is still mutable)
```

Use `copy(value)` when you need a fresh mutable copy and `is_immutable(value)` to branch on mutability at runtime. Primitive values are already immutable.

## Automatic Conversions

GS performs automatic conversions when operands require it:

- Numeric expressions promote values to the most precise numeric type appearing in the expression (`3 / 2.0 == 1.5` vs `3 / 2 == 1`).
- `string` concatenation calls `AsString()` on the right-hand side; numeric operations call the numeric conversion helpers (`"123" + 4 == "1234"` vs `4 + "123" == 127`).
- `bool.int`, `int.bool`, `string.bool`, `string.time`, etc. are exposed as selector properties to keep conversions explicit when necessary.

Use these selectors instead of ad-hoc helper functions so your scripts continue to match the behavior locked down by the tests.

## Records and Maps

Records are lightweight string-keyed structs. The literal `{}` syntax produces a record, records support selector syntax (`rec.name`), and they can be indexed via `rec["name"]`. Records deliberately avoid helper APIs so the structure is purely data.

Maps wrap a record and add helper members. Construct them with the `map()` builtin (or by importing a module that returns a map) and use properties such as `map.len`, `map.keys`, `map.values`, and higher-order helpers (`filter`, `count`, `all`, `any`). `map.record` returns a record view of the map and `record := immutable(record)` produces read-only records.

## Value Members

Every value exposes selectors that return either properties (no parentheses) or functions. The following lists highlight the selectors surfaced in `value/*.go`.

### array

- Properties: `array`, `bytes`, `string`, `record` (string-keyed record), `empty`, `len`, `first`, `last`, `min`, `max`, `sum`, `avg`.
- Functions: `sort()`, `filter(fn)`, `count(fn)`, `all(fn)`, `any(fn)`, `map(fn)`, `reduce(initial, fn)`.

Callbacks can accept `(value)` or `(index, value)`.

### bool

- Properties: `bool`, `int`, `string`.

### bytes

- Properties: `bytes`, `array` (array of ints), `record` (map from indexes to ints), `string`, `empty`, `len`, `first`, `last`.

### char

- Properties: `char`, `bool`, `int`, `string`.

### float

- Properties: `float`, `int`, `string`.

### int

- Properties: `int`, `float`, `bool`, `char`, `string`, `time` (via Unix seconds).

### map

- Properties: `map`, `record`, `empty`, `len`, `keys`, `values`.
- Functions: `filter(fn)`, `count(fn)`, `all(fn)`, `any(fn)`.

Callbacks can accept `(key)` or `(key, value)` depending on arity.

### record

Records convert to maps via `map(record)` and support selector/index access. They do not expose helper functions.

### string

- Properties: `string`, `array` (chars), `bool`, `bytes`, `char`, `float`, `int`, `time` (parsed using `dateparse`), `record` (index → char), `empty`, `len`, `first`, `last`, `lower`, `upper`.
- Functions: `trim(chars?)` (defaults to whitespace).

### time

- Properties: `time`, `bool` (true unless zero time), `int` (Unix seconds), `string`, `year`, `month`, `day`, `hour`, `minute`, `second`, `nanosecond`, `unix`, `unix_nano`, `week_day`, `year_day`, `month_name`, `week_day_name`, `utc`, `local`, `date_str`, `time_str`, `date_time_str`, `zone_offset`, `zone_name`.

### function and lambda

Functions expose `callable` semantics. Lambdas use the same runtime type but are declared with arrow syntax (`args => expression`). Functions can be variadic (`func foo(...xs)`) and calls can use spread arguments (`f(arr...)`).

### errors and undefined

`error("msg")` constructs an error object that carries a message string. `undefined` represents an absent value and is returned by lookups or functions that fail silently. Use `is_error(value)`/`is_undefined(value)` or direct comparisons to branch on them.

## Interfaces

Objects can opt into two optional interfaces:

- **Callable** – implements a `Call(args...)` method and can appear anywhere a function is expected. Checked via `is_callable()`.
- **Iterable** – implements `Iterate()` to return an iterator with `next()` and `value()` methods. Arrays, maps, records, strings, and bytes are iterable by default. Checked via `is_iterable()`.

The allocator helpers in `alloc/` produce the concrete types used here, and the builtins in `vm/builtins.go` expose predicates such as `is_array`, `is_record`, `is_map`, and `is_time` to keep runtime type-checking simple.

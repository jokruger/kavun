# Functions

GS exposes a small set of builtins in `vm/builtins.go`. They are registered in every VM (REPL, CLI, host runtime) and mirrored in `tests/unit/vm_test.go`.

## Collections and Conversions

- `len(value)` – returns the length of an array, string, bytes, record, or map.
- `copy(value)` – deep-copies the provided value. Containers become mutable copies even if the source was immutable.
- `append(array, values...)` – returns a new array with `values` appended.
- `delete(recordOrMap, key)` – removes a string key from a mutable record or map. Returns `undefined`.
- `splice(array, start?, count?, replacements...)` – mutates the mutable array, returns the deleted items as a new array, and inserts any replacement values.
- `format(pattern, ...values)` – returns a formatted string using the verbs in [`reference/formatting.md`](formatting.md).
- `range(start, stop, step?)` – returns an array of ints from `start` to `stop` (exclusive). `step` defaults to `1` and must be positive; if `start` is greater than `stop` the range counts down.
- `type_name(value)` – returns the runtime type name (`int`/`array`/etc.).
- `string(value, default?)`, `int(value, default?)`, `float(value, default?)`, `bool(value)`, `char(value, default?)`, `bytes(value, default?)`, `time(value, default?)` – convert values to the requested type. Passing no arguments returns the zero value (empty string, zero int, false, etc.). When a second argument is provided it is returned when conversion fails instead of `undefined`. `bytes(int)` allocates a zeroed buffer of the requested size.
- `map(value?)` – without arguments returns an empty mutable map. With one argument it clones the provided map or record into a new mutable map.

## Type Predicates

All predicates take a single argument and return a bool:

`is_string`, `is_int`, `is_float`, `is_bool`, `is_char`, `is_bytes`, `is_array`, `is_record`, `is_map`, `is_time`, `is_error`, `is_undefined`, `is_function`, `is_callable`, `is_iterable`, `is_immutable`.

Use these helpers in guards instead of manual `type_name` checks so your code lines up with the semantics enforced by the tests.

## Errors

- `error(message)` – keyword expression that constructs a runtime error value with the given message. The returned value behaves like any other object and can be compared, passed around, or assigned. Builtins also return errors to signal invalid argument types.

Remember that `undefined` is a keyword and can be compared directly or detected with `is_undefined(value)`.

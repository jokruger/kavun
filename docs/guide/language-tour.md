# Language Tour

## Values and Types

GS ships with a compact set of runtime types that each expose properties and member functions. Reassigning a variable binds a new value.

```text
answer := 42                        # int
ratio  := 3 / 2.0                   # float because one operand is float
ready  := true                      # bool
a := 'a'                            # char
letters := "abc"                    # string
byteset := bytes([97, 98, 99])      # bytes
items := [1, 2, 3]                  # array
state := map({count: 1})
config := {env: "dev", retries: 3}  # record literal
when := time("2009-02-13 23:31:30 +0000 UTC")
```

`undefined` is a keyword that represents the absence of a value. Array, map, and record lookups return `undefined` when the key does not exist, which keeps chains like `state.user.name` simple.

### Record vs Map

Records are lightweight string-keyed structs. Use `{}` literals to build them, access them through selectors (`record.key`) or index expressions (`record["key"]`). Records do not expose helper methods; they only carry fields.

Maps wrap a record and add helper members such as `map.len`, `map.keys`, and `map.filter`. Use the builtin `map()` to convert a record into a mutable map and `map.record` to go the other way.

```text
settings := {region: "uk", zone: "1"}    # record
dynamic := map(settings)                 # map
dynamic.keys.sort()                      # sorted array of keys
settings.zone                            # selector access
```

### Immutable values

Wrapping any expression with `immutable(expr)` freezes the outer container: arrays become immutable arrays, records become immutable records, and maps become immutable maps. Nested values are left untouched, which mirrors the unit tests:

```text
nums := immutable([1, 2, [3, 4]])
nums[0] = 5           # runtime error: immutable-array
nums[2][0] = 9        # allowed because the inner array is still mutable
profile := immutable({user: {name: "gs"}})
profile.user.name = "core"   # still allowed
```

Use `copy(value)` to obtain a mutable copy or `is_immutable(value)` to check at runtime.

## Automatic Type Conversion

GS performs type conversions automatically when operands require it:

- Numeric expressions promote values to the most precise numeric type appearing in the expression.
- When both operands implement the `string` conversion the left-hand side type decides whether the right-hand side is treated as a string or coerced to a number.

```text
ratio := 3 / 2.0     # float 1.5
whole := 3 / 2       # int 1
mix1 := "123" + 1    # "1231"
mix2 := 1 + "123"    # 124
```

Use explicit conversion members (`value.int`, `value.float`, `value.string`, `value.time`, etc.) when you need more control or are passing values into host code.

## Arrays, Strings, Bytes, and Records

Collections expose properties such as `len`, `first`, `last`, and `empty`. Many also provide higher-order helpers that accept lambda expressions.

```text
arr := [97, 98, 99]
arr.bytes      # bytes object
arr.record     # {"0": 97, "1": 98, "2": 99}
arr.sort()     # new sorted array
arr.filter(x => x >= 98)
arr.count((i, x) => x == i + 1)
arr.reduce(100, (acc, x) => acc + x)
```

Strings share the same interface. `"abc".array` returns `['a', 'b', 'c']`, `"abc".upper` yields `"ABC"`, and `" abc ".trim()` removes surrounding whitespace (or characters you pass in).

`bytes` values expose conversions to arrays (`bytes.array`), maps (`bytes.record`), and strings (`bytes.string`).

Records act as lightweight structs. Use selector access for convenience and the `record` property on other types (`array.record`, `map.record`) for conversions.

## Maps

Maps hold dynamic data and provide functional-style helpers. Each helper can receive one or two parameters (key/value or index/value depending on the type), which is why the unit tests call them with lambdas that accept either one or two arguments.

```text
data := map({a: 1, b: 2})
data.keys         # ["b", "a"] (order is not guaranteed)
data.keys.sort()  # ["a", "b"]
data.values       # array of values
data.filter(k => k != "a")
data.any((k, v) => v > 1)
```

## Functions, Lambdas, and Variadics

Functions are declared with `func` and return the last expression or an explicit `return`.
Lambda syntax (`=>`) keeps inline logic concise and is heavily used by array and map helpers.

```text
add := func(a, b) {
    return a + b
}

twice := x => x * 2
pair := (a, b) => { return a * b }

[1, 2, 3].map(twice)
```

Functions can be variadic. Use `func foo(...values)` to collect the remaining arguments into an array, and use `callable(arr...)` to spread an array (mutable or immutable) into positional arguments.

```text
collect := func(head, ...rest) {
    return append([head], rest...)
}
run := func(f, ...args) {
    return f(args...)
}
```

Lambdas capture variables from the current scope, so you can build callbacks and pipelines without ceremony.

## Control Flow

GS offers `if`, `for`, and `for-in` statements.

```text
if condition {
    // ...
} else if other {
    // ...
} else {
    // ...
}

for i := 0; i < 5; i++ {
    fmt.println(i)
}

for x in items {
    fmt.println(x)
}
```

`break` and `continue` behave as expected, and `for-in` supports iterating over arrays, maps, strings, bytes, and any user type that implements the iterator interface documented in [`reference/type-system.md`](../reference/type-system.md). Use the builtin `range(start, stop, step?)` when you need a numeric sequence: `for i in range(0, 10) { ... }`.

## Modules

Modules are loaded using `import("name")`. Module names can be absolute or relative; relative paths honor the CLI's `-resolve` flag. Global module state is shared between imports unless the module exposes a factory function.

When authoring modules, wrap the public API in an `export` block. The exported value is deep-copied, marked immutable, and returned to callers—as shown by the module tests in `tests/unit`.

```text
// file mod.gs
helper := func(value) {
    return value.upper
}

export {
    greet: func(name) { return "Hello " + helper(name) },
    meta:  immutable({version: 1})
}
```

```text
m := import("./mod")
m.greet("gs")      # => "Hello GS"
m.meta.version = 2  # runtime error: module export is immutable
```

## Comments

GS supports single-line (`// ...`) and block (`/* ... */`) comments. Comments can be used anywhere whitespace is allowed, which keeps parity with the test suite and makes large scripts easy to annotate.

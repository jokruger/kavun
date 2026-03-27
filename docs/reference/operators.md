# Operators

GS keeps the operator surface compact. Unary operators apply to a single operand, while binary operators accept two operands and respect the precedence table at the end of this document. A ternary conditional (`condition ? a : b`) completes the expression set.

## Unary Operators

| Operator | Meaning | Types |
| --- | --- | --- |
| `+x` | no-op sign | `int`, `float` |
| `-x` | negate | `int`, `float` |
| `!x` | logical NOT | any (uses truthiness rules) |
| `^x` | bitwise complement | `int` |

## Binary Operators

| Operator | Meaning | Types |
| --- | --- | --- |
| `==`, `!=` | equality / inequality | any |
| `&&`, `||` | logical AND / OR | any |
| `+` | addition / concatenation | `int`, `float`, `string`, `char`, `time`, `array` |
| `-` | subtraction | `int`, `float`, `char`, `time` |
| `*`, `/` | multiplication / division | `int`, `float` |
| `%` | remainder | `int` |
| `&`, `|`, `^`, `&^` | bitwise ops | `int` |
| `<<`, `>>` | bit shifts | `int` |
| `<`, `<=`, `>`, `>=` | comparisons | `int`, `float`, `char`, `time`, `string` |

Automatic type conversions apply when operands mismatch (e.g. `3 / 2.0` produces a float).

## Ternary Conditional

`condition ? whenTrue : whenFalse` evaluates the first expression when the condition is truthy, otherwise the second expression.

```text
my_min := func(a, b) { return a < b ? a : b }
fmt.println(my_min(5, 10))
```

## Assignment and Increments

| Operator | Expansion |
| --- | --- |
| `+=`, `-=`, `*=`, `/=` | `lhs = lhs <op> rhs` |
| `%=` | `lhs = lhs % rhs` |
| `&=`, `|=`, `^=`, `&^=` | bitwise compound assignments |
| `<<=`, `>>=` | shift compound assignments |
| `++`, `--` | increment or decrement by 1 (statements, not expressions) |

## Precedence

From highest to lowest (same level associates left-to-right):

1. Unary `+`, `-`, `!`, `^`
2. `*`, `/`, `%`, `<<`, `>>`, `&`, `&^`
3. `+`, `-`, `|`, `^`
4. Comparisons `==`, `!=`, `<`, `<=`, `>`, `>=`
5. `&&`
6. `||`
7. ternary `? :`

Parentheses override precedence as expected.

## Selectors, Indexers, and Slices

Use `.` to access record fields or module members and `[]` to index arrays, strings, bytes, maps, or records. For sequences, `[:]` creates slices.

```text
items := ["one", "two", "three"]
items[1]        # "two"

m := {
    a: 1,
    b: [2, 3, 4],
    call: func() { return 10 },
}
m.a             # 1
m["b"][1]       # 3
m.call()        # 10
m["missing"]    # undefined
m.b[0:2]        # [2, 3]
```

Selectors cannot be keywords. Use string keys plus index syntax when you must store values under names such as `in` or `func`.

## Immutable Expressions

Use `immutable(expr)` when you need to freeze the outer container of an array, record, or map. The returned value is marked as `immutable-*` and the runtime will raise an error if you try to assign through selectors or indexes. Nested values are left untouched unless they were also wrapped in `immutable(...)`.

```text
nums := immutable([1, 2, [3, 4]])
nums[0] = 5        # runtime error: immutable-array
nums[2][0] = 9     # allowed (inner array was not frozen)
```

Use `copy()` to obtain a mutable clone.

## Variadic Parameters and Spread Calls

Functions can declare variadic parameters with `...`:

```text
collect := func(head, ...rest) {
    return append([head], rest...)
}
```

Calls can spread an array or immutable array into positional arguments:

```text
args := [2, 3]
collect(1, args...)   # => [1, 2, 3]
```

This is the same syntax exercised throughout `tests/unit/vm_test.go`.

# bool

The two truth values, `true` and `false`.

## Overview

`bool` is a scalar with conversions only — no arithmetic, no elements. Two distinct questions meet here
and stay separate:

- **Conversion** — `bool(x)` / `x.bool()`: a numeric zero test, or a text *parse*.
- **Truthiness** — `is_true(x)` / `x.is_true()` / `!!x`: is this value different from its type's zero value?
  Truthiness is defined for *every* type; the logical operators `&&`/`||`/`!` and `if`/`for` conditions use
  it, not the conversion.

The two disagree on purpose: `bool("false")` → `false` (parses the word), `is_true("false")` → `true`
(a non-empty string).

`bool` values are immutable.

## Literals and Construction

```go
t := true
f := false
bool()          // false — the zero value
```

## Operators

### No arithmetic

`bool` takes part in no arithmetic at all — every combination raises:

```go
true + 1        // raises: bool + int
true + true     // raises: bool + bool
true * 2        // raises: bool * int
-true           // raises: - bool
```

Unary `^` is boolean negation of a `bool` (same-type complement); `!` is the universal truthiness negation:

```go
^true           // false
!true           // false
```

### Comparison

`false < true`; against the numeric scalars a `bool` compares as 0/1:

```go
false < true    // true
true <= true    // true
true == 1       // true
false == 0      // true
true < 2        // true
true < 1.5      // true
true == b'\x01' // true
true == 2       // false — equality is by value, not truthiness
true == "true"  // true — against text, equality compares the bool's text form
```

### Logical operators — truthiness, any type, short-circuiting

`&&` and `||` evaluate *truthiness* of any operand type, short-circuit, and answer the deciding **operand
value** (not a forced `bool`); `!` always answers a `bool`:

```go
1 && "a"        // "a" — left is truthy, right decides
0 && "a"        // 0   — left decides, right never evaluated
"" || "x"       // "x"
"y" || "x"      // "y"
!5              // false
!""             // true
!!"x"           // true — the idiomatic "as bool" spelling
```

A value whose truthiness is an error state raises rather than answering:

```go
float("nan") && true   // raises: float NaN is neither true nor false in a boolean context
```

## Members

The full roster: `bool` `int` `string` `runes` `copy` `freeze` `format` `is_true`.

### Conversions

Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given.

```go
true.int()      // 1
false.int()     // 0
true.string()   // "true"
true.runes()    // u"true"
true.bool()     // true — identity
```

There is no `float()`, `decimal()`, `byte()`, or `rune()` member — `int` is the sole gateway
(`true.int().float()`), and no `time()`/container targets exist.

### The free `bool(x)` conversion — zero test or parse

`bool(x)` (≡ `x.bool()` where the member exists) is defined on exactly two domains:

**Numeric** — the `bool → int` edge read backwards, a zero test:

```go
bool(0)               // false
bool(2)               // true
bool(-1)              // true
bool(0.0)             // false
bool(0.5)             // true
bool(decimal("0"))    // false
```

**Text** — a parse of the wide literal set, case-insensitive: `true`/`false`, `1`/`0`, `t`/`f`, `yes`/`no`.
Invalid text **raises** (never a silent `false`); the member's optional default rescues it:

```go
bool("false")         // false
bool("FALSE")         // false
bool("Yes")           // true
bool("t")             // true
bool("0")             // false
bool("abc")           // raises: cannot convert string to bool
bool("")              // raises — empty is not a boolean word
"abc".bool(false)     // false — the member's default rescues bad data
"false".bool()        // false — same operation, member spelling
```

Nothing else converts: `bool([])` and `bool(b'\x00')` raise — for every other type the question is
truthiness, spelled `is_true`/`!!x`.

### Truthiness — `is_true()`

Both spellings exist on every type — the member and the universal free form:

```go
true.is_true()        // true
false.is_true()       // false
is_true("false")      // true  — non-empty string; contrast bool("false") -> false
is_true("")           // false
is_true(0)            // false
is_true([])           // false
```

### Render — `format([spec])`

```go
true.format()         // "true" — default verb is t
true.format("t")      // "true"
true.format("d")      // "1"
false.format("d")     // "0"
```

### `copy()` / `freeze()`

Identity no-ops on an immutable scalar — kept so generic code never type-errors:

```go
true.copy()     // true
true.freeze()   // true
```

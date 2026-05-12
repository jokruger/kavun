# Examples

A short tour of Kavun through small, realistic snippets focused on the language itself — its expression-oriented style,
member chaining, lambdas, f-strings, decimals, records, closures, modules, and error values. Standard-library use is
kept to a minimum (`fmt` for output) so the language features stand on their own. For full reference material see the
[Language Reference](language.md) and the [Type Reference](types.md).

All snippets are self-contained and runnable with `kavun script.kvn`. Each example below also exists as a standalone
file in [`docs/examples/`](examples/).

## Expression-oriented data pipelines

Transformation-heavy code reads as a single expression instead of a loop with mutable state. Member functions on
collections chain naturally; a newline before `.` is treated as line continuation, so pipelines stay readable.

```go
fmt = import("fmt")

orders = [
  {customer: "Ada",   total: 120, paid: true},
  {customer: "Linus", total:  75, paid: false},
  {customer: "Grace", total: 210, paid: true},
  {customer: "Ken",   total:  95, paid: true},
]

paid_total = orders
  .filter(o => o.paid)
  .map(o => o.total)
  .sum()

vips = orders
  .filter(o => o.total >= 100)
  .map(o => o.customer)
  .sort()

fmt.println(f"paid total: {paid_total}")
fmt.println(f"vips: {vips}")
```

## F-strings and the format mini-language

F-strings interpolate any expression and accept a rich format spec parsed at compile time. The format spec itself can
contain interpolated expressions (nested `{...}`), so widths and precisions can be computed.

```go
fmt = import("fmt")

name  = "Kavun"
qty   = 7
price = 1234.5
width = 12

fmt.println(f"hello, {name.upper()}!")
fmt.println(f"qty   = {qty:05d}")                  // qty   = 00007
fmt.println(f"price = {price:>{width},.2f}")       // width comes from a variable
fmt.println(f"row   = {name:<10}{qty:>5d}{price:>10,.2f}")
```

## Exact decimal arithmetic for money

`decimal` is a first-class numeric type with its own literal syntax (`1.23d`) and exact arithmetic. Mixed expressions
promote to decimal when any operand is decimal, so a price calculation never silently loses pennies.

```go
fmt = import("fmt")

cart = [
  {sku: "A", price: 19.99d, qty: 3},
  {sku: "B", price:  4.50d, qty: 2},
  {sku: "C", price: 12.34d, qty: 1},
]

subtotal = cart
  .map(line => line.price * line.qty)
  .reduce(0d, (acc, x) => acc + x)

tax_rate = 0.07d
tax      = subtotal * tax_rate
total    = subtotal + tax

fmt.println(f"subtotal: {subtotal}")
fmt.println(f"tax     : {tax}")
fmt.println(f"total   : {total}")
```

## Records, dicts, and grouped aggregation

Records (`{key: value}`) are structures with identifier keys and dot access. Dicts (`dict()`) are dynamic maps with
arbitrary keys and index access. Combining `reduce` with a dict turns group-by aggregation into a single expression.

```go
fmt = import("fmt")

events = [
  {user: "ada",   weight: 1},
  {user: "ada",   weight: 2},
  {user: "linus", weight: 1},
  {user: "ada",   weight: 1},
  {user: "linus", weight: 5},
  {user: "grace", weight: 3},
]

by_user = events.reduce({}, (acc, e) => {
  prev = acc[e.user]
  acc[e.user] = is_undefined(prev) ? e.weight : prev + e.weight
  return acc
})

// Iterating `for k, v in dict` exposes both key and value in one statement.
// Convert dict entries to an array of records, then chain pure transforms.
ranked = []
for u, score in by_user {
  ranked = append(ranked, {user: u, score: score})
}

// Sort descending by score: encode as a sortable string, sort, reverse, unpack.
sorted = ranked
  .map(r => f"{r.score:010d}|{r.user}")
  .sort()
  .reverse()

for rank, row in sorted {
  parts = row.split("|")
  fmt.println(f"{rank + 1}. {parts[1]:<8} {int(parts[0]):>3d}")
}
```

## Safe access through `undefined`

Field, index, and call chains short-circuit through `undefined` instead of raising. Combined with the ternary operator
and `is_undefined`, defensive code against partial data stays compact.

```go
fmt = import("fmt")

response = {
  user: {
    name: "Grace",
    address: {city: "Kherson"},
  },
}

// Direct chain — every step is defined.
fmt.println(response.user.address.city)        // Kherson

// Missing intermediate — chain returns `undefined`, no error.
fmt.println(response.account.billing.zip)      // undefined

// Pick a default with a ternary.
zip = response.account.billing.zip
zip = is_undefined(zip) ? "00000" : zip
fmt.println(f"zip: {zip}")
```

## Closures and first-class functions

Functions are values and capture surrounding variables by reference. Returning a record of closures is an idiomatic way
to model a small object with private state.

```go
fmt = import("fmt")

make_counter = func(start) {
  n = start
  return {
    next:  func()  { n += 1; return n },
    reset: func(v) { n = v },
    value: func()  { return n },
  }
}

c = make_counter(10)
c.next()
c.next()
fmt.println(c.value())   // 12
c.reset(0)
fmt.println(c.next())    // 1

// Functions are values — pass them around.
apply = (fn, xs) => xs.map(fn)
fmt.println(apply(x => x * x, [1, 2, 3, 4]))   // [1, 4, 9, 16]
```

## Modules with explicit exports

A `.kvn` file becomes a module by calling `export` exactly once. The exported value is automatically made immutable, so
importers cannot mutate shared state by accident.

```go
// geometry.kvn
pi = 3.14159265358979d

export {
  pi:   pi,
  area: r => pi * r * r,
  circ: r => 2 * pi * r,
}
```

```go
// main.kvn
fmt = import("fmt")
geo = import("geometry")

radii = [1, 2, 5, 10]
table = radii.map(r => ({r: r, area: geo.area(r), circ: geo.circ(r)}))

for row in table {
  fmt.println(f"r={row.r:>3d}  area={row.area:>10.3f}  circ={row.circ:>10.3f}")
}
```

A module can also export a single function directly, making the import callable:

```go
// double.kvn
export func(x) { return x * 2 }
```

```go
double = import("double")
double(21)   // 42
```

## Errors as values

`error(payload)` returns a first-class error value. A function may return either a normal result or an error, and the
caller branches with `is_error`. The payload can be any value — including a record with structured details.

```go
fmt = import("fmt")

parse_port = func(s) {
  n = int(s)
  if is_undefined(n)         { return error({code: "bad_format",   input: s}) }
  if n < 1 || n > 65535      { return error({code: "out_of_range", value: n}) }
  return n
}

inputs = ["8080", "70000", "abc", "443"]

for raw in inputs {
  r = parse_port(raw)
  if is_error(r) {
    info = r.value()
    fmt.println(f"reject {raw}: {info.code}")
  } else {
    fmt.println(f"accept {raw} -> {r}")
  }
}
```

## Defer, recover, and named results

For Go-style cleanup and recoverable runtime errors, Kavun supports an optional named return value and `defer` /
`recover()` bound to function frames. A deferred function can read or modify the named result and call `recover()` to
catch a raised error (whether produced by the VM or via the `raise` builtin). Critical errors (stack overflow,
allocation limits) are not recoverable.

```go
safe_div := func(a, b) result {
    defer func() {
        e := recover()
        if e != undefined {
            // e.is_runtime() is true, e.kind() is "division_by_zero"
            result = error({op: "safe_div", reason: e.kind()})
        }
    }()
    result = a / b
}
```

Full example: [`docs/examples/defer-recover.kvn`](examples/defer-recover.kvn).

## Variadic parameters, spread, and membership

Variadic parameters collect extra arguments into an array; the same `...` syntax spreads an array back into a call site.
Membership checks (`in`, `not in`) work uniformly on strings, arrays, records, dicts, and ranges.

```go
fmt = import("fmt")

stats = func(label, ...xs) {
  return {
    label: label,
    n:     len(xs),
    min:   xs.min(),
    max:   xs.max(),
    avg:   xs.avg(),
  }
}

samples = [3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5]
fmt.println(stats("primes", 2, 3, 5, 7, 11))
fmt.println(stats("samples", samples...))

// Membership and slicing as expressions.
allowed = ["GET", "POST", "PUT"]
method  = "DELETE"
fmt.println(method in allowed)              // false
fmt.println("err" in "alert: error 42")     // true
fmt.println(samples[::-1])                  // reversed copy
fmt.println(samples[1:8:2])                 // strided slice
```

## Init-scoped `if` and block-local `:=`

Both `if` and `for` accept an init statement. Inside that init, `:=` declares a variable confined to the block, while
plain `=` modifies the surrounding scope. This makes "compute, then test" patterns local without polluting the outer
namespace.

```go
fmt = import("fmt")

classify = func(s) {
  // `n` is local to this if/else chain.
  // The init `n := int(s)` returns `undefined` when `s` is not a valid number.
  if n := int(s); is_undefined(n) {
    return f"{s}: not a number"
  } else if n == 0 {
    return f"{s}: zero"
  } else if n % 2 == 0 {
    return f"{s}: even ({n})"
  } else {
    return f"{s}: odd ({n})"
  }
}

for s in ["7", "12", "0", "abc", "-3"] {
  fmt.println(classify(s))
}

// `n` is not visible here — it lived only inside the if/else.
fmt.println(is_undefined(type_name))   // false; outer scope intact
```

# Purity Contract

This document defines the purity contract that operators, methods, and type-descriptor hooks in Kavun must obey. It
is the source of truth used by the AST optimizer to decide which subexpressions may be evaluated at compile time.

The contract is enforced mostly by convention and code review. There is no per-operator purity metadata; the
categories below are properties of the descriptor field a function is bound to, not of the specific token it
dispatches on. When a new hook function is added, its author is responsible for ensuring it obeys the category rule.

`MethodCall` is the one exception: `ValueTypeDescr.IsMethodPure(name string) bool` gives per-*method-name* purity
metadata within a type, because unlike operators, purity genuinely varies by method name within a single type (e.g.
`time.hour()` is pure but `time.local()` reads ambient process state; `record`'s method dispatch redirects to an
arbitrary stored callable of unknown purity). See "Method purity" below.

## Definitions

A function is **pure** in Kavun when all of the following hold:

- It does not read any external state (wall-clock time, random, environment variables, filesystem, network,
  imports, module globals, static registries).
- It does not write to any state observable outside itself (its receiver, its arguments, any captured or shared
  container, or any of the above external sinks).
- Given the same inputs (receiver + arguments), it always returns an equal result (or the same runtime error).

A **runtime error** raised by a pure function does not make it impure. A pure function may raise `division by zero`,
`index out of bounds`, `decimal overflow`, `invalid conversions`, etc. — the error is a deterministic function of the
inputs.

Iterator advancement is a special case: advancing an iterator mutates the iterator's internal cursor. This is a
localized, encapsulated write and is documented explicitly below rather than being treated as a general impurity.

## Categories

### 1. Always pure (foldable by the optimizer)

The following `ValueTypeDescr` hooks must be pure for every registered type. Adding a type whose implementation
breaks the contract is a bug.

| Hook | Notes |
| --- | --- |
| `UnaryOp` | All unary operators. |
| `BinaryOp` | All binary operators, including `==`, `!=`, `<`, `<=`, `>`, `>=`, `in`, `not in`. |
| `Access` | Read-only index or field access (`a[i]`, `r.k`). |
| `Slice` | Two-part slice (`a[i:j]`). |
| `SliceStep` | Three-part slice (`a[i:j:k]`). |
| `Contains` | `in` / `not in` container test. |
| `Len` | Container length. |
| `Equal` | Value equality. |
| `IsTrue` | Truthiness test. |
| `IsIterable`, `IsCallable`, `IsVariadic`, `Arity` | Value-shape predicates. |
| `Name`, `String`, `Format`, `Interface` | Textual / host projections. |
| `EncodeJSON`, `EncodeBinary` | Serialization. |
| `Clone` | Deep copy. |
| `Iterator` | Constructs a fresh iterator. Iterator advancement is a separate hook — see below. |
| `As*` | Value conversions. |

Operators additionally must not mutate their receivers or arguments. When constructing a new value, prefer
`slices.Concat` or an explicit `make + copy` over `append(receiver, ...)`, because `append` will silently write into
the receiver's backing storage when spare capacity exists.

`MethodCall` is deliberately absent from this table: unlike every hook above, it is not pure for every registered
type by contract — its purity varies per method name within a type. See the "Method-dependent" category below.

### 2. Always impure (never folded)

| Hook | Reason |
| --- | --- |
| `Assign` | Writes into the receiver (`a[i] = v`, `r.k = v`). |
| `Delete` | Removes an entry from the receiver (dict). |
| `DecodeBinary` | Writes into a `*Value` target. |

### 3. Localized state (documented exception)

| Hook | Reason |
| --- | --- |
| `Next`, `Key`, `Value` (on iterator values only) | Advance and read the iterator's cursor. The iterator itself is expected to be held by a single consumer for the duration of the iteration; the optimizer never speculatively evaluates iterator advancement. |

### 4. Go-style: may share backing storage with the receiver

| Hook | Rule |
| --- | --- |
| `Append` | Returns a value; may or may not reuse the receiver's backing storage, mirroring Go's `append`. Callers are expected to overwrite the receiver: `x = append(x, ...)`. Not required to be pure. The optimizer treats `Append` as non-foldable. |

### 5. Callable-dependent

| Hook | Rule |
| --- | --- |
| `Call` | Pure iff the callable is (a) `*BuiltinFunction` with `Pure == true`, or (b) a `*CompiledFunction` whose body has been proven pure by an interprocedural purity pass. Never pure for arbitrary closures. **Currently, only (a) is implemented** — the optimizer never folds a call to a user-defined function; (b) is the contract such a pass would need to satisfy if one is added (see `compiler/optimizer.go`'s `foldConstantSubexpressions` doc comment). |

### 6. Method-dependent

| Hook | Rule |
| --- | --- |
| `MethodCall` | Not one fixed category — purity varies per method name within a type. A call is a folding candidate only when `ValueTypeDescr.IsMethodPure(name)` returns `true` for the method being invoked **and** every argument independently satisfies `isFoldableExpr`. See "Method purity" below for the full mechanics, including the higher-order caveat. |

## Method purity

`MethodCall` is a single hook per type, so it tells the optimizer nothing on its own about which specific method is
being invoked or what was passed to it. Two independent checks gate folding a method call, both of which must pass:
`IsMethodPure` decides whether *this method, on this type* is eligible at all; argument foldability (covered by the
higher-order caveat below) decides whether *this particular call* is safe given what was actually passed to it.

### `IsMethodPure`: the per-method-name gate

`ValueTypeDescr.IsMethodPure(name string) bool` reports whether calling the named method on this type is safe for
the optimizer to fold:

- **Signature is name-only, no `Value` or args.** Purity of a method is a property of the *type*, not of any
  particular receiver instance or call site — the same method name is pure (or not) for every value of that type.
- **Conservative by construction.** `DefaultValueType.IsMethodPure` always returns `false`. A type — built-in or
  registered via `SetValueType` — that doesn't explicitly override it is treated as "unknown, don't fold." This
  means adding a new type, or a new method to an existing type, never silently becomes foldable; it must opt in.
- **Only consulted when the receiver's type is already statically known** — i.e. the receiver AST node is already a
  literal (either written directly in source, or already folded to one earlier in the same bottom-up optimizer
  pass — see `isFoldableExpr`'s `MethodCall` case in `compiler/optimizer_impl.go`). If the receiver isn't (yet) a
  literal, its type is unknown and the method call is never a folding candidate, regardless of `IsMethodPure`.
- **Current overrides:** every scalar type with no `_in_place` method (`bool`, `int`, `float`, `decimal`, `string`,
  `rune`, `byte`, `undefined`) returns `true` unconditionally — this includes `string`'s higher-order methods
  (`filter`, `count`, `for_each`, `find`, `all`, `any`); see the caveat below for why that alone doesn't make a call
  with a callback argument foldable. `time` returns `true` for everything except `"local"` — `time.local()` reads
  `time.Local`, the *compiling process's* ambient timezone (sourced from the OS/`TZ` environment at process start),
  not a property of its receiver; folding it would bake the compile-time environment's answer into the bytecode
  forever instead of re-evaluating against the run-time environment on every execution. This is why `int`'s `AsTime`
  (and every other `AsTime` conversion in this codebase) normalizes to UTC rather than relying on Go's `time.Unix`,
  which defaults to `time.Local` — `time.local()` is meant to be the *only* place ambient-timezone-dependence can
  enter a Kavun program.
- `array` and `dict` do not override `IsMethodPure` at all, so it defaults to `false` for every method on both types
  — `filter`/`map`/`reduce`/`for_each`/`all`/`any`/`find`/`count` included. In practice this is redundant with the
  receiver check above: an `array`/`dict` composite literal's `IsScalarLiteral()` always returns `false` (see
  `ast/expression/composite/array.go`), so `isFoldableExpr` already rejects the receiver before `IsMethodPure` would
  even be consulted. `record` also returns `false` unconditionally, for a different reason: its `MethodCall` doesn't
  dispatch a fixed method set at all — it looks up the method name as a record *key* and calls whatever value is
  stored there, which may be an arbitrary closure of unknown purity (currently moot in practice, since records have
  no AST literal syntax either, and `safeValueToLiteral` doesn't materialize `Record` values as literals).

### The higher-order caveat: function-valued arguments

Some methods that `IsMethodPure` marks pure are higher-order — `string.filter`, `.count`, `.for_each`, `.find`,
`.all`, and `.any` all accept a callback. (`array`/`dict` have the analogous `filter`/`map`/`reduce`/`for_each`/
`all`/`any`/`find`/`count` methods, but as noted above those never reach this point today: their receivers are never
foldable literals in the first place.)

A method's own logic — iteration, predicate application, result construction — being pure doesn't make the call as a
whole safe to fold if a function-valued argument's own behavior is unknown. The correct rule, in principle, is:

> A method call is pure iff every function-valued argument passed to it is also pure.

## Escape hatch: `_in_place` methods

If a mutating method is ever required on a container type, it MUST be exposed under a name ending in `_in_place`
(see [Conventions](conventions.md#mutating-vs-non-mutating-methods)). Such methods are covered by the impure
category above and are never folded.

Prefer, in order:

1. Adding the operation as a pure method that returns a new value.
2. Exposing the impure operation as a top-level builtin function (registered with `Pure = false`).
3. Only as a last resort, adding an `_in_place` method on the type descriptor.

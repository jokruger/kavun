# Purity Contract

This document defines the purity contract that operators, methods, and type-descriptor hooks in Kavun must obey. It
is the source of truth used by the AST optimizer to decide which subexpressions may be evaluated at compile time.

The contract is enforced by convention and code review. There is no per-operator or per-method purity metadata; the
categories below are properties of the descriptor field a function is bound to, not of the specific token or method
name it dispatches on. When a new hook function is added, its author is responsible for ensuring it obeys the
category rule.

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
| `MethodCall` | See method-purity subsection below for the higher-order caveat. |
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
| `Call` | Pure iff the callable is (a) `*BuiltinFunction` with `Pure == true`, or (b) a `*CompiledFunction` whose body has been proven pure by the interprocedural pass. Never pure for arbitrary closures. |

## Method purity — the higher-order rule

Methods obey the general rule: they must be pure with respect to their receiver and to external state. The only
subtlety concerns higher-order methods like `array.filter`, `array.map`, `array.reduce`, `array.for_each`,
`array.all`, `array.any`, `array.find`, `array.count`, and the analogous `dict.*` methods.

The method's own logic — iteration, accumulation, result construction — is always pure. Any impurity observed by a
call site comes exclusively from the user-supplied function argument. Consequently:

> **A method call is pure iff every function-valued argument passed to it is pure.**

## Escape hatch: `_in_place` methods

If a mutating method is ever required on a container type, it MUST be exposed under a name ending in `_in_place`
(see [Conventions](conventions.md#mutating-vs-non-mutating-methods)). Such methods are covered by the impure
category above and are never folded.

Prefer, in order:

1. Adding the operation as a pure method that returns a new value.
2. Exposing the impure operation as a top-level builtin function (registered with `Pure = false`).
3. Only as a last resort, adding an `_in_place` method on the type descriptor.

## Consequences for the optimizer

The AST optimizer's constant-folding pass (`foldConstantSubexpressions`, see
[compiler/optimizer.go](../compiler/optimizer.go)) uses this contract as follows:

- Any subtree whose root is a pure hook and whose operands recursively satisfy the same is a candidate for
  speculative evaluation via `tryEvaluateConstant`.
- The following AST node kinds correspond to pure hooks and are candidates when their children are: `UnaryExpr`,
  `BinaryExpr`, `IndexExpr` in read position, slice expressions, `MethodCallExpr` (subject to the higher-order
  rule), literal container/scalar constructors, and calls to `*BuiltinFunction` values with `Pure == true`.
- The following are never candidates: assignment targets, `Delete`-like operations, iterator advancement,
  `Append`, `Call` where the callee is not a proven-pure builtin or compiled function, and any expression whose
  evaluation would observe external state.

## Enforcement

There is no automated purity check. Adherence is enforced by:

- Short purity comments on every hook implementation (`// PURE:`, `// IMPURE:`, `// GO-STYLE:`, `// CALLABLE-DEPENDENT:`).
- Code review of any new type registered via `SetValueType` or any change to an existing hook.
- The optimizer test suite, which would surface behavioral divergence between folded and unfolded results.

# TODO list for Kavun - these are just notes, not necessarily a roadmap or priority list

- uuid type
- use json/v2
- try use new simd package

- multi-index select/remove/etc

- range of runes, range of decimals, range of times, etc + array of decimals, array of times, array of ints, etc => range_T is lazy version of vec_T / array_T / Ts

- analyze what are the most commonly mentioned problems in Python, JS, Lua, etc - ensure Kavun doesn't have them, or has a clear design for them

- **[Pick up right after the current safety redesign finishes] Design and implement a consistent error-handling
  policy across the whole runtime.** This needs its own dedicated design session first, then likely its own
  branch — it's real language-design work in the same spirit as the value-semantics/mutation redesign, but a
  separate concern from it, and too large to fold into that effort's remaining scope.

  **What the design session needs to produce:** a stated, consistent rule set for deciding, for any given
  failure a builtin/constructor/operator can hit, which of four treatments it gets:
  1. **Return an `error` value directly** (`is_error(v)` check, no control-flow event at all) — for failures a
     caller very likely wants to branch on immediately without `defer`/`recover()` ceremony.
  2. **Raise a recoverable ("logical") error** — an ordinary script-level mistake, catchable via `defer` +
     `recover()`.
  3. **Raise a fatal ("critical") error** — a genuine VM/host invariant violation or resource exhaustion,
     deliberately never catchable, always stops the VM.
  4. **A bare Go `panic()`** — if this should ever be legitimate for a script-triggerable path at all, or
     whether every such path must always go through the structured error system below instead (this needs an
     explicit answer, not just consistent practice).

  **What already exists — a real, working mechanism, not a blank slate:** `errs.Error` (`errs/errors.go`)
  already carries a `Kind` string tag plus a `Recoverable bool`, with a stated policy in its package doc comment
  ("Recoverable: ordinary script-level mistakes... Fatal: VM/host invariant violations...").
  `errs.IsCritical(err)` is what `vm/unwind.go` actually checks at every unwind point to decide whether an
  in-flight error can reach a `recover()` call at all. `docs/language.md`'s "Errors and recovery" section
  documents this to script authors as "Logical" (recoverable) vs "Critical" (not recoverable). There's also
  real precedent for the *audit* half of this work: a prior fix recategorized `format()`, `chunk()`, and
  `text.substring` from wrongly-fatal `internal:` errors to recoverable kinds for ordinary bad user input, while
  correctly leaving true VM/bytecode invariant violations fatal. What's missing is (a) the stated *rule set*
  above — that fix was ad hoc, driven by a specific bug report, not derived from a documented policy — and (b) a
  full systematic sweep applying that rule set everywhere, not just the handful of spots found by accident.

  **Concrete scope once the rules exist:**
  - Audit every error-producing site in `vm/`, `core/`, `stdlib/` against the new rule set: confirm it uses
    `errs.Error` at all (not a raw, unclassified Go error or panic), and that its `Kind`/`Recoverable` value is
    actually correct per the rules, not just whatever its original author happened to pick.
  - Grep for any bare Go `panic()` reachable from script execution that bypasses `errs.Error`/`IsCritical`
    entirely — those would escape the whole classification system and likely crash the host process instead of
    stopping the VM cleanly; each one needs to become either a fatal `errs.Error` or is confirmed as truly
    Go-internal/unreachable from a script.
  - Resolve whether type constructors (`int(x)`, `decimal(x)`, ...) should keep raising on bad input or should
    return an `error` value instead so calling script code can react without `defer`/`recover()` — this was
    flagged as a footgun before `recover()` existed; decide whether `recover()` already resolves it or whether
    ctors specifically warrant category 1 above.
  - Resolve why conversion methods (`.byte()`, `.string()`, `.decimal()`, etc.) can silently produce a
    wrong/degenerate result on invalid input instead of signaling failure at all (today: neither raise nor
    return, a third failure mode the rule set above needs to cover) — decide the exact current failure mode per
    method first (silent truncation? zero-value? something else?), then which of the four categories it should
    become.
  - Every conversion/constructor/builtin function's actual behavior should be verified empirically against the
    new rules (compile and run real scripts), not just reasoned about from reading the Go source.

- **Review NaN behavior in `decimal` arithmetic.** Deliberately deferred — not decided. In `dec128` NaN is an
  **error state**, not a value, and it is reachable from many arithmetic paths, not just from a failed parse
  (all verified 2026-08-22): `decimal("1") / decimal("0")` → NaN, `decimal("0") / decimal("0")` → NaN,
  `decimal("-1").sqrt()` → NaN, `decimal("1e999999")` → NaN (overflow), `decimal("inf")` → NaN. It then
  **propagates silently** (`decimal("abc") + decimal("1")` → NaN) and, unlike IEEE NaN, compares **equal to
  itself** (`decimal("abc") == decimal("abc")` → `true`), so it behaves as an ordinary value everywhere
  downstream.

  The inconsistency to resolve: `int` division by zero already **raises** (`1/0` → error), `float`'s NaN/`+Inf`
  are legitimate IEEE values produced by real arithmetic, and only `decimal` uses an in-band sentinel for a
  genuine error. The conversion side is being fixed separately — constructors, conversions and parses will raise
  on a NaN result unless a default is supplied — but that fixes only the *construction* path. If the arithmetic
  path keeps returning NaN, the inconsistency has been moved rather than removed.

  Options to weigh: (a) arithmetic raises on a NaN result, matching `int`'s division-by-zero; (b) arithmetic
  keeps NaN and Kavun documents it as decimal's error value, with `is_nan()` the mandated check; (c) a
  `decimal`-specific "checked" vs "unchecked" split. Interacts with the operator design (operators are otherwise
  settled) and with the error-handling policy item above — decide it together with that, not in isolation.

  Related bug found while investigating: `decimal.error_details()` on a **valid** (non-NaN) decimal panics the Go
  host — `core/decimal.go:473` calls `o.ErrorDetails().Error()` where `ErrorDetails()` returns `nil`. Needs a nil
  check plus a regression test.

- **`dec128`: implement scientific-notation parsing** (upstream, `github.com/jokruger/dec128`). Verified
  2026-08-23: `dec128` does not parse exponent notation **at all**, so a value well inside its range is
  unreachable through the string path.

  | expression | result | note |
  | --- | --- | --- |
  | `decimal("1e10")` · `decimal("1E10")` · `decimal("1.5e3")` · `decimal("1e-5")` | `NaN` | notation unsupported |
  | `decimal("10000000000")` | `10000000000` | the same value, plain digits — fine |
  | `(1e10).decimal()` | `10000000000` | the float route bypasses the string parser entirely |
  | `1e10d` · `1.5e3d` · `1e-5d` | `Parse Error: invalid decimal literal` | the literal path has the same gap |

  So there are three spellings of one value and only two work, and the two failure modes disagree: the literal
  path fails **loudly** at parse time, the string path returns `NaN` **silently**.

  Distinct from the real range ceiling, which is the 128-bit coefficient and is working correctly:
  `decimal("340282366920938463463374607431768211455")` (2^128−1) parses, 2^128 → `NaN`, `(1e38).decimal()`
  parses, `(1e39).decimal()` → `NaN`. Both are in scope for the same fix only in the sense that both must end up
  raising rather than yielding `NaN` — see the error-handling item above; the settled conversion design makes
  constructors, conversions and parses raise on a NaN result (with the optional `[default]` as the escape).

  Fix upstream (accept `[eE][+-]?digits` in the parser, scaling into the coefficient and rejecting a result that
  overflows 128 bits), then accept the exponent form in the scanner's decimal literal (`parser/scanner.go:519`)
  so all three spellings agree. Needs tests for `1e10d`, `1.5e3d`, `1e-5d`, round-tripping through
  `.string()`, and for the overflow boundary at 2^128.

- functions contracts - a guarantees on inputs/outputs (types, checks, etc)

- compiler-enforced `_in_place` function contract: a user-defined Kavun function may mutate an argument's shared
  body only if its own name is `_in_place`-suffixed — the naming policy is already documented as a hard
  requirement (docs/conventions.md, "Mutating vs Non-mutating Functions"), but nothing checks it today. Needs a
  sound alias-tracing static analysis: track every local alias of each parameter within a function body;
  propagate the check interprocedurally across calls to other user-defined functions (handling forward
  references and recursion without unbounded fixpoint blowup); treat any call through a runtime-supplied
  callback parameter as unprovable/mutating by default; decide how far to trace through branches/loops/early
  returns without either missing a real mutation or rejecting ordinary safe functions so often that the default
  becomes unusable. Once designed, wire it into a host-configurable compile-time check mirroring
  `Script.SetAssignmentMode`, defaulting to on, with an unprovable function being a compile error (a false "safe"
  verdict is worse than none).

- do we actually need a copy_shallow for scalars? it makes no sense and only confuses?

- review the type conversion system: whole matrix of conversion, pair by pair, especially string related conversions (pay attention to symmetry)

- **`int_range` arithmetic** — deliberately deferred out of the type/operator redesign (see
  `docs/types.md`'s "Operators across types" section and `docs/extending-types.md` for what that
  redesign did land), not decided. Today `int_range` gets no operators at all (vm error against
  everything, same as any other undecided pairing). Needs its own pass: does `int_range ± int`
  make sense (an earlier draft considered materializing into a concrete, element-wise-shifted
  `array`, since `int_range` is a lazy fixed-bounds view and can't stay lazy under genuine
  per-element arithmetic); if so, is it symmetric (`int + int_range`) the way `byte`'s `±` is, or
  one-directional the way `rune - int`/`int - rune` is; whether `-` makes sense at all for a range;
  and whether materializing to `array` is even the right result shape. Should be designed together
  with, or cross-checked against, the vector/container broadcasting backlog elsewhere in this file,
  since both concern "arithmetic across a bunch of elements at once."

- **`bool` arithmetic scope** — deliberately deferred out of the type/operator redesign, not
  rejected. `bool` currently has no arithmetic at all, including with itself (`true + 1`,
  `true + true` are both errors) — that redesign only gave it same-type ordering (`false < true`)
  and unary `^` (logical negation); the comparison redesign (see `docs/types.md`'s "Operators
  across types") later extended `bool`'s ordering (and `==`/`!=`) across the rest of the numeric
  family too (`true < 2`, `true == 1`), but arithmetic is still untouched and still deliberately
  undecided. Whether `bool` should widen into `int` the way `byte` eventually might, or stay
  arithmetic-free permanently, is real future design work, not decided either way yet.

- **`dict`/`record` containment check** — not decided, not started. `dict`/`record` support
  structural `==`/`!=` across each other (see `docs/types/dict.md`'s "Equality and ordering") but
  deliberately have no ordering (`< > <= >=`) at all — a dict has no natural total order the way a
  `string`/`array` does, and Python 3 removing dict ordering (Python 2 had one; it was considered a
  mistake) is real precedent against inventing one here. The one ordering-*shaped* question that
  would have principled meaning is subset/superset ("does every key/value in `a` also appear in
  `b`?"), the same relationship Python's `set`/`frozenset` expose via `<`/`<=`. That's a genuinely
  different, *partial* order, though (two dicts can be simply incomparable), unlike every other use
  of `<` in Kavun — so if this ever gets built, it should be an explicit method
  (`a.is_subset_of(b)`, `a.is_superset_of(b)`), not an operator overload, to avoid giving `<` two
  different kinds of meaning depending on operand type.

- Pipe operator `x |> f(_) |> g(y, _)` — `_` marks where the piped value lands.
- piping and flow (`x |> f1(_) |> f2(y, _) ...`)
  - builtin type member functions allow write nice calc pipes, but user defined functions still will require nesting
  - idea is to be able describe a pipe where prev call result is passed as an argument to next call in pipe
  - ideally when describing next function we should be able define the argument to which the prev result is passed, and define other args

- big_float, big_int and big_rational types

- builtin `merge(r1, r2)` for record, `dict.merge()` for dict.

- enumerate() → array[(index, value)] (or dict-like pairs)
- builtin `enumerate()` → array of `(index, value)`.
 
- zip(other) → array[array] of len 2; unzip ???
- builtin `zip(other)` / `unzip()`.

- builtin `window(n, step=1)` → sliding-window array of arrays.
- window(n, step=1) → array[array]

- member functions for `int`/`float`/`decimal`: `abs()`, `pow()`, `sqrt()`, `sign()`, `is_zero()`.

- member functions for `array`: `reverse()`, `unique()`, `flatten()`, `chunk(n)`, `window(n, step)`, `enumerate()`,
  `sort_by(fn)`, `intersperse(x)`, `cycle(n)`, `fill(n, val_or_fn)`, `take(n)`/`drop(n)`, `push`/`pop`/`insert`.
- container types: .reverse(), .shuffle(), .unique(), .chunk(size), .window(size, step), .enumerate()

- member functions for `array`: split `append` (new array) vs `extend` (in-place).
- array.append (array) => new array
- array.extend (array) => inplace

- member functions for `string`: `has_prefix()`, `has_suffix()`, `replace()`, `pad_left(n, ch)`/`pad_right`/`center`.

- member functions for `bytes`: `hex()`, `base64()`.

- member functions for `string`/`runes`: `quote()`/`unquote()` — string escaping/unescaping. `text.quote`/
  `text.unquote` (Go's `strconv.Quote`/`Unquote` verbatim) were deleted with the `text` module rather than moved;
  if this comes back as a member pair, the **escape grammar is the decision** (Go's? Kavun's own literal syntax?
  JSON's?) — `json.encode` and `format()` cover most display/interop uses today.

- member functions for `range`: mirror array methods — `filter`, `reduce`, `sum`, etc.
  
- member functions for `time`: `is_leap_year()`, `is_weekend()`, `is_weekday()`, `is_holiday(calendar)`.
  
- member functions for `rune`: methods mirroring Go's `unicode` package.
- rune - implement methods from <https://pkg.go.dev/unicode>

- new type `Set` type with set operations (union/intersect/diff/membership).

- Builtin `regex` type.

- `??` nullish-coalescing operator — `x ?? default`, substitutes only when `x` is `undefined` (unlike a
      truthy-based fallback, which would wrongly override an explicit `0`/`""`/`false`). New; no existing
      operator covers this today.

- go style switch with multi-value cases, default, etc
- Pattern-matching `switch`/`match` with destructuring — extends the `TODO.md` "go style switch with
      multi-value cases, default" item to also match on shape: `[a, b]` vs `[a, ...rest]`, or a record by
      field presence. Builds directly on the destructuring-assignment syntax that already exists.

- array, string, bytes - multi-index get: array[1, 3, 5], or array[x] where x is array of ints
- ... and multi-index set: array[1, 3, 5] = [10, 30, 50]
- Multi-index get/set — `a[1, 3, 5]` / `a[1, 3, 5] = [10, 30, 50]`.

- Elementwise/broadcast array ops — `a .+ b`, `a .* 2`, etc.
- vector types: bytes, ints, floats
- typed vectors, J core operators
- vector/array operations like /+, /-, /\*, etc - elementwise operations for vectors
- Typed/vector arrays (`ints`, `floats`, `bytes` vectors) as the storage backing for elementwise ops.
      Design note carried over from the R comparison: if these land, elementwise/map
      operations on a typed vector should raise on a type mismatch rather than silently downgrade to a
      heterogeneous `array` — R's `sapply` vs `purrr::map_dbl` is the cautionary example either way.
      Cross-reference: the type/operator sub-system review near the top of this file should be designed with
      these elementwise/broadcast semantics in mind (or done first) — an order-independent numeric rule is what
      makes broadcasting tractable, per that entry's own reasoning.

- Generator/`yield` sugar over the iterator protocol — a user-defined function that `yield`s values,
      desugaring to the existing iterator interface. New, but low-risk: `docs/purity.md` already carves out
      "localized state" as a purity exception for iterators, so this doesn't introduce a new impurity category.

- Function introspection properties: `arity`, `is_variadic`.
- function property "arity" and "variadic"

- Builtin memoization for pure functions. Already in `TODO.md` — pairs naturally with the purity contract
      (only safe to auto-memoize a function the optimizer already knows is pure).

- Lightweight table/dataframe type with basic joins (including as-of join for time-series).
  inspired by kdb+'s columnar tables and R's dplyr, and a real workload shape for the finance use case
  ("match trade to most recent quote"). Needs its own design pass (storage model, join semantics, interaction with
  existing `record`/`dict`) before scoping

- Enforce allowed-module list at the VM level (host can permit bytecode execution but deny specific
      modules) — directly serves the sandboxing goal.

- Stable/deterministic dict iteration order — directly serves the reproducibility goal.

- Split `rand` into two explicit modes: a seeded, deterministic PRNG (for reproducible simulation/decisioning
      — same seed, same script, same output, replayable for audit) vs. an unpredictable source (for anything
      needing real entropy, e.g. token generation). New nuance: `TODO.md`'s "migrate to crypto/rand" is solving
      the *opposite* problem (unpredictability) from what a reproducible finance/decisioning script usually
      wants (a fixed seed) — these are two different needs likely served by two different APIs, not one module.

- Allow the host to inject a deterministic clock for `times.now()` during replay/testing, so a script that
      reads wall-clock time can still be re-run byte-for-byte identically in an audit/test context. New.

- User/file-module import-result caching: today every `import(...)` call site independently re-executes the
  module's own init code — no result is cached or shared, so two `import("mod")` calls anywhere in a script get
  two independent values, never the same one. Adding caching (so repeated imports of the same module share one
  result) is real, unbuilt functionality — safe to build now specifically because `export`'s codegen already
  does a deep, in-place immutability flip (not just a shallow one) before an export is ever handed back, so a
  cached, shared result can't let one importer's mutation attempt corrupt another's view of "their own" module.
  Builtin (stdlib) modules already work this way (one process-wide shared `Value` per module); this would extend
  the same sharing model to user-authored modules.

- Record's member-call/selector-access dispatch consistency: `record_val.foo` (no call, selector access) and
  `record_val.foo(...)` (call) both currently resolve `foo` by looking it up as a record key, but through two
  separate mechanisms (the `Access` hook for the no-call form, `MethodCall`'s field-lookup-and-invoke for the
  call form) that were never explicitly checked against each other for consistency. Worth a dedicated look at
  whether these two paths agree in every case (e.g. missing key, non-callable stored value, `Access` vs.
  `MethodCall` error wording) or whether they've quietly diverged.

## Optimizations

- `copy()`'s deep-clone traversal could short-circuit at an already-immutable subtree — an immutable `Value` can
  never be mutated through any live alias, so cloning it during a deep `copy()`/`freeze()` is wasted work; the
  traversal could return the immutable node as-is (sharing it) instead of recursively cloning underneath it.
  Low-risk, purely an implementation detail — doesn't change `copy()`'s observable behavior (an immutable node
  still can't be mutated either way, cloned or shared), just avoids the allocation.

- PushFloat - use when float in script can be encoded as float32 exactly

- PushShortString / PushShortRunes / PushShortBytes. Any string literal of length ≤ 7 bytes (ASCII identifiers like "id", "name", "ok", "err", single-char separators, empty string) fits entirely in the operand. Store len in Op1 (values 0..7), 7 bytes in Op2+Op3. VM materialises a Value around an inline byte array — needs a small pool or per-frame scratch, or you accept one allocation but skip static-table indexing + the NewStaticStringValue pointer chase.

- PushEmpty* family - PushEmptyString, PushEmptyBytes, PushEmptyRunes, PushEmptyArray, PushEmptyRecord, PushEmptyDict — one-instruction, no static, no allocation if you keep singleton empty-immutable values around. MakeArray 0 / MakeRecord 0 are common in generated init code.

- AccessSelectorConst - cases like "x.name", "x.id", etc - only one selector used, it is a static string, so we can encode selector id as operand! Similar for StoreIndexedLocal/Free/Global

- AccessIndexConst - cases like "x[0]", "x[1]", etc - only one index used, it is a static int which fits in operand!  Similar for StoreIndexed*

- Separate instruction to load local/free/global without de-referencing ValuePtr - compiler should know that variable was never captured by closure, so no need to check and dereference ValuePtr - use separate instruction for this case. Same for Store.

- LoadLocalReturn - combine LoadLocal + Return into one instruction (very frequent pattern)

- AbortCheckJump — every for/for-in compiles as ... ABORT_CHECK; JUMP preCond. Merge them: Op3 = target IP, and the abort check is done inline. Removes one dispatch per loop iteration.

- JumpBack — a dedicated “loop back-edge” op that both jumps and polls abort. Frees the compiler from needing a separate AbortCheck.

- LoadLocalJumpFalsy — the AST-level if x { ... } and for x < n { ... } shapes end up as LOAD_LOCAL x; JUMP_FALSY end. Fuse: Op2 = local idx (as byte width is enough for most), Op3 = target IP. One instruction, no stack round-trip.

- IntBinaryOpJumpFalsy (or IntCmpJump) — the loop condition pattern. for i < n becomes LOAD_LOCAL i; LOAD_LOCAL n; BINARY_OP Less; JUMP_FALSY end. A fused INT_LT_LOCAL_LOCAL_JUMP_FALSY (Op1 = op token, Op2 = local1, Op3 = target IP; second local packed into unused bits) reduces the hottest four instructions of every counting loop into ONE.

- IterNextJumpFalsy — every for-in is LOAD_LOCAL :it; ITER_NEXT; JUMP_FALSY end. Fold to a single ITER_NEXT_JUMP_FALSY: Op2 = local idx of iterator, Op3 = target IP. Halves the per-iteration dispatch cost of every for-in.

- IncrementLocal / DecrementLocal — i++, i--, i += 1. Encode Op2 = local idx, Op3 = signed 32-bit delta. Skips two loads, a binop, and a store.

- IncrementLocalJumpFalsy
 
- DefineLocalPushInt / DefineLocalPushConst — very common in loop init (for i := 0).

- LoadLocalCallMethod / LoadStaticStringCallMethod — chain of method calls. x.method(...) starts with LOAD_LOCAL x; CALL_METHOD "m", nargs. Fuse the local-load.

- LoadLocalPtrCallFunction for closure calls where the callee is a captured free variable.

- LoadBuiltinFunctionCallFunction — for direct builtin calls like len(x), append(a, b). Combined dispatch: fetch the builtin ID from Op3, arg count from Op2, do the call in one step. Skips the type-check branch in CallFunction entirely (we know it’s builtin at compile time).

- LoadStaticCompiledFunctionCallFunction — direct top-level function call. Even bigger win: skip the callable-type dispatch and the switch val.Type, jump straight into the CompiledFunction call path.

- LoadLocalReturn — return x. Extremely common. Combined instruction dodges the extra LoadLocal and any ValuePtr dereference bookkeeping.

- PushIntReturn / PushBoolReturn / PushUndefinedReturn — small constants as return values.

- ReturnConst<T> — even better, return true, return false, return 0, return nil become one instruction.

- Iterator hot-loop mega-op
    - The full for-in prelude looks like: LOAD_LOCAL :it; ITER_NEXT; JUMP_FALSY end; LOAD_LOCAL :it; ITER_VALUE; DEFINE_LOCAL v; ...body. You could introduce a ITER_STEP super-op:
    - Op1 = which parts to extract (key|value|both)
    - Op2 = iterator local index
    - Op3 = target IP (jump on exhausted)
    - Body starts with the key/value already on the stack (or already stored into predefined local slots via extra bits).
    - That fuses five instructions of every for-in iteration into one.

- LoadLocalFormatStaticSpec: f-string emission sequence is typically LOAD_LOCAL x; LOAD_STATIC_FORMAT_SPEC i; FORMAT_STATIC_SPEC. Fuse to LoadLocalFormatStaticSpec: Op2 = local idx, Op3 = spec idx.

- PlusInt, MinusInt for int32
- Load[Static/Local/Global/etc]BinaryOpInt for int16
- Load[]IncStore, and Load[]DecStore

- static analyzer:
  - check all opcodes are valid
  - check all jumps are valid (address is within bytecode)
- No need to check if opcode is valid in VM - it is already checked by static analyzer
- Use unsafe for vm.ip so no bounds check on each opcode fetch

- composite opcodes - some common structures/patterns (loops, calls, assign-inc, etc) are implemented as multiple opcodes - we can implement them as single opcode

- add "reuse" flag to hooks which return value

- SeqIterNextHook, SeqIterKeyHook, etc, and any generics receiving resolve callback can be changed to generic type Target and (*Target)(v.Ptr) directly!

- hooks which return value - accept flag indication that current value can be reused (so we can avoid some allocation) - in future compiler can detect when it can use this!

- `x = x.method(...)` → `x.method_in_place(...)`/`method_view(...)` rewrite (the reuse-flag idea above, applied
  to every safe-default/`_in_place`/`_view` twin pair — `append`/`append_in_place`, `delete`/`delete_in_place`,
  `splice`/`splice_in_place`, `slice`/`slice_view`, `chunk`/`chunk_view`). Not just a nice-to-have: without it,
  the single most common loop idiom for building up a container —
  ```
  x := []
  for item in source {
      x = x.append(item)
  }
  ```
  pays a full copy every iteration under the new safe-by-default semantics, even though in this exact shape `x`
  never has any other live alias between iterations — nothing could observe the difference if the compiler used
  `append_in_place` instead. That's an O(n²) loop where the old always-shared model was O(n), so this matters for
  reaching performance parity with reference-style languages (Python/JS/Ruby), not just other copying designs.
  Only sound at a call site where the compiler can prove `x`'s body has no other live alias at that point (no
  other variable bound to it, no closure capture, never passed to a function that might retain it or returned) —
  Kavun's ambient container-sharing model (every value shares its body on assignment/argument-passing by design)
  means this is a real risk even without any `ref()`-like construct: `y := x; x = x.delete("k")` must never
  affect `y`, but a naive in-place rewrite would silently mutate `y`'s shared body too. Must be a **best-effort,
  soundness-gated** pass, scoped like the existing `O0`-`O3` optimizer levels (more passes/deeper analysis
  catches more cases, never a completeness claim) — never at the cost of `docs/purity.md`'s "never change
  observable behavior" bar; Kavun's VM is a single-tier bytecode interpreter with no guard/deopt fallback the way
  a JIT would have, so an unsound rewrite has no runtime safety net if the proof turns out wrong. Needs actual
  escape/uniqueness analysis, not just AST pattern-matching on the `x = x.foo(...)` shape — meaningfully beyond
  what `compiler/optimizer.go`'s current O0-O3 passes do (constant folding, copy/constant propagation, DCE; no
  alias tracking yet). Candidate approaches to pick from when this is designed: a purely static, conservative
  scope check (the rewrite only fires when, since `x`'s last rebind in the same scope, no other name was
  assigned from `x`, `x` wasn't passed as a call argument or returned, and `x` wasn't captured by a closure —
  declines instantly otherwise, no interprocedural analysis attempted); a Swift-style *runtime* uniqueness check
  (`isKnownUniquelyReferenced`-equivalent) immediately ahead of the mutating call, falling back to a copy only
  when it fails — catches more cases than the static approach but needs a uniqueness signal on `Ptr`-backed
  values that doesn't exist today; or a Clojure-style explicit `transient`/`persistent!`-shaped scoping construct
  (arguably already *is* what the `_in_place`/`_view` twins themselves are, which argues for scoping this to
  "recognize and rewrite the single-local-loop idiom specifically" rather than solving aliasing in general).

- use pool for low level slices (bytes, runes, arrays)

## Other

- ensure we write some new value to stack each time we increment it

- validate changes to stack pointer when we got error in vm (sp must always be updated same as in success case)

- check type conversion: string(["a", "b", "c"]) and ["a", "b", "c"].string()

- now primitives are easy to distinguish, so we can have fast path in equal for instance (no call to hook, just compare data)

- control allowed modules on VM level!!! required for security, so we can allow bytecode execution but disallow some modules!

- type as data + extension methods:
  - array.foo => call array static method
  - array.sum = foo => override array type method (globally)
  - array.myfoo = foo => extend array type with new method (globally)

- add to desc "written in pre Go, no CGo"

- compiler - find a way to analyze expressions and generate a code which does not require new variables on each binary op and can reuse existing.
  - we may need to change interface of hooks so instead of returning value thay will have a receiver as argument, so compiler can decide if new var is needed

- builtins are stored as a map, but max num of builtin functions is 256, so we can use array!
- check if vm limits are enforced (globals, etc)
- knowing vm limits (max nums / sizes), what can be optimized? (i.e. we could potentially use some preallocs, etc)?
- inspect all panics - return errors
- can we de-dupe constants in same time we emit them?

- need a stable dict iterations / map / tostr / etc

- add Hash function for Value (and all types). For ptr based values hash can be cached in .Data, use it in comparison

- refactor core/tools.go , looks like coerceSepToString, coerceSepToBytes, etc can be replaced with .AsString, etc?
- refactor member functions - in many cases we can have generic implementation used from concrete types

- make sure you cannot crash VM from script: limit num of allocs, total size of containers and mem used, catch panics
- for arrays, bytes, runes, strings - store data=leng and ptr=underlying data (&[0] / StringData, etc) to avoid allocation of header struct
- use store underlying array/dict pinter in Value.Ptr instead of using wrapper struct
- try use unsafe.StringData / unsafe.String to store and rebuild strings?
- do atomic load check for "abort" flag every X cycles, not every cycle
- for int/float/string/etc args, fast path for specific types, only then call .AsX()
- string - make it unicode indexed (slice, index and member function work with unicode by iterating! - note on performance in docs)
- runes.trim - custom implementation that uses runes slice from allocator

- migrate to crypto/rand
- Move strings package functions to the string type member functions
- optimization for "modify and assign" pattern (reuse variable, pass argument to inform type logic)
- fold(f, init) → value (same as reduce-with-init; pick one name)
- array.sort(lambda(a, b) => bool)
- move type related functions to type member functions; remove duplicates from stdlib (i.e. stdlib must be complimentary extension of type member functions)
- Arrays: `sort_by`
- missing ctors(0/1/2): array, record
- range methods: dict, filter, reduce, sum, etc (mirror array methods)
- generic range (just like int range but use Value for start/stop/step) - to be used for time, float, etc ranges as well
- splice - use AsArray
- move splice function to container types (methods)
- in VM slice logic, use fast path for Int
- format for decimal
- type() member function for all types, returning type name as string
- remove dict/record to string conversion - it breaks consistency... complex values should be printed, not converted to string implicitly
- add flag to `immutable` function to do a deep immutability (for arrays/dicts/records) - so all nested structures will be immutable as well

- Array.fill(n, val)`/`Array.fill(n, fn)
- array.intersperse(x)
- array.cycle(n)
- array.take(n)`/`drop(n)
- array.push/pop,insert

- string/rune/bytes/array \* int => repeat n times; need to be in sync with global vectorization strategy
- implement hashing for each data type, optimize "dedupe / unique / equal" using hash
- compile time tail call optimization - runtime vm should not be smart, just a stupid loop over switch cases, all decisions should be made at compile time

- coalesce(...) return first non-null arg

---

- find a way to reuse value envelopes: receiver ptr instead of return value, mark as tmp, on assign copy if tmp, etc - primary usecase = loops
- how to use string value or envelope ptr in map keys, so we can use them when iterating over keys (instead of creating new strings)
- builtin cron support (expressions, next event, etc)
- shell we rename fmt to io ?
- input functions - console input, key, etc
- use caches for runtime parsing, etc (use cache package with controlled cache size)

- optional static types - does not allow reassign to other types, fail function calls, etc

- builtin logging

!!! check vm.go, "case opcode.CallFunction" and "case opcode.CallMethod"
it looks like we first put spread args to the stack (and can overflow) but then
immediately reshape it to collapse the tail args into variadic (a single array arg).
It should be possible to avoid temp copying to stack !

- Performance optimizations enabled by precise MaxStack
- Now that each `CompiledFunction` knows its exact peak operand-stack height, several optimizations become possible:
1. **Per-frame stack allocation** — currently the VM has one giant shared `stack []Value`. With MaxStack known per function, each frame could carry its own slice (or use a bump allocator), improving cache locality and enabling parallel call stacks for goroutine-style features.
2. **Tighter default stack size** — the default `stackSize` heuristic could shrink for small scripts. Programs that statically never recurse can use exactly `sum(MaxStack)` for the call chain.
3. **Drop residual safety branches** — any leftover defensive stack checks in hot paths (e.g., the OpCall guard itself can become a debug-only assert in release builds, since the compiler proves the invariant).
4. **Smarter inlining** — small callees whose MaxStack + caller's current height fits without growth become candidates for bytecode-level inlining (no new frame, no OpCall overhead).
5. **Disassembler/profiler surface** — expose MaxStack and NumLocals in disassembly so users can spot deep-evaluation hotspots.
6. **Stack pre-touching / zeroing only the needed range** — `Reset()` and frame entry only need to clear `NumLocals+MaxStack` slots, not the whole stack.
7. **Specialized tiny-frame VMs** — for leaf functions with MaxStack ≤ a small N (say 4), a register-style fast dispatch could be generated.

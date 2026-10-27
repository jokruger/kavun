# Conventions

## Coding Conventions

This section outlines coding conventions for Go code, including VM specific guidelines and general best practices.

### Variadic Arguments: Immutability Contract

Functions that accept variadic arguments (`...Value`) or slice of arguments (`[]Value`) must **never mutate** the
arguments slice or its elements. This is both a Go best practice and a critical requirement for performance in this VM.

To avoid allocations, the VM passes stack slices directly to callees. The full capacity of these slices extends to the
end of the stack array. If a callee appends to `args`, it corrupts subsequent stack frames.

Functions should not have side effects on caller state beyond their explicit return values. Mutating arguments violates
this principle.

The contract also covers *keeping* the slice: a callback member fills one argument buffer per member call and
passes it to every callback (`ElemCallback`/`ReduceCallback` in `core/tools.go`), so a callee that stored `args`
— or a variadic roll-up that built its rest array over it — would see the next element's values. Copy what you
keep.

### Value model: scalars vs. containers

Every `core.Value` is a fixed-size header — `Type`, `Immutable`, a `Data` word, and a `Ptr` (`core/value.go`).
Assigning or passing a `Value` always copies this header only, never anything it may point to. For types that
store their whole payload in `Data` (`int`, `float`, `bool`, `decimal`, ...), that header copy is a full,
independent copy of the value. For types that store payload behind `Ptr` (`array`, `dict`, `record`, `bytes`,
`runes`, and also `string`), the header copy shares the same backing storage as the original — mutating through
one binding is visible through every other binding that shares that pointer.

`string` is the one type in the second group (`Ptr`-backed) that scripts still perceive as scalar, and that's
not an accident of implementation — it's what makes the user-facing model correct: `string` has no mutating
operation at all, so its shared backing bytes are never observable as shared. Whether a type is "scalar" or
"container" from a script author's point of view is determined entirely by whether it has a mutating operation,
not by whether its `Value` is `Data`-inline or `Ptr`-backed internally.

**Constraint for anyone adding a new builtin type:** if it's `Ptr`-backed (heap storage, shared on
assignment/passing — the normal case for anything non-trivial), it must never gain a mutating operation unless
it's meant to behave as a container in the user-facing model. If a new type should read as scalar to script
authors — copied, never surprising, no shared-body behavior — it must never be given an `_in_place` method or
any other body-mutating hook, regardless of whether it happens to be `Ptr`-backed internally (see `string`).

### Implementing operators (`BinaryOp`/`UnaryOp`/`Equal`)

See [Extending types: operators](extending_types.md) for the full contract — the three-rule dispatch model
(domain-specific, narrow safe-conversion, reflected delegate) for `BinaryOp`/`UnaryOp`, the simpler one-hop
delegate model for `Equal` (`final` flag, no error return — `==`/`!=` are total operators), and the one hard
cross-cutting rule shared by both (never wildcard-match `undefined`/`error`). Same weight as the Purity Contract
above: enforced by code review, not by any automated check, and required of any new builtin or embedder
(`SetValueType`) type that implements operators at all.

### Implementing members (`Methods` tables)

Readability beats deduplication here: a member is read far more often than it is added, and a script author's
question — "what exactly does `bytes.keep("ab")` do?" — should be answerable from one function. So:

- **One name, one entry.** A builtin type's members form a `Methods []Method` table keyed by `members.*` ids,
  generated into `core/kavun_gen.go` (`var arrayMethods = []Method{…}`, named after the root file's stem) and
  referenced at the bottom of its descriptor (`Methods: arrayMethods,`). Each entry names one function
  `<type><Member>(vm VM, v Value, id member.ID, args []Value) (Value, error)` in `core/<type>_members.go`, declared
  by the `//kavun:member` directive in its doc comment (see [Declaring native
  functions](#declaring-native-functions-directives-and-kavun-gen)); source order carries no meaning. Every entry
  names its own function, and two bodies share logic only through a helper they call; a body may report through
  `id.String()` (so an error names the member that was called), but never branches on which member was called.
  `record` and embedder types without a table answer by name through `CallNamedMethod`, one `case` per name under
  the same rule. Nothing is answered centrally: even `is_true` is an entry each builtin type declares
  (`//kavun:member is_true pure () -> bool` on `intIsTrue`, generated as `members.IsTrue: NewPureMethod("()",
  intIsTrue)`), and a host type has it only if it declares it too.
- **Build every entry with a constructor, never a struct literal.** `NewPureMethod(template, fn)` claims the member
  pure (the optimizer may fold it — docs/purity.md); `NewMethod(template, fn)` is the safe default, so a forgotten
  claim costs speed, not correctness. `template` is the signature template — the parameter list a script author
  would write, receiver excluded: `"()"`, `"(n[, fill])"`, `"(...xs)"` (see [Arity and
  templates](#arity-and-templates)). The generated tables call exactly these constructors; a host type writes its
  entries by hand the same way (or runs `kavun-gen`, see [extending_types.md](extending_types.md)). Properties
  likewise: `NewPureProperty(get, set)` / `NewProperty(get, set)`, the claim covering `get` (`set` is impure by
  definition, `nil` when read-only).
- **The template is the count check's floor; the body bounds its rest.** The dispatcher (`Method.Call`,
  `BuiltinFunction.Call`, `BuiltinClosure.Call` — every table-driven path, the VM's fast tracks included) checks
  the count against the entry's signature before the body runs: exactly the arity when fixed, at least the arity
  when variadic. A fixed body indexes `args[N-1]` unguarded. A body whose template names an upper bound
  (`([sep])`) or a gap (`(a[, b, c])`, `() | (text) | (bands, min, max)`) keeps that one check and refuses by
  `return Undefined, ErrArgCount`; the dispatcher turns the sentinel into the `wrong_num_arguments` error quoting
  the template (`(join) expected ([sep]), got 2 arguments`), so a body never spells its own template. A count the
  template accepts is never a count error: an argument of the wrong type for the reading that count selects is
  `invalid_argument_type`. `errs.NewWrongNumArgumentsError` is called only by `core.ArgCountError` — the VM's
  script-function paths reach it through `CompiledFunction.ArgCountError` (`TestArgCountSingleSource`); a host hook with no declaration (`CallNamedMethod`, a host
  type's `Call`) reports through `core.ArgCountError` with `core.SignatureOf`. `TestMemberArity` calls every entry
  of every table through the dispatcher with every count `0 … max+1` and fails wherever the refusals and
  `Signature.Accepts` disagree.
- **One function per member, `_in_place` twins included.** `arrayKeep` and `arrayKeepInPlace` are separate
  functions with the member shape. When twins share a body, it is a lowercase helper taking what differs as a
  parameter (`arrayTypeAppend(v, args, mutate)`, `decimalRoundFixed(v, id, args, mode)`), never the member name.
- **Concrete code per type.** Logically identical members on `array`, `string`, `runes` and `bytes` are written
  out per type, over the type's own element slice. Duplication is accepted; cross-type parity is pinned by
  tests (`TestSequenceMemberParity` in `kavun_test.go`), not by a shared engine.
- **Generics only without function parameters.** A generic helper over `comparable` elements (find a run, measure
  a match) is fine; a generic that has to be configured with callbacks (`toElem`, `encode`, `alloc`, `resolve`,
  ...) is an engine, and is not.
- **Sharing is inside the body, in two forms by size.** A body too short to earn a name is **written out per
  type**, even where it is the same everywhere: `copy`/`freeze` of an always-immutable type (`return v, nil`),
  `is_true` (the type's `IsTrue` hook wrapped in a `bool`), `slice` on the sequence types (`readSliceArgs`, then the
  `Slice` hook). A body with real logic and a clear name is a **helper called from a one-line wrapper**:
  `format([spec])` is `return FormatWithSpec(v, id, args)` on every type (`core.FormatWithSpec` — exported, since
  module-owned types call it too), and `undefined`'s conversions each `return undefinedConvert(id.String(), args)`.
  A helper keeps a helper's shape (only what it needs, no `VM` it does not use), not a member's.
- **Share leaves, not flows.** Shared helpers are small, non-generic leaves: argument parsing (`parseIntArg`,
  `readElemCallback`), bounds and allocation guards (`core/tools.go` for the core-bound ones, `internal/seq` for the generic ones), and error builders, so duplicated code still raises
  byte-identical messages.

### Declaring native functions: directives and `kavun-gen`

Every native function — a member, a free builtin, a module function — is a named Go function whose **doc comment
is its declaration and its documentation**, in this order:

1. **the summary** — the first paragraph, one sentence ending in a period (required: it leads the reference entry
   and its index line);
2. **the body** — optional Markdown: contracts, fenced `go` examples (each one runs as written — see
   [Examples](#examples-are-tests) below);
3. **the directive, last** — one `//kavun:` line (no space after the slashes; gofmt keeps it at the end).

```go
// Pad with the fill element until length `n`.
//
// The fill is always **one element**, whatever its type, and defaults to `undefined`.
//
// ```go
// [1, 2].pad_start(5, 0)     // => [0, 0, 0, 1, 2]
// ```
//
//kavun:member pad_start pure (n[, fill]) -> array
func arrayPadStart(_ VM, v Value, id member.ID, args []Value) (Value, error) { … }
```

The directive kinds:

| directive | where | declares |
| --- | --- | --- |
| `//kavun:type <name>` | free-standing, in `<stem>.go` | a type; its doc block is the head of the type's page (Overview, Literals, Operators, …) |
| `//kavun:member <name> pure\|impure <template> -> <result>` | on the member body, in `<stem>_members.go` (or `<stem>.go`) | one `Methods` entry |
| `//kavun:builtins` | free-standing, in `vm/builtins.go` | the head of the builtins page |
| `//kavun:builtin <name> slot=<n> pure\|impure <template> -> <result>` | on the builtin body | one free builtin function |
| `//kavun:module <name> pure\|impure` | free-standing, in the module package | the module and its default grant; its doc block heads the module page |
| `//kavun:fn <name> slot=<n> pure\|impure <template> -> <result>` | on the module function body | one module function |

`pure`/`impure` is the purity claim (the optimizer may fold a `pure` entry — [purity.md](purity.md)); `->
<result>` is free text for the page only. `go generate ./...` runs `cmd/kavun-gen`, which writes **one
`kavun_gen.go` per body-owning package** — `core` (every type's table), `vm` (the builtin function table), each
`stdlib/<module>` (the module's `init()` registration; a package with constants or owned types adds
`func moduleExtras(m *core.ModuleBuilder)`) — and the whole of `docs/reference/` (one page per type, per module,
the builtins page, the stdlib index and the function matrix). Both are committed and never edited by hand;
`TestGeneratedInSync` regenerates them in memory and fails on any difference, and `go run ./cmd/kavun-gen -check
./...` does the same from the command line.

Generation is strict and stops on: an unknown directive kind or key, a malformed line or template, a member or
`fn` name that is not a bound member name (`member.Lookup`), a duplicate member or slot, a missing or two-sentence
summary, a `#`/`##` heading outside a type, module or builtins block, an indented line outside a fenced block, a
`~~~` fence, a `go` fence with anything after the language but the `norun` marker, and a members file that declares
no member.

#### Examples are tests

A fenced ```` ```go ```` block in a doc block is a **standalone Kavun script**, and its comments say what it shows:

| line | claims |
| --- | --- |
| `expr // => <render>` | the expression's value renders as `<render>` — its `format()` spelling as an element (`[1, 2]`, `"ab"`, `u"ab"`, `bytes([1])`, `1.5d`, `date("2026-01-31")`, `undefined`); on an assignment, the assigned target's value |
| `stmt // Error: <kind>[: text]` | the statement raises an error of that kind (the text after the kind is prose); the script goes on with the next line |
| any other `// …` | prose |

A note may follow a render after ` — ` (`// => [1, 2, 3] — spreads`). A check comment goes on the line that ends its
statement, or alone on the line right after it when the statement is too long to share its line. A block that is
not a runnable script — placeholders (`len(x)`, `{ ... }`), a signature summary, a block that needs a host binding
or the previous block's variables — is fenced ```` ```go norun ````, optionally followed by the reason
(```` ```go norun — a signature summary ````); the page shows a plain ```` ```go ```` fence. Prefer a setup line
(`fin := import("fin")`, `t := t"2026-08-29T15:04:05Z"`) over `norun` when one line makes the block standalone.

The harness has three edges. A `// Error:` statement runs inside a closure, so a declaration on that line does not
reach the lines after it: bind first, then check the raising use (`x := …` on one line, `x.m() // Error: …` on the
next). A `// =>` checks an expression or an assignment only: a `defer`, `for` or `if` line cannot carry one (check a
value the block leaves behind instead), and a `defer` on an `// Error:` line would run when its closure returns.
After an unexpected raise the rest of the block is not checked: the first failure is the one reported.

`TestDocExamples` (`kavun_test.go`) enforces all of it: it reads every doc block through `internal/kavungen`
(`kavungen.Examples`), runs each block that is not `norun` as one script — every module granted, an empty standard
input, output discarded, a temporary working directory — and fails, naming the example's file:line, on a block that
does not compile, a line that raises unexpectedly, a render that differs, and an `Error:` line that does not raise
that kind. The render is the binary's truth: a page that spells a value differently is the page to fix.

## Kavun Language Conventions

This section defines conventions for naming, behavior, and design choices affecting the Kavun language itself.

### Properties vs Member Functions

For builtin scalar/container types, use **member functions for all operations**, including zero-argument operations.

- Use `len()` instead of `len`.
- Use `sum()` instead of `sum`.
- Use `min()` instead of `min`.
- Use `is_empty()` instead of `empty`.

Rationale:

- One mental model: `name(...)` always means "evaluate behavior now".
- Future-proofing: zero-arg APIs can later accept optional parameters without breaking style.
- Better chain consistency: methods compose predictably.
- Avoids ambiguity around computed properties vs stored fields.

Reserved use of properties:

- Properties are allowed for plain data objects/records/modules that expose stored fields.
- Builtin type capabilities should remain method-based.

### Naming Style

- Use `snake_case` for all member APIs.
- Use short, concrete verbs/nouns.
- Avoid abbreviations unless universally recognized (`len`, `min`, `max`, `avg`).
- Use verb-based names for transformations and aggregations, noun-based names for queries.

Examples:

- Transformations: `map`, `keep`, `sort`, `reverse`, `upper`, `lower`, `trim`
- Aggregations: `reduce`, `sum`, `avg`, `min`, `max`, `count`
- Queries: `len`, `is_empty`, `contains`, `index`

### No `to_` Prefix on Members

A member name never carries a `to_` prefix: the receiver already says what is being converted, so `to_` is pure
noise. It is `.upper()`, `.title_case()`, `.string()`, `.array()` — never `to_upper`, `to_title`, `to_string`.
The conversion family's shape is `x.T([default])`: the target type *is* the name.

`parse_T()` is the one prefixed family the convention reserves, and the exception is principled: it names a
*different reading* of the receiver, not a direction of travel. A prefix that distinguishes two readings earns
its place; a prefix that restates the receiver does not. Text ships two: `s.parse_time(layout[, default])` and
`s.parse_date(layout[, default])` read the text under a layout, beside the plain conversions `s.time([default])` /
`s.date([default])`, which read the canonical grammar like every other `x.T([default])`.

### Qualifiers Are Suffixes

A name is *verb first, qualifier after*: `push_first`, `index_last`, `trim_start`, `pad_end`, `sort_in_place`,
`slice_view`, `copy_shallow`. No member name is prefix-qualified — a vocabulary where most qualifiers trail and
one leads disagrees with itself.

The positional vocabulary is `first`/`last` for elements and `_start`/`_end` for the ends of a sequence — never
`left`/`right`, which are visual words: an array has no left, it has a start. A *prefix* is at the start, so
`has_prefix`/`remove_prefix` sort with `trim_start`/`pad_start`.

### Predicate Prefixes (`is_`, `has_`, `can_`)

Boolean-returning members must use explicit predicate prefixes.

- State checks: `is_empty()`, `is_zero()`, `is_nan()`, `is_true()`
- Anchored/content checks: `has_prefix()`, `has_suffix()`
- `can_` is reserved for capability checks; no builtin uses it yet

Do not expose bare adjectives/nouns for booleans (`empty()`, `sorted()`, `zero()`).

Two predicates may coexist when they answer two different questions: `is_function(x)` is the *type* predicate
(`type_name(x) == "function"`), `is_callable(x)` the *capability* predicate — a host-defined callable type
satisfies the second without being the first. The one-case divergence is deliberate and documented.

### Mutating vs Non-mutating Methods

Default convention: methods are non-mutating and return new values where relevant.

- `sort()` returns a sorted copy.
- `upper()` returns a new string.

If an in-place variant is required, mark it explicitly with `_in_place`. The twin has the same argument
contract as its pure sibling, mutates the receiver's shared body (visible through every alias without
reassignment), and **returns the receiver**, so twins correspond: `y = x.m(args)` and `x.m_in_place(args)`
leave the same content in `x`. On a frozen receiver every mutating member raises the one error kind
`not_mutable` — `(member) type immutable-array is immutable` — so a script catches "I mutated a frozen value"
with a single kind test. The mutability check comes **first**, before anything about the arguments: a frozen
receiver raises `not_mutable` however the member was called. Every error a twin raises names the twin itself
(`sort_in_place`, never `sort`), with the same argument wording as its sibling.

A twin exists only where three conditions hold: the receiver has a mutable heap body (not `string`, immutable
by construction; not `range`, which has no storage); the result is the receiver's own type (so `map`, `split`,
`chunk` never have twins); and the reuse is the whole receiver (so `repeat` has none — its result is n× the
receiver). The suffix is load-bearing at the optimizer layer: the mutable types derive method purity from it,
so an unsuffixed mutating member would be silently constant-folded — never overload a non-mutating name with
mutation.

Never provide two methods with the same base name where one mutates and one does not. Note that
`push`/`push_first` are **not** a twin pair — they are two operations at two ends; the twins are
`push`/`push_in_place` and `push_first`/`push_first_in_place`.

This convention is a direct consequence of the [Purity Contract](purity.md): operators and methods must be pure by
default so the AST optimizer can fold constant subexpressions safely. Impure operations should be exposed as
top-level builtin functions (registered with `Pure = false`); `_in_place` methods are the last-resort escape hatch.

### Mutating vs Non-mutating Functions (Kavun-language, user-defined)

This is a **requirement**, not a recommendation: a Kavun function must not mutate any of its arguments' shared
body unless its own name carries the `_in_place` suffix. This extends the member-method convention above from
builtin types up to whole functions written in Kavun — since containers share their body across assignment and
argument-passing by design, a caller can only trust what a function does to what they passed by reading its
name, not its implementation.

```go
// Wrong: mutates the caller's array with no signal at the call site.
normalize = func(items) {
    items.sort_in_place()
    return items
}

// Right: the name tells the caller items will be mutated.
normalize_in_place = func(items) {
    items.sort_in_place()
    return items
}
```

A function that invokes a caller-supplied callback on its own arguments is held to the same rule: unless it can
guarantee the callback never mutates, it must be named `_in_place` too — a callback's behavior isn't known
until runtime, so it can't be assumed safe by default.

**Current status:** this rule is enforced today only by code review and this documented contract — the compiler
and VM do not yet verify it. Host-configurable, compiler-enforced checking (mirroring `Script.SetAssignmentMode`,
`script.go:90`) is planned but requires a sound alias-tracing/interprocedural static analysis (parameter alias
tracking through a function body, propagated across calls to other user-defined functions, with any
runtime-supplied callback treated as unprovable/mutating by default) that hasn't been designed yet. Until that
lands, follow this convention as a hard rule, not a "usually" — a violation won't be caught for you.

### Range/Slice Bounds: Inclusive-Start, Exclusive-Stop

`range()`, three-part slicing (`a[start:stop:step]`), and the `..`/`..:` range literal (`low..high[:step]`, see
[Range literals](language.md#range-literals)) all use the same bound convention: the start is inclusive, the stop is
exclusive. `1..5` and `range(1, 5)` both mean 1, 2, 3, 4 — never 5.

This was a deliberate choice, not just "that's what `range()` already did." The half-open convention (inclusive
start, exclusive stop) is the stronger engineering default independent of which language you're coming from — see
Dijkstra's ["Why numbering should start at zero"](https://www.cs.utexas.edu/users/EWD/transcriptions/EWD08xx/EWD831.html)
for the canonical argument:

- **Composes cleanly**: `a..b` followed by `b..c` covers `a..c` with no gap or overlap. An inclusive `a..b` would
  need `a..(b-1)` then `b..c` to compose the same way.
- **Empty ranges need no sentinel**: `a..a` is naturally empty. Inclusive semantics have no way to spell "empty
  starting at `a`" without something like `a..(a-1)`.
- **Length is just subtraction**: `stop - start`, with no `+1` anywhere to get wrong.
- **No boundary overflow**: an inclusive upper bound at a type's max value has no valid "one past the end" to fall
  back to (this is exactly why Rust needed a dedicated `..=` operator alongside plain `..`).

It also keeps `..` honest as *pure sugar*: `compileRangeExpr` (`compiler/compiler_impl.go`) compiles it to a call to
the exact same underlying `range` builtin as writing `range(...)` by hand, so it must mean exactly what `range()`
means. If `..` were inclusive while `range()` stayed exclusive, the sugar would have to translate `a..b` into
`range(a, b+1)` — which only makes sense for integers and breaks the moment `range()` grows support for another type
with no well-defined "successor" (a float, a decimal, a string).

Note this is "the same call" in the sense of always invoking `range`'s one true implementation, not "the same
identifier reference": `low..high` is a language construct, immune to a local `range := ...` reassignment, the same
way a `b"..."` bytes literal is immune to reassigning `byte` — see
[Shadowing and reassigning builtins](language.md#shadowing-and-reassigning-builtins).

### Modules and module-owned types

- **A builtin module** has a short, well-known, one-word lowercase name that is never a builtin identifier
  (`io`, `fs`, `re`, `clock`), and is either *pure* (only computes; importable by default) or *world access*
  (needs a grant) — never both: a module is the grant unit, so world access splits into its own module (`os`,
  `fs`, `exec`). No function exists both as a module function and as a type member (`x.sqrt()`, not
  `math.sqrt(x)`), and no module is named after a type's plural.
- **A module-owned type** (one that exists only for one module) is named `<module>.<constructor>` in Kavun —
  `re.pattern`, `fs.file`, `exec.command`, `exec.process`, `rand.generator`, `fin.year_fraction` — and is built only
  by that module's constructor. On the Go side it lives in its module's package (`stdlib/re/pattern.go`),
  registered by the module (`core.ModuleBuilder.Type`); its Go identifiers drop the prefix the package already
  gives (`re.TypePattern`), while its type id keeps it (`value.RePattern`).
- **A handle on the world** (`fs.file`, `exec.*`) is equal by identity, has no `string()`, renders as
  `<type:detail>`, and every member is impure; **a value with text content** (`re.pattern`) has `string()` and
  renders as the expression that builds it.

### One Name, One Template

A member name shared by two or more types means the same thing on each, so it has the **same signature template
on every type** — the same count set and the same parameter names — and code holding a value of unknown type can
call it without a type check: every conversion is `x.T([default])`, including the total ones (the default is then
never used) and the same-type copies; every locator is `index([x[, default]])`. A type may **refuse** a reading it
cannot give meaning to — `dict` has no blank set, so its `contains`/`keep`/`index`/… require an argument — but never
silently accept a narrower or different one. A different meaning under a shared name is a naming defect, not a
licence for a second template. `TestMemberTemplateParity` walks every `Methods` table and pins the rule; its
exception table (name, types, reason) is the record of every deliberate divergence.

### Arity and templates

Every callable — script function, builtin, module function, bound member, host closure — answers one pair:

| term | meaning |
| --- | --- |
| **arity** `N` | the count of fixed parameters |
| **variadic** | the callable also accepts more; the tail is its **rest** |
| accepts `k` | `k == N`, or variadic and `k > N` |
| render | `/N` fixed, `/N+` variadic ("N or more"): `<compiled-function/2+>`, `<builtin-function:min/1+>` |

A script function's rest is its last parameter, an immutable array (`[]` when nothing follows):
`func(a, b, ...c)` has arity 2 and is variadic. A native function declares its parameters with a **signature
template** — the parameter list a script author would write for the same body, receiver excluded:

| grammar | example | pair |
| --- | --- | --- |
| names | `(width, fill)` | 2, fixed |
| optional group `[, x]` | `(width[, fill])`, `([sep])` | 1 / 0, variadic |
| rest `...name` | `(first, ...rest)` | 1, variadic |
| readings ` \| ` | `() \| (components) \| (start, stop[, step])` | the fewest over the readings (0), variadic |

The arity is the fewest arguments the body accepts; variadic when it accepts more. Every reading must accept a
count no other reading does — `(a) | (b)` or `([a]) | ()` is refused: one spelling per shape, and readings that
share a count but differ by role belong in prose.

The dispatcher checks the floor from the template before the body runs (see [Implementing
members](#implementing-members-methods-tables)); a body keeps only a bound or a gap on its rest and refuses with
`core.ErrArgCount`. A wrong count raises `wrong_num_arguments` in one shape: a native callable quotes its
template — `(join) expected ([sep]), got 2 arguments`, `(range) expected () | (components) | (start, stop[,
step]), got 4 arguments` — and a callable without one (a script function, a closure built with
`core.SignatureOf`) states its pair — `(call) expected 2 arguments, got 1 argument`, `(call) expected 2 or more
arguments, got 0 arguments`.

### Callbacks

A member that takes a function calls it in the **fewest form it accepts**, read from its pair, never from what
kind of callable it is (`core.CallbackForm`):

- an element callback (`map`, `flat_map`, `for_each`, and the predicate reading of `keep`, `remove`, `contains`,
  `count`, `any`, `all`, `index`, …) gets the element when it accepts one argument, else `(index, element)` (a
  dict: the key, else `(key, value)`);
- a `reduce` callback gets `(acc, element)` when it accepts two, else `(acc, index, element)`;
- a `fin.root` / `fin.brackets` callback gets `x`, and must accept one argument.

Anything else raises `invalid_argument_type` (`expects type f/1 or f/2, got <compiled-function/0>`). So
`func(x, ...rest)` and `func(...args)` get the element, and `min`, `int`, `"abc".contains`, `"abc".has_prefix` are
element callbacks like any lambda. **A lambda is the one way to call a callable in another form** — to choose the
form, reorder, fix or drop arguments: `xs.map((i, x) => f(x, i))`, `[2, 3].map(n => "7".pad_start(n, '0'))`. There is
no adapter builtin and no per-callable dispatch rule.

### Optional arguments

Two argument devices exist, and a member may use **one or the other, never both**:

- **The optional trailing `[default]`** — for members that can *miss*: `first(d)`, `min(d)`, `index(x, d)`, and
  every conversion `x.T(d)`. Absence answers the default instead of `undefined`/raising.
- **The variadic set** — for members whose arguments are peers: `append(...items)`, `contains(a, b, ...)`,
  `trim(...set)`, `split(...seps)`.

The two cannot share a member because no type test separates a second needle from a fallback: in `x.m(a, b)`,
`b` is a set member under one device and a default under the other. The mechanical test is *can this member
miss?* — if yes, it takes the default and is never variadic.

An **optional argument with a default value** (`join([sep])` — no separator means the empty separator;
`format([spec])`) is fine. A **behavior-switching scalar option** is not: an extra scalar that changes the
operation's meaning collides with the variadic reading (`split(",", 1)` would silently read the `1` as a second
separator) and hides a second operation under one name — give the second operation its own name instead
(`partition` is the split-once spelling; `split` itself takes no limit).

Meaning is selected by the **argument's type**, never by a name suffix or a flag: no argument means the blank
set, a function is a predicate, an argument of the receiver's own kind is a contiguous run, anything else is
one element. A member that does not declare a reading raises, naming the ones it has.

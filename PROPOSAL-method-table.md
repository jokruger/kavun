# PROPOSAL: a declarative per-type method table

Status: **proposal, not scheduled.** Written 2026-10-03 after the dispatch measurements recorded in `NOTES.md`.
Nothing here is implemented. The proposal is self-contained; where it quotes a number, the number comes from
`benchmarks/dispatch_benchmark_test.go` on the machine named in `NOTES.md`.

Updated 2026-10-04: §13 now compares all options (including a per-member opcode and a single `CallCore` opcode),
and §14 gives the recommendation. Since this was written, the de-generic refactor removed `core/generic_seq.go`.
Every type now has one `case` per member name calling one function. As a result, §7's engine discussion and the
"structural member passes three switches" row in §1 are out of date: those functions go straight into the
table. The purity-predicate row (`!HasSuffix(name, "_in_place")`) still holds.

## 1. Summary

Today every builtin type answers a member call with one hook, `ValueTypeDescr.MethodCall(vm, v, name, args)`, whose
body is a `switch name` over every member the type has (up to 57 cases, 212 distinct names across `core`). A second
hook, `IsMethodPure(name) bool`, tells the optimizer which of those names it may fold.

This proposal replaces both hooks with **data**: a table that maps each member name to an entry carrying the
implementation, its purity, its arity and its `_in_place` flag. Names are interned into `uint16` IDs at
registration, call sites are resolved to IDs once per bytecode load, and a call becomes an indexed lookup plus one
indirect call.

**It is not a performance proposal.** Measured, the table is within one nanosecond of the string switch per call
(3.0 ns vs 4.0 ns for the dispatch step, invisible in a whole call of 13 ns). The reasons are correctness and
maintenance:

| problem today | with the table |
| --- | --- |
| Purity is a separate string predicate. For array, bytes, runes and dict it is `!strings.HasSuffix(name, "_in_place")`, which answers "pure" for names that do not exist. | `Pure` is a field beside the implementation. Invariants (`InPlace ⇒ !Pure`) are checked at registration. |
| Nearly every `case` opens with the same `len(args)` check and renders the same error text by hand (175 occurrences of `"0"` alone). | Arity is declared once per entry and checked centrally with uniform text. |
| `docs/types/function-matrix.md` is re-derived by hand from `case` labels; CLAUDE.md names this the most common documentation defect. | The matrix is generated from the tables, and a test keeps it in sync. |
| A structural member (`trim`, `has_prefix`, …) passes three switches plus `HasSuffix`/`TrimSuffix` before its engine runs (23 ns vs 9.5 ns flat). | The verb and the in-place flag are resolved fields; the engine is called directly. |
| An embedder writes a string switch, duplicates the universal members, and has no way to declare purity per member except by writing a second switch. | An embedder fills a map literal. Universal members are inherited. |
| Internal re-dispatch spells names as string literals (`seq.MethodCall(vm, "repeat", …)` in `vm/builtins.go`). | Internal calls go by ID or through one lookup helper. |

## 2. Vocabulary

```go
package core

// MethodID identifies a member name process-wide. 0 is MethodUnknown: a name no registered type has.
// IDs are assigned at registration and are stable for the life of the process only — they are never
// serialized (see §6).
type MethodID uint16

const MethodUnknown MethodID = 0

// MethodInfo is what every implementation receives about the member it is answering.
type MethodInfo struct {
	ID      MethodID
	Name    string   // the spelling the script used — for error text
	InPlace bool     // this is the "_in_place" twin
	Base    MethodID // the non-mutating twin's ID (== ID when there is no twin)
}
```

Interning is global and init-time:

```go
// InternMethod answers the ID for a member name, assigning a new one on first sight.
// Registration-time only: it is not safe to call concurrently with compilation or execution.
func InternMethod(name string) MethodID

// MethodName answers the spelling for an ID ("" for MethodUnknown).
func MethodName(id MethodID) string

// LookupMethod answers the ID for a known name without assigning one; MethodUnknown if none.
func LookupMethod(name string) MethodID
```

`uint16` leaves room for 65,535 names; the builtin set is 212 and an embedder adds a few dozen.

## 3. The table

```go
// MethodFunc implements one member. m carries the name for error text and the InPlace flag, so one function
// serves both twins of a pair.
type MethodFunc func(vm VM, v Value, m *MethodInfo, args []Value) (Value, error)

// Arity is the accepted argument-count range. Max == Variadic means "Min or more".
type Arity struct{ Min, Max int }

const Variadic = -1

func Exactly(n int) Arity      { return Arity{n, n} }
func Between(lo, hi int) Arity { return Arity{lo, hi} }
func AtLeast(n int) Arity      { return Arity{n, Variadic} }

// ArityAny skips the central check: the implementation validates and renders its own message.
var ArityAny = Arity{0, Variadic}

type MethodEntry struct {
	Fn      MethodFunc
	Pure    bool  // folding candidate for the optimizer (docs/purity.md); must be false when InPlace
	Arity   Arity // checked before Fn runs; ArityAny to opt out
	InPlace bool  // mutates the receiver's body through its pointer (docs/conventions.md); name must end in _in_place
}

// MethodTable is the declarative form a type registers. Keys are the member names as a script spells them.
type MethodTable map[string]MethodEntry
```

The arity renderer produces exactly the strings the code emits today, so no test expectation moves:

| `Arity` | rendered expected-count |
| --- | --- |
| `Exactly(0)` | `"0"` |
| `Between(0, 1)` | `"0 or 1"` |
| `Between(1, 3)` | `"1 to 3"` |
| `AtLeast(1)` | `"1 or more"` |

Entries whose message today carries extra explanation (`"1 or more (a map has no blank reading)"`) declare
`ArityAny` and keep rendering their own.

## 4. The descriptor

`ValueTypeDescr` loses two hooks and gains two fields:

```go
type ValueTypeDescr struct {
	// ... every other hook unchanged ...

	// Methods is the declarative member table. Registration compiles it into the per-type dispatch slice;
	// after registration the map is not consulted on the call path.
	Methods MethodTable

	// MethodFallback answers a name the table does not have. Nil for every builtin type except record, whose
	// members are its callable fields and so cannot be tabled. An embedder type with per-value members uses it
	// the same way. It is never a folding candidate.
	MethodFallback func(vm VM, v Value, name string, args []Value) (Value, error)

	// removed: MethodCall, IsMethodPure

	// not exported, built by setValueType:
	methods []*boundMethod // indexed by MethodID; nil = absent
}

type boundMethod struct {
	MethodEntry
	info MethodInfo
}
```

Universal members move into `DefaultValueType.Methods`:

```go
var DefaultValueType = ValueTypeDescr{
	// ...
	Methods: MethodTable{
		"is_true": {Fn: universalIsTrue, Pure: true, Arity: Exactly(0)},
	},
}
```

`setValueType` today fills every nil hook from `DefaultValueType` by reflection. For `Methods` it does a **merge**
instead of a replace: the default entries are added unless the type overrides the same name. Then it builds the
slice and validates:

- every key is interned (`InternMethod`), the slice is sized to the type's highest ID;
- a key ending in `_in_place` must have `InPlace: true` and vice versa (the naming rule of `docs/conventions.md`,
  today enforced by review only);
- `InPlace` entries must not be `Pure`;
- an `_in_place` entry's `Base` is the ID of the same name without the suffix, which must also be in the table
  (the twin rule — a type never exposes a mutating member without its non-mutating twin);
- `Fn` is non-nil.

A violation is a programming error in the type, so `setValueType` panics at init with the type and member named.
This is the one place a panic is correct: it happens before any script can run, in the embedder's own `init()`.

## 5. Dispatch

The instruction set does not change. `CallMethod` keeps `Op3 = static string index`. What changes is one parallel
table in `Static`, built at the three points where a `Static` is finalized today
(`compiler/static.go`, `vm/bytecode.go` Decode, `vm/vm.go` load) — exactly like `StringLens`:

```go
type Static struct {
	Strings    []string
	StringLens []int64
	MethodIDs  []MethodID // LookupMethod(Strings[i]) for every static string; MethodUnknown if no type has it
	// ...
}

func (s *Static) BuildMethodIDs() // idempotent, like BuildStringLens
```

Resolving every static string, not only the ones used as method names, costs one map lookup per string per load
and avoids threading a "this string is a method name" bit through the compiler.

The VM's `CallMethod` case becomes:

```go
case bc.CallMethod:
	// ... spread handling unchanged ...
	idx := v.curInsts[v.ip].Op3
	res, err := receiver.CallMethod(v, v.static.MethodIDs[idx], v.static.Strings[idx], v.stack[v.sp-numArgs:v.sp])
```

and `Value.CallMethod` is the whole dispatcher:

```go
// CallMethod resolves and runs the member. name is only read on the miss path (error text, fallback).
func (v Value) CallMethod(vm VM, id MethodID, name string, args []Value) (Value, error) {
	d := &ValueTypes[v.Type]
	if int(id) < len(d.methods) {
		if m := d.methods[id]; m != nil {
			if m.Arity != ArityAny {
				if n := len(args); n < m.Arity.Min || (m.Arity.Max != Variadic && n > m.Arity.Max) {
					return Undefined, errs.NewWrongNumArgumentsError(name, m.Arity.String(), n)
				}
			}
			if m.InPlace && v.Immutable {
				return Undefined, errs.NewNotMutableError(name, v.TypeName())
			}
			return m.Fn(vm, v, &m.info, args)
		}
	}
	if d.MethodFallback != nil {
		return d.MethodFallback(vm, v, name, args)
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}
```

Per call that is: one index into `ValueTypes`, one bounds check, one load, one nil check, the arity compare, and
an indirect call. Measured as the `id_table` row of `BenchmarkDispatch`: 3.0 ns against the switch's 4.0 ns when
the selector is stable, 13 ns for both when it is random.

Other callers of the old hook:

- `DeferMethod` captures `MethodID` and name in the `deferred` record; `invokeDeferred` calls `CallMethod`.
- Internal re-dispatch (`vm/builtins.go`: `seq.MethodCall(vm, "repeat", …)`) uses a package-level interned ID
  (`var methodRepeat = InternMethod("repeat")`) or the convenience `v.CallMethodByName(vm, "repeat", args)`,
  which does the lookup and is for cold paths only.
- The optimizer (`compiler/optimizer_impl.go`, the `IsMethodPure` call) becomes
  `core.MethodIsPure(receiver.Type, name)`: lookup, `entry.Pure`, `false` on a miss. The conservative default is
  preserved — a name the table lacks is never a folding candidate, and a `MethodFallback` is never pure.

## 6. Why IDs are resolved at load and never stored in bytecode

Bytecode is gob-serialized (`vm/bytecode.go` `Encode`/`Decode`) and an embedder may compile in one process and
run in another, with a different set of host types registered and therefore a different interning order. A
`MethodID` baked into `Op3` would be meaningless there. Keeping the name in the static pool and resolving at load
costs nothing per call and keeps bytecode exactly as portable as it is today. It also keeps the compiler ignorant of
the registry: compilation needs no types registered at all.

Consequence, to document in `docs/embedding.md`: **register every host type before loading or running bytecode**
that calls its members. A type registered after a `Static` was finalized is reachable only through
`MethodFallback` of a type that existed, or after re-loading. This is the same rule `SetValueType` effectively has
today (a script cannot call a type that does not exist yet), made explicit.

## 7. Shared engines: verbs become parameters, not names

`core/generic_seq.go` is the one implementation of the sequence surface for array, string, runes and bytes. Today
its entry points switch on the name a second time (`SeqStructuralMember` on `strings.TrimSuffix(name,
"_in_place")`, `applyMatchVerb` on `name`). With the table, the type binds the verb when it fills its entry:

```go
// in core/array.go
Methods: MethodTable{
	"trim":             {Fn: arraySeqTrim(true, true),   Pure: true, Arity: ArityAny},
	"trim_in_place":    {Fn: arraySeqTrim(true, true),   InPlace: true, Arity: ArityAny},
	"trim_start":       {Fn: arraySeqTrim(true, false),  Pure: true, Arity: ArityAny},
	"has_prefix":       {Fn: arraySeqAnchored(false, false), Pure: true, Arity: AtLeast(1)},
	// ...
}

func arraySeqTrim(start, end bool) MethodFunc {
	return func(vm VM, v Value, m *MethodInfo, args []Value) (Value, error) {
		res, err := SeqTrimMember(m.Name, v, args, NewArrayValue, arrayTypeResolve, arrayEncodeStructuralArg,
			func(a, b Value) bool { return a.Equal(b) }, IsBlankElement, start, end)
		if err != nil || !m.InPlace {
			return res, err
		}
		(*Array)(v.Ptr).Set((*Array)(res.Ptr).Elements)
		return v, nil
	}
}
```

The engine's per-verb functions (`SeqTrimMember`, `SeqAnchoredMember`, `SeqReplaceMember`, …) already exist; what
disappears is the name switch in front of them and the `HasSuffix`/`TrimSuffix` pair in every type. `m.Name` still
reaches every error message unchanged, which is the rule the current code states in `SeqStructuralMember`'s doc
comment ("the full member name reaches every error message; only the verb switch ignores the suffix").

The "change a sequence member HERE, not four times" rule of CLAUDE.md survives: the engine function is still the
single implementation; the four types each add one table line per member, as they add one `case` label today.

## 8. The embedder's view

`docs/extending-types.md` gains a section that replaces "write a `MethodCall` switch" with:

```go
const Money = value.FirstUserDefinedType + 0

func init() {
	core.SetValueType(Money, core.ValueTypeDescr{
		Name:   func(core.Value) string { return "money" },
		String: moneyString,
		Methods: core.MethodTable{
			"amount":   {Fn: moneyAmount,   Pure: true, Arity: core.Exactly(0)},
			"currency": {Fn: moneyCurrency, Pure: true, Arity: core.Exactly(0)},
			"convert":  {Fn: moneyConvert,  Pure: false, Arity: core.Exactly(1)}, // reads a rate table: impure
		},
	})
}

func moneyAmount(_ core.VM, v core.Value, _ *core.MethodInfo, _ []core.Value) (core.Value, error) {
	return core.NewDecimalValue(moneyOf(v).amount), nil
}
```

What the embedder no longer writes: the `default:` branch raising `invalid_method`, the `is_true` case, the
`len(args)` checks, and a second `IsMethodPure` switch. What the embedder gains: a registration-time panic naming
the member if a `_in_place` name is declared pure, instead of a silently mis-folded script.

A host type whose members are per-value (a reflective wrapper over a Go struct, a record-like bag) declares
`MethodFallback` and is treated as `record` is: callable by name, never folded.

## 9. Introspection byproducts

Not in scope to expose to scripts, but available once the data exists:

- **`function-matrix.md` generation.** A test (`TestFunctionMatrixInSync`) renders the matrix from
  `ValueTypes[*].Methods` and diffs it against `docs/types/function-matrix.md`; a `go generate` target writes
  it. The page's role stays what CLAUDE.md says — existence only — and the per-type pages keep the contracts.
- **A `has_method`-style query** for the host (`core.HasMethod(t, name)`), useful to `kavun_custom_types_test.go`
  and to embedders validating configuration before running a script.
- **The arity column** of the matrix comes for free.

## 10. Performance, stated plainly

| path | today | with the table | source |
| --- | ---: | ---: | --- |
| dispatch step, stable selector | 4.0 ns | 3.0 ns | `BenchmarkDispatch/mono` |
| dispatch step, random selector | 15.4 ns | 13.3 ns | `BenchmarkDispatch/uniform` |
| whole `a.len()` call | 13.5 ns | est. 13 ns | `BenchmarkMemberCall` |
| structural member, dispatch part | 23 ns | est. 5 ns | `tmp/dispatchbench` chain benchmark |

Two effects cancel: the indirect call stops Go from inlining tiny bodies such as `len` (about one nanosecond), and
the lookup is one nanosecond cheaper than the jump-table-plus-compares. The structural flattening is real but
small against those members' bodies (145 to 194 ns). **Expect no measurable change in `cmd/bench`.** If a run
shows one in either direction, that is a finding, not the goal.

Memory: one `[]*boundMethod` per registered type, sized to that type's highest ID. With global IDs assigned in
registration order and ~220 names, a type holds at most ~220 pointers (1.8 KB); forty types are under 100 KB.
`MethodInfo` is embedded in the bound entry, so the hot path allocates nothing.

## 11. Migration plan

The conversion is mechanical and can land type by type behind the same tests, because the external behaviour —
which members exist, what they answer, what they raise and with which text — does not change.

- **S0 — scaffolding.** `MethodID`, interning, `MethodEntry`/`MethodTable`, `Arity` and its renderer,
  `Static.MethodIDs` + `BuildMethodIDs` at the three finalize points, `Value.CallMethod`. Keep the old hooks
  alive: `setValueType` wraps a legacy `MethodCall` into `MethodFallback` so untouched types keep working, and
  `IsMethodPure` is consulted by `MethodIsPure` when the table has no entry. The suite stays green with zero
  types converted.
- **S1 — one scalar type** (`bool` or `byte`, 7–9 members) to settle the pattern and the arity renderer against
  the existing error expectations in `kavun_test.go`.
- **S2 — the sequence family** (`array`, `string`, `runes`, `bytes`): engine entry points lose their name
  switches (§7); `_in_place` twins become `InPlace` entries sharing a bound function; the `!HasSuffix` purity
  predicates disappear.
- **S3 — the rest of core**: `dict`, `int_range`, `int`, `float`, `decimal`, `time`, `date`, `error`, `rune`,
  the callables, the iterators, the `fin.*` types. `record` keeps its dynamic behaviour as `MethodFallback`.
- **S4 — remove the legacy hooks**, the wrapper in `setValueType`, `Value.MethodCall`, the `"repeat"` literal in
  `vm/builtins.go`; `deferred.method` becomes an ID.
- **S5 — docs and tests.** `docs/purity.md` ("Method purity" becomes "the `Pure` field"), `docs/extending-types.md`
  (§8), `docs/conventions.md` (the `_in_place` rule is now enforced at registration), `docs/embedding.md`
  (registration-before-load, §6), `docs/vm.md` (the `MethodIDs` table), `CHANGELOG.md`. Add
  `TestFunctionMatrixInSync` and a registration test that a mis-declared table panics at init.

Each stage is independently reviewable and leaves the suite green. The matrix test in S5 is also the audit that
no member was lost in S1–S3.

## 12. Risks and open questions

- **Init-order dependence of IDs.** IDs are assigned in registration order, which depends on Go's package-init
  order. Nothing may store an ID beyond the process (§6); a test should assert `Static.MethodIDs` is rebuilt on
  every `Decode`.
- **Central checks vs. bespoke messages.** A few members render argument-count messages with explanation
  (`"1 or more (a map has no blank reading)"`). They opt out with `ArityAny`; the renderer must never change the
  text of the ones it does take over, which `kavun_test.go` pins.
- **Two mutability errors today.** `errs.NewNotMutableError` and `immutableTwinError` are the same error under
  two names; the central `InPlace` check in `CallMethod` uses one. Confirm no test distinguishes them (none
  should: `immutableTwinError` is a one-line alias).
- **`Append`/`Delete` hooks.** The operator forms `+`/`-` reach `Append`/`Delete` directly and stay hooks; the
  member forms `append`/`append_in_place` become table entries that call the same functions. Parity stays where
  `value_binaryop_matrix_test.go` pins it.
- **Should builtin IDs be constants? Answered in §14: yes.** A `core/method` package with append-only
  `const Len, Copy, … MethodID` fixes builtin IDs regardless of init order. It also makes storing them in bytecode
  as safe as storing opcodes, which option I (§13) depends on.
- **Open: expose `methods()`/`has_method()` to scripts?** Out of scope here; the data makes it a small follow-up
  if the language wants it, and `TODO.md` is the place to park that.

## 13. Options compared

Added 2026-10-04. Nine ways to resolve `x.name(args)`. The numbers are the Go-level dispatch step from
`tmp/dispatchbench/RESULTS.md` (71 real `runes` labels, i7-9750H). "Mono" means the same member at every call;
"poly" means a uniform random mix, which is limited by branch misprediction and is a worst case. A direct call
with no dispatch costs 1.3 ns. A whole trivial member call (`a.len()`) is 13.5 ns. The builtin `len(a)`, which
calls the `Len` hook with no name lookup, is 12.7 ns.

### Background: why members aren't hooks today

The descriptor's other hooks (`BinaryOp`, `Access`, `Iterator`, `Format`, `As*`, …) are called **by the
runtime**: VM opcodes, conversions, `json`, `fmt`. That set is closed because the VM's set of operations is
closed. Members are called **by scripts, by name**, and that set is open (212 names across `core`, and embedders
add more). Where the two overlap, the member already *is* a hook: `Len`, `Contains` (`in`), `Append`/`Delete`
(`+`/`-`), `Copy`, `Slice`, `IsTrue`. A useful rule follows that holds under every option below: **a member gets
a typed hook when Go code (VM, builtins, optimizer, stdlib) must call it on a type it doesn't know**. The rest of
the surface is data.

### A. Keep the string switch (status quo)

One `switch name` per type in `MethodCall`, plus a separate `IsMethodPure(name)` predicate.

- **Pros:** works, and it's already there. Go compiles it to a jump table on `len(name)` followed by integer
  compares, with no `cmpstring` (4.0 ns mono, 15.4 ns poly). Embedders find it easy to understand. Nothing
  to migrate.
- **Cons:** the problems in §1 stay conventions enforced only by review. Purity is a second switch, and for
  array/bytes/runes/dict it is `!HasSuffix(name, "_in_place")`, which answers "pure" for names that don't exist.
  Every `case` repeats its own `len(args)` check. `function-matrix.md` is re-derived by hand. Embedders must
  write the `default:`/`is_true` boilerplate.

### B. `uint16` ID switch, no table

Names are interned to IDs at load, and each type switches on the ID.

- **Pros:** slightly faster than A (3.2 ns mono, 11.8 ns poly).
- **Cons:** keeps every structural problem of A and adds the interning registry. Embedders must switch on
  interned *variables*, which Go cannot turn into a jump table. This option buys the least for the same
  machinery.

### C. Declarative table, IDs assigned in registration order, resolved at load (§2–§6 of this proposal)

`Methods: MethodTable{name: {Fn, Pure, Arity, InPlace}}`, compiled into a slice indexed by `MethodID`. `Static.MethodIDs`
maps each static string to its ID once per load. A miss falls to `MethodFallback`.

- **Pros:** purity, arity and the `_in_place` flag become fields, checked at registration (`InPlace ⇒ !Pure`,
  twin rule, suffix rule). Arity errors are checked centrally with the same text as today. The function
  matrix can be generated and tested. Embedders write a map literal. Bytecode stays as portable as it is
  today. No shadowing risk: tables are per type and keyed by name, so a new builtin member never takes over an
  embedder's member of the same name. Speed 3.0–3.8 ns mono, 13.3–13.8 ns poly.
- **Cons:** a large mechanical migration (§11). IDs depend on Go's package-init order, so they must never leave
  the process. One indirect call per member, which stops Go from inlining tiny bodies (about 1 ns, cancelling
  the lookup saving). No measurable speed-up is expected.

### D. C with frozen builtin IDs (`core/method` constants, append-only) — refinement of C

As C, except that every builtin member name has a `const` ID in a `core/method` package. Numbering is
**append-only**: new names get new numbers, and a removed name's number is never reused. Embedder names are
interned above the constant range at registration.

- **Pros:** everything C has. IDs no longer depend on init order. Hot internal callers (`vm/builtins.go`'s
  `"repeat"`, the optimizer) use constants. Builtin IDs become as stable as opcode numbers, which are already
  baked into bytecode with no version check, so storing builtin IDs in bytecode adds no new kind of coupling.
  This is what makes I possible.
- **Cons:** one more file to keep in step with the type tables, though a registration-time check can catch a
  table key that has no constant. Append-only numbering leaves gaps.

### E. Registration-order IDs baked into bytecode

The compiler writes the interned ID into `Op3`.

- **Pros:** removes the one `Static.MethodIDs` load per call.
- **Cons:** breaks gob portability between processes with different registrations (§6). It also makes the
  compiler depend on the type registry. Rejected. D gives the safe version of this for builtin names.

### F. Per-type `map[string]`

- **Pros:** the simplest code to write.
- **Cons:** the slowest option measured (14.5 ns mono, 29.3 ns poly) and no better structurally than C's slice.
  Rejected.

### G. Call-site inline caches

Each `CallMethod` site remembers (receiver type → resolved entry).

- **Pros:** the fastest when a site is polymorphic (4.1 ns flat for any mix).
- **Cons:** saves about 5 ns per call only where sites are polymorphic. Needs mutable storage owned by
  `Compiled`, because `Clone` shares `*vm.Bytecode`. Adds a mutable structure to a runtime that is otherwise
  read-only after load. Rejected at Kavun's call costs.

### H. A typed hook and a dedicated opcode per core member

Choose a fixed core set (up to about 100 members). Each gets a descriptor hook (`Len func(Value) int64`, …) and
an opcode (`LEN`, …). The compiler turns `x.len()` into `LEN`, and the VM calls `ValueTypes[t].Len`. The only
switch is the VM's opcode switch, which is paid on every instruction anyway.

- **Pros:** no member-level lookup at all. Each opcode body can be specialized: fixed arity, no args slice, no
  spread check, result written straight to the stack. That skips part of the generic `CallMethod` machinery,
  which is probably where most of the 13.5 ns goes. Each member gets its own indirect call site in the VM, which
  may predict better in polymorphic code. Typed hooks can also be reused by Go code, as `Len` already is.
- **Cons:**
  - **Opcode space.** `Opcode` is a `byte` and about 70 are in use. 100 more leaves about 86 for every future
    language feature, including what `TODO.md` defers.
  - **VM loop growth.** Every opcode is another `case` in `vm/vm.go`'s run loop. A much larger run function can
    slow *every* opcode through register pressure and instruction-cache misses.
  - **Per-member cost.** Each member needs a VM case, a `maxstack` entry, a descriptor field, purity metadata
    and a fallback path. 100 named fields on `ValueTypeDescr` is unwieldy.
  - **No gain for heavy members.** The win exists only for members with tiny bodies. For `trim` (194 ns) or
    `map` (566 ns) it is noise, so most of a 100-member set gains nothing.
  - **Unmeasured.** The best data point is the builtin `len(a)` at 12.7 ns against `a.len()` at 13.5 ns. A
    dedicated opcode should come in under that, but by how much has not been measured.

### I. One `CallCore` opcode, core member ID in `Op1` (on top of D)

A core member call never needs the spread path, so `Op1` (the spread flag in `CallMethod`) is free for the ID:

```
CallCore   Op1 = core member ID (byte)   Op2 = nargs   Op3 = static name index (fallback + error text)
```

```go
case bc.CallCore:
	in := v.curInsts[v.ip]
	recv := v.stack[v.sp-1-int(in.Op2)]
	if m := core.ValueTypes[recv.Type].methods[in.Op1]; m != nil {
		res, err = m.Fn(v, recv, &m.info, v.stack[v.sp-int(in.Op2):v.sp])
	} else {
		res, err = recv.CallMethodByName(v, v.static.Strings[in.Op3], args) // record, embedder names
	}
```

- **Pros:** the same benefit as H for dispatch, using one opcode instead of one per member. The VM loop grows
  by one case. The core set can grow to 255 without touching the VM. It reuses D's table, so there is no second
  per-member structure.
- **Cons:** compared with C/D it saves only the `Static.MethodIDs` load and, for fixed-arity entries, the
  central arity compare. That is sub-nanosecond in mono, so on dispatch alone it is barely distinguishable from
  D. Compared with H it loses specialized opcode bodies and per-member branch sites. Requires D's frozen IDs.
  Builtin IDs from 256 up must use the generic `CallMethod`.

### Rules every core-member option (H, I) must follow

These apply whenever the compiler binds a name to a core member:

1. **Compile the generic `CallMethod` instead** for a spread call (`x.f(a...)`), for an argument count the
   entry cannot accept (the generic path raises the usual error with the usual text), and for
   `defer x.f()` (`DeferMethod`).
2. **A nil slot falls back to the name.** That covers record (`r.len()` reaches a field named `len`) and
   embedder types that answer the name in their own handler. It also covers the forward-compatibility worry: when
   a later release adds a core member, an embedder type that already had a member of that name has a nil slot
   and keeps answering through its own handler.
3. **Member slots are never default-filled.** Today `setValueType` fills every nil hook from
   `DefaultValueType`. Doing that for a member slot would let a slot added in a later release take over an
   embedder's member, the one way rule 2 can fail. Only the universal members that have existed from the
   start (`is_true`) have a default, and record opts out of even those.
4. **Calls by name still reach core members.** `CallMethodByName`, old bytecode, and the cases in rule 1
   resolve name → core ID → slot before the type's fallback.
5. **Purity and arity are declared per type's entry**, not per global ID, so a nil slot is never a folding
   candidate.

### Summary

| option | dispatch step (mono / poly) | structural fixes (§1) | opcode cost | portability | verdict |
| --- | --- | --- | --- | --- | --- |
| A string switch | 4.0 / 15.4 ns | none | 0 | as today | baseline |
| B ID switch | 3.2 / 11.8 ns | none | 0 | as today | rejected |
| C table, registration IDs | 3.0–3.8 / 13.3–13.8 ns | all | 0 | as today | good |
| **D** C + frozen builtin IDs | as C | all | 0 | as today | **recommended** |
| E registration IDs in bytecode | ≈ C − 1 load | all | 0 | broken | rejected |
| F `map[string]` | 14.5 / 29.3 ns | partial | 0 | as today | rejected |
| G inline caches | 4.1 / 4.1 ns | none | 0 | as today | rejected |
| H opcode per member | unmeasured, < 12.7 ns whole call | partial (core set only) | ~100 | tied to build, like opcodes | rejected as stated |
| I one `CallCore` opcode | ≈ D − 1 load | via D | 1 | tied to build, like opcodes | spike candidate |

## 14. Recommendation

1. **Adopt D**: this proposal's table, with frozen, append-only `const` IDs for builtin member names. The case
   for it is correctness and maintenance, not speed. Purity, arity and `_in_place` become declared data
   checked at registration, the function matrix becomes generated, and embedders get a map literal instead of
   a string switch. Speed stays within a nanosecond of today. The migration plan in §11 applies unchanged, with
   S0 also creating `core/method`.
2. **Don't adopt H as stated.** Of its ~100 opcodes, most would serve members whose body dwarfs dispatch. It
   uses up most of the remaining opcode space, and it grows the VM run loop with a cost that hits every opcode.
   The idea of choosing members because "several types have them" is the wrong filter. The members that gain
   are the ones with **tiny bodies on hot paths** (`len`, `is_empty`, `first`, `last`, `abs`, `is_true`, …),
   about 10–20 of them.
3. **After D lands, if speed is wanted, spike I and measure H against it on one member.** Implement `CallCore`
   for the 10–20 tiny-body members, plus a dedicated `LEN` opcode as the H data point. Measure with `cmd/bench`
   and the member loops in `tmp/dispatchbench`. Adopt I only if `cmd/bench` shows a gain. Adopt per-member
   opcodes only for the few members where the specialized body beats `CallCore` by a clear margin. The rules
   in §13 apply to both.
4. **Keep the hook/member boundary as stated in §13's background.** A member is promoted to a typed descriptor
   hook only when Go code needs to call it on a type it doesn't know. Everything else is a table entry, and
   entries like `len()` and `contains()` wrap the typed hook, so member↔operator parity has one implementation.

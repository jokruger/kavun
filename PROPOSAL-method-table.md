# PROPOSAL: a declarative per-type method table

Status: **proposal, not scheduled.** Written 2026-10-03 after the dispatch measurements recorded in `NOTES.md`.
Nothing here is implemented. The proposal is self-contained; where it quotes a number, the number comes from
`benchmarks/dispatch_benchmark_test.go` on the machine named in `NOTES.md`.

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
- **Open: should builtin IDs be constants?** A `core/method` package with `const Len, Copy, … MethodID` would let
  hot internal callers avoid even the package-level `var`. It also fixes the IDs of builtin names independent of
  init order. Cheap to add later; not needed for correctness.
- **Open: expose `methods()`/`has_method()` to scripts?** Out of scope here; the data makes it a small follow-up
  if the language wants it, and `TODO.md` is the place to park that.

## 13. Alternatives considered

- **Keep the string switch** (the current decision, `NOTES.md`): correct and fast; it leaves the five problems in
  §1 as review-enforced conventions.
- **`uint16` ID switch, no table**: same speed as the table, but still one function per type with a switch,
  purity still a separate predicate, embedders must intern names and write `case` over variables (no jump table
  for them). Buys the least for the same registry machinery.
- **IDs baked into bytecode**: breaks gob portability across processes with different registrations (§6).
- **Per-type `map[string]`**: measured 14.5 ns vs 4.0 ns; the slowest option and no better structurally than a
  slice indexed by ID.
- **Call-site inline caches**: only win on polymorphic selectors (about 5 ns per call measured in Kavun loops),
  need `Compiled`-owned mutable storage because `Clone` shares `*vm.Bytecode`, and add a mutable structure to a
  runtime that is otherwise read-only after load. Not worth it at Kavun's call costs.

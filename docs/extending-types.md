# Extending types: implementing operators

This is the implementor-facing contract behind every builtin and embedder `ValueTypeDescr` in
`core/*.go` — the counterpart to [Purity Contract](purity.md) for a different obligation:
**how `BinaryOp`/`UnaryOp` hooks must behave so cross-type operator dispatch stays correct and
predictable.** Read this before implementing `BinaryOp`, `UnaryOp`, or `Equal` for any new builtin
type, or before registering an embedder type via `SetValueType` (type IDs 64+) that participates in
operators at all. For what a *script author* sees as the result of any expression, see
[Type Reference](types.md)'s "Operators across types" section and each type's own page — this
document is implementation vocabulary a script author never needs.

## The problem this solves

Kavun's operators (`+ - * / % & | ^ &^ << >> < > <= >=`, unary `- ! ^`) dispatch through a
per-type `BinaryOp`/`UnaryOp` hook in `core.ValueTypeDescr`, one per builtin/embedder type, stored
in `ValueTypes[256]`. A naive lhs-only dispatch (call the left operand's hook, done) has two
recurring failure modes once more than one type wants to participate in the same pairing:

- **Asymmetric bugs.** If only the lhs type's hook can ever fire, `a op b` and `b op a` can only
  agree by accident — one side has to happen to implement the mirror image of the other's logic,
  and nothing enforces that they stay consistent. (This was a real, shipped bug before this
  mechanism existed: `byte(255) + 1` and `1 + byte(255)` used to behave differently.)
- **Implicit-conversion footguns.** A hook that calls a generic `other.AsX()` conversion instead of
  checking `other.Type` against an explicit allowlist will silently accept *any* type that happens
  to implement that conversion — including types that were never meant to participate at all.
  (Also a real, shipped bug: `stringTypeBinaryOp`'s unconditional `rhs.AsString()` meant `"a" + 5`,
  `"a" + true`, `"a" + array(...)` all silently stringified instead of erroring. A second instance
  surfaced later in `timeTypeBinaryOp`, which trusted `other.AsTime()` succeeding instead of
  checking `other.Type == value.Time` — `int`, `string`, and `runes` all implement `AsTime()` for
  unrelated reasons, so `time(...) < "2024-01-01"` silently succeeded until this was fixed.)

The three-rule dispatch model below exists to make "exactly one type owns any given pairing, and
every other type either recognizes it too via a shared narrow mechanism or explicitly opts out" a
structural property of the code, not a matter of hoping every implementor got it right.

## The three-rule dispatch model

`Value.BinaryOp(op, rhs)` (`core/value.go`) is a single-line dispatch — `ValueTypes[v.Type].BinaryOp(v,
rhs, op, false)` — exactly like every other per-type hook (`MethodCall`, `Access`, `Copy`, ...). All
three dispatch rules live inside the hooks themselves, not in `Value.BinaryOp`:

1. **Domain-specific (rule 1).** The lhs type's `BinaryOp` hook recognizes `other.Type` directly and
   implements the operation because it has a genuine, meaningful interpretation for that specific
   pairing. Hand-authored per pair; can be asymmetric (`time + int` defined, `int + time` not) or
   symmetric (`byte + int` both directions, same result either way). This is where the large
   majority of pairings live.
2. **Narrow safe-conversion (rule 2).** A small, closed mechanism reserved for pairings where "safe
   conversion" has one unambiguous meaning — pure magnitude/representation widening, no judgment
   call. A richer type declares "I safely accept this narrower type" and converts it on the way in
   (e.g. `float` accepts `int`; `rune` accepts `byte`, a lossless Latin-1 bijection; `bytes`/`runes`/
   `string` each accept their scalar). By default the narrower type must **not** also claim the same
   pairing via rule 1/2 in the other direction — it declines via rule 3, the richer type's reflected
   branch answers, and the semantics live in exactly one place. The one sanctioned exception is a
   deliberate performance mirror — see "Mirrored ownership" below.
3. **Reflected delegate (rule 3).** If a hook applies neither rule 1 nor rule 2 for this
   `other.Type`, **the hook itself** calls the other type's hook directly, with the operands
   swapped and a `reflected: true` flag, and returns whatever comes back:
   `ValueTypes[other.Type].BinaryOp(other, v, op, true)`. This gives the other type a fair shot via
   its own rules 1/2. There is no central retry point — every hook that might need rule 3 performs
   this call itself, in its own non-reflected branch, on any type/op it doesn't recognize as its
   own.

No shared rank/arbiter data structure exists anywhere in this model — every family resolves via
rule 1 and/or a couple of rule 2 declarations. See `docs/types.md` for the concrete per-family
outcomes this produces.

## The shape every hook follows

Every `BinaryOp` implementation is structured as **two separate top-level cases**, not one switch
threaded with the `reflected` flag:

```go
func xTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		// v is the original rhs (self, playing receiver); other is the original lhs. Every case
		// here computes "other OP v". This branch is TERMINAL: it either answers or returns a
		// real error — it must never delegate further.
		switch other.Type {
		// ... whatever this type answers for when it's the delegate target ...
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
		}
	}

	// v is genuinely the lhs; other is genuinely the rhs. This branch performs rules 1/2, and rule
	// 3 (delegate) on anything it doesn't recognize.
	switch other.Type {
	// ... this type's own rule 1/2 pairings ...
	default:
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}
}
```

This is deliberate, not incidental style: the set of types/ops a hook answers for genuinely
differs by direction (a non-commutative operator like `-`, or a pairing that's only ever valid one
way, means the reflected and non-reflected cases are *not* mirror images of each other — see
"non-commutative and one-directional pairings" below). Writing them as one shared switch with
`reflected` checked inline invites exactly the class of bug this shape exists to prevent: a branch
that forgets to check the flag. `core/rune.go`'s `runeTypeBinaryOp` and `core/byte.go`'s
`byteTypeBinaryOp` are the canonical examples to copy from.

Most types' reflected branch is a bare, one-line decline — nothing ever delegates into them,
because they either recognize both directions of their own pairings directly (`int`, `float` and
`decimal` each implement their own side of `int op float`/`int op decimal`, so neither ever needs
the other to answer for it — the deliberate mirror described under "Mirrored ownership" below) or
answer everything on their own non-reflected side (`bool`, `record`).
Real reflected-branch logic is only needed by types that another operand's decline can land in: the
text sequence types answer for a scalar-on-the-left (`b'a' + "bc"`, `'a' + u"bc"` — the scalar takes
the sequence's type), `array` answers `+` for anything-on-the-left by prepending it as one element
(`3 + [1, 2]` → `[3, 1, 2]`; it declines every other operator, because only the add side has a front
member to mirror), and `dict` answers for `record + dict`. In the text family the RECEIVER —
the left operand — decides the result type, so each sequence type handles every text operand in
its own NON-reflected branch (`"ab" + u"cd"` is string's cell and answers a string); a sequence
type's reflected branch therefore only ever sees scalars and comparisons. Trace who might delegate
into a given type before assuming its reflected branch needs real logic; `record`'s is a bare
decline because `dict` always answers itself and never delegates into `record`.

A reflected branch that does real work must still honour the universal contracts: `array`'s re-raises
for an `error` on the left rather than prepending it as an element, because an `error` propagates
through every operator. (`undefined` never reaches a reflected branch — it answers `undefined`
without handing over.)

### A hook must never panic

A hook signals every failure by returning an error — `errs.NewInvalidBinaryOperatorError`,
`errs.NewInvalidArgumentTypeError`, and the rest of the `errs` constructors. A Go panic is not a signalling
path: it is a defect in the host type.

The runtime contains one anyway. `VM.Run` recovers a panic from anywhere inside a run and answers a **fatal**
`internal` error carrying the panic text and the script's stack trace, so a defective hook cannot take the host
process down. Fatal means the script's own `recover()` never sees it — the script did nothing wrong and has
nothing to handle. Treat such an error as a bug report about the host type, not as a condition to catch.

The same containment applies to a slice index, a nil dereference or a type assertion inside a hook. Do not rely
on it: validate arguments and answer an error, so the failure reads as a script-level diagnosis instead of
`internal: panic: runtime error: index out of range`.

## Declining vs. delegating

There is no shared "decline" sentinel error anymore, and nothing inspects a hook's return value to
decide whether to retry — `Value.BinaryOp` doesn't see any of this, since it's a single dispatch.
"Declining" means something different depending on which of the two top-level branches you're in:

- **In the non-reflected branch**, declining a type/op combination means performing rule 3
  yourself: call `ValueTypes[other.Type].BinaryOp(other, v, op, true)` directly and `return` its
  result exactly as given — success or error, don't inspect or reformat it. The callee is
  responsible for producing the final, correctly-ordered answer.
- **In the reflected branch**, declining means the pairing is genuinely invalid — return a real,
  final `errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())`. Note the
  argument order: `other` (the original lhs) first, `v` (the original rhs, self) second — this
  reconstructs the order the caller actually wrote, since inside a reflected call `v` and `other`
  have swapped roles relative to the written expression. Never delegate further from here: the
  reflected branch is the end of the chain by construction, which is also what makes rule 3
  terminate without a loop guard — a genuinely invalid pairing bounces at most once (non-reflected
  decline → the other side's reflected branch → real error), never twice.

**One exception, an optimization, not a new mechanism:** if a hook recognizes `other.Type`
generally but knows *no* operator the delegate could support applies either — `byteTypeBinaryOp`'s
bitwise-with-`int` case is the concrete example: `int` is a type `byte` has a real relationship
with (arithmetic, ordering), but bitwise is same-type-only on both sides, so delegating to `int`
would just pay for a hop that's certain to fail — the hook may reject directly instead of
delegating. This is a hard-won judgment call about a *specific, known* type/op combination, not a
general license to skip delegation for anything that looks unlikely to work; when in doubt,
delegate.

## Mirrored ownership — the sanctioned hot-path deviation

The base rule is single ownership: one type implements a pairing, the other declines via rule 3, and
the semantics exist in exactly one place. Delegation costs one extra indirect hook call, invisible
everywhere except the arithmetic inner loop of a running script — so **a pairing may be implemented
on both sides for performance**, and the numeric family already is.

`int op float` and `int op decimal` are implemented twice: once in `intTypeBinaryOp`'s non-reflected
branch (`core/int.go`'s `value.Float`/`value.Decimal` cases) and once in
`floatTypeBinaryOp`/`decimalTypeBinaryOp`'s own `Int` cases. `int` is by far the most common lhs in
real scripts — `x * 0.05d`, `i + 1.5` — and the mirror saves the delegate hop on every one of them,
the same motivation as the `int`/`int` fast track in the VM's `OpBinaryOp`. The consequence is that
neither `float` nor `decimal` ever *receives* an `int` in its reflected branch; those branches exist
only for `bool`/`byte`/`rune` and the `decimal`↔`float` ordering bridge.

The secondary win is that no arithmetic in the numeric family is written with inverted operands. A
reflected branch has to compute `other OP v` while `v` is the receiver — cheap and obvious for
ordering (`core/decimal.go`'s reflected branch uses `r.LessThan(*l)`), but for `-`, `/` and `%` it
means every expression written backwards, in the one branch nothing but a test can check. The mirror
keeps every arithmetic implementation in the "`v` really is the lhs" orientation.

**If you mirror, these are the obligations — all of them:**

- **Both copies produce identical results, including the result type.** A mirror is an optimization,
  never a place to differ: `int + decimal` and `decimal + int` both yield `decimal`, `int / float`
  and `float / int` both yield `float`.
- **Keep the operator allowlists in lockstep.** `decimal`'s arithmetic accepts `Decimal`/`Int` and
  deliberately not `Float`; `int`'s mirror accepts exactly `Decimal`, no more and no less. If one
  side gains or loses an operator, the other changes in the same commit.
- **Cross-reference both copies in the doc comments.** Each side names the other — the comments on
  `intTypeBinaryOp`, `floatTypeBinaryOp` and `decimalTypeBinaryOp` are the pattern to copy — so
  nobody edits one half without discovering the other.
- **The mirrored type's reflected branch declines the mirrored type.** Once both sides own the
  pairing directly nothing delegates, and an unreachable case left in a reflected branch is dead
  code that invites drift. `int`'s reflected branch answers only for `bool`.
- **Test both directions in `value_binaryop_matrix_test.go`, non-commutative operators especially.**
  Test the *result*, never which side answered: which hook handled a row is precisely the
  implementation detail a mirror makes unobservable, so asserting it would only pin today's
  optimization decision in place. `a - b` vs `b - a` and `a / b` vs `b / a` are where an inverted
  mirror hides.

**Do not mirror by default.** Two copies of a pairing is two places to get it wrong, and the reason
to pay that is a hot path, not symmetry for its own sake. Everything outside the numeric family —
the sequence/text rank chain, `dict`/`record`, `byte`/`rune` against `int` — keeps single ownership,
and so does `Equal` even within the numeric family: `intTypeEqual` has no `Float`/`Decimal` case at
all and delegates up (see "`Equal`'s one-hop delegate mechanism" below). `==` has no operand order
to invert and no per-operator allowlist to keep in sync, so a mirror there would buy only the hop,
which hasn't been judged worth a second copy.

## The `reflected` flag — why it matters

When `reflected` is `true`, this call is rule 3's retry: the hook is being asked to evaluate
`originalLhs op originalRhs`, but `v` here is what was originally the *rhs*, and `other` here is
what was originally the *lhs*. For a commutative pairing this distinction doesn't matter. For a
**non-commutative** operator (`- /` and friends), it matters a great deal — from `core/rune.go`'s
actual `Int` handling, non-reflected and reflected side by side:

```go
// non-reflected: v (self) is the real lhs, other is the real rhs — "rune - int" is defined
// (offset backward, stays rune).
case value.Int:
    l, r := int64(v.Data), int64(other.Data)
    switch op {
    case token.Sub:
        return RuneValue(rune(l - r)), nil
    // ...
    }

// reflected: other is the real lhs, v (self) is the real rhs — "int - rune" has no natural
// "position minus offset" reading (contrast rune - int above), so Sub isn't among the cases this
// branch handles at all, and it falls through to the terminal decline.
case value.Int:
    l, r := int64(other.Data), int64(v.Data)
    switch op {
    case token.Add:
        return RuneValue(rune(l + r)), nil
    // no case token.Sub here — reaches `default:` and declines.
    }
```

Because the two branches are separate switches (see "The shape every hook follows" above), a
one-directional pairing like this is expressed simply by **omitting** the case from the branch
where it doesn't apply — there's no `if reflected { decline }` scattered inside a shared `Sub`
case to remember. The same pattern covers operator families with **no reflected direction at
all**: sequence removal (`-`) only ever means "remove from the lhs sequence," so
`stringTypeBinaryOp`/`bytesTypeBinaryOp`/`runesTypeBinaryOp`'s reflected branches simply have no
`token.Sub` case anywhere in their switches — any reflected call for `Sub` falls straight to the
terminal decline, by the same structural omission, not a special check.

**When writing a new hook:** if an operator is commutative and your logic is symmetric either way,
give it the same case in both branches (or compute from `other`/`v` symmetrically). If it's
non-commutative, or your type's pairing with the other side has no meaning in one particular
direction, that direction's branch simply omits the case — never assume `v op other` is what the
caller actually wrote just because you're in the reflected branch too.

## The implementor contract: never wildcard-match `undefined`

**The one hard, cross-cutting rule every implementor must uphold**, builtin or embedder alike: a
rule-1 (or rule-2) match must never be a wildcard that silently includes `undefined`.

For the common case — your rule 1 logic is an enumerated list of specific `other.Type` values
(e.g. `time` matching `int`/`time`, `byte` matching `int`/`byte`) — this is automatic and requires
no extra code: `undefined` simply isn't in your list, so your non-reflected branch naturally
delegates to it via rule 3, and `undefined`'s own hook always matches and propagates
unconditionally (`x op undefined → undefined`, for any `op` other than `==`/`!=`, regardless of
which side triggered the call):

```go
// core/undefined.go — ignores op, v, other, and reflected entirely. This is what makes it win for
// free whenever another type's BinaryOp doesn't recognize a pairing and delegates directly here.
func undefinedTypeBinaryOp(Value, Value, token.Token, bool) (Value, error) {
	return Undefined, nil
}
```

It becomes a real, explicit obligation only for a type whose rule 1 is itself phrased as a broad
catch-all over "any type I don't otherwise specifically recognize" — where declining would mean
implementing some *other* real behavior for the catch-all, not just handing off. Today, no builtin
type does this: `error` (`core/error.go`) is the type that looks like a catch-all at first glance,
but its actual contract is to have **zero** special-casing, because its "catch-all" is *always* to
delegate, never to compute a value of its own:

```go
// error has no rule-1/rule-2 pairing with anything, in either direction — every non-reflected case
// delegates unconditionally, so that undefined's own BinaryOp, which never itself declines, gets a
// chance to claim it (the "error declines to undefined" contract). The reflected branch is
// terminal like every other hook's.
func errorTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}
```

`defaultBinaryOp` (`core/tools.go`) — the fallback for any built-in slot or embedder type that
doesn't override `BinaryOp` at all — follows the identical shape for the identical reason.

If you ever *do* write a hook whose non-reflected branch wants a genuine broad "anything else"
behavior (a real computed value, not a decline-or-delegate), you must explicitly check
`other.Type == value.Undefined` and delegate before applying that behavior, so `undefined`'s
contamination guarantee ("unknown poisons every operation it touches") still holds no matter what
other types exist in the runtime.

**No other cross-type obligation exists.** A type is otherwise completely free to recognize exactly
the `other` types it chooses to via rule 1/2, and delegate (or, per the narrow exception above,
reject) everything else — there is no requirement to support any particular operator, or to be
symmetric with any other specific type.

## `Equal` is a separate, narrower mechanism — not `BinaryOp`

`==`/`!=` **never reach `BinaryOp`/`UnaryOp` at all.** The VM's `bc.Equal`/`bc.NotEqual` opcodes
dispatch to the separate `Value.Equal(rhs Value) bool` hook (`core/value.go`), and unary `!`
(`UnaryNot`) calls `Value.IsTrue()` directly — neither goes through the dispatch model above. This
is a real, easy mistake to make when writing type tests or new hooks: don't implement `==`/`!=`
semantics inside `BinaryOp`, and don't test them by calling `BinaryOp(token.Equal, ...)` — call
`.Equal()` directly.

### `Equal`'s one-hop delegate mechanism

`Equal`'s signature is `func(v Value, other Value, final bool) bool` — no error return, ever:
`==`/`!=` are total operators for the whole language, so an unrecognized `other.Type` means `false`,
never a runtime error, matching every type's existing behavior (`array() == 5`, `1 == "foo"`).

`final` is `Equal`'s counterpart to `BinaryOp`'s `reflected` flag, but simpler — equality doesn't
care which operand is which, so there's no direction to track, only "has this already had its one
chance to delegate." `Value.Equal` (the entry point) always calls in with `final: false`:

```go
func xTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case <own type>:
		return <direct comparison>
	case <types this type recognizes directly>:
		return <convert other into this type's domain, compare>
	default:
		if final {
			return false
		}
		return ValueTypes[other.Type].Equal(other, v, true)
	}
}
```

If a type doesn't recognize `other.Type` and isn't already `final`, it takes one delegate call —
`ValueTypes[other.Type].Equal(other, v, true)`, giving the other side a chance to recognize *this*
type instead — with `final: true`, so that call must decide immediately, no further delegation. This
bounds delegation to exactly one hop in either direction: at most **A asks B, B answers (or declines
to `false`)** — never a chain.

**The delegate is mandatory, not a design decision.** Every `default` arm delegates unless `final`,
whatever `other.Type` happens to be — including a type with nothing "above" it in the builtin
hierarchy, and including the container and identity-semantics types (`array`, `dict`, `record`,
`time`, `error`, `int_range`, the iterators, `compiled_function`, `format_spec`) whose only
recognized pairing is their own type. "Nothing among the builtins could answer differently" is a
closed-world argument, and `ValueTypes[256]` is open: embedder types register at IDs 64+, a
builtin's `switch other.Type` allowlist can never name a type ID that didn't exist when it was
written, and therefore **every embedder/builtin pairing must be owned by the embedder's hook and
reached through the builtin's `default` delegate** — it is the only place such a pairing can live.
Dropping the delegate doesn't just remove the pairing, it removes it *asymmetrically*: an embedder
`money` type would compare equal to `decimal` (which delegates) but not to `dict` (which wouldn't),
reintroducing exactly the commutativity bug this mechanism exists to prevent. The cost of a
guaranteed-`false` hop is one call on a comparison that was returning `false` anyway.

Types that register no `Equal` hook at all inherit `defaultEqual` (`core/tools.go`), which follows
the same rule: same-type values compare by raw `Value` identity, everything else takes the hop.

**`undefined` is the single sanctioned exception.** `undefinedTypeEqual` never delegates and equals
nothing but itself, `final` or not. "We don't know" is not a value another type is allowed to claim
equality with, however null-like it is — and letting it be claimed is precisely how
`undefined == false` used to silently succeed. Note this is `Equal` only: `undefined`'s `BinaryOp`
absorbs everything, which is the opposite behavior for the opposite reason.

This is also the one place where [Type Reference](types.md)'s "equality is always commutative"
guarantee is conditional: an embedder type whose own `Equal` claims equality with `undefined` gets
`myNull == undefined` → `true` and `undefined == myNull` → `false`, because the second order never
reaches its hook. Among the builtins the guarantee is unconditional, since none of them claims
equality with `undefined`. If you write a null-like embedder type, don't claim it.

**This mechanism is new** — before it, every builtin `Equal` was a single lhs-typed dispatch with no
delegation at all, which is *why* the builtin numeric/text types used to disagree on commutativity
(`byte(1) == float(1.0)` and `float(1.0) == byte(1)` used to give different answers, since whichever
side's hook ran was whatever happened to trust `AsX()`). The one-hop delegate model is what makes
commutativity structurally guaranteed instead of merely hoped for: exactly one side's hook computes
the real answer for any given pairing (see [Type Reference](types.md)'s equality matrix for which
builtin type owns which pairing), and the other side reaches that same computation by delegating —
they can't disagree because there's only ever one implementation actually deciding.

**The one hard rule, same as `BinaryOp`'s: never wildcard-match `undefined`/`error`.** A gate phrased
as "does `other` implement conversion X" rather than an explicit `other.Type` allowlist will leak —
`undefined` and `error` both implement almost every `AsX` conversion as a uniform
default/propagation behavior (`undefined.AsBool()` is always `false`, every `error.AsBool()` is
`true`), not because they're meaningfully equal to anything. (This is exactly the bug the Phase 6
matrix-driven test originally caught in `boolTypeEqual`, before this delegate mechanism existed:
trusting `rhs.AsBool()` unconditionally meant `undefined == false` and `error == true` both silently
succeeded.) Use an explicit `switch other.Type` allowlist, as in the shape above, and this class of
bug is structurally impossible — `undefined`/`error` simply aren't in the list, so they fall through
to the delegate (or a bare `false`), exactly like any other unrelated type.

## Hard rejection is now possible — use it sparingly

Earlier versions of this dispatch model had a hard limitation: declining always meant "let the
other operand try," with no way to reject a pairing outright. That's no longer structurally true —
because rule 3 is now performed by each hook directly rather than by a central dispatcher, a hook's
non-reflected branch *can* simply return an error instead of delegating for a type it doesn't
recognize, and nothing will retry the other side. This is exactly the "recognized type, no
supported op" optimization described above under "Declining vs. delegating" — reject directly only
when you can articulate *specifically why* delegating is certain to fail (you know the other type's
own rules, not just a guess), and delegate in every other case. A hook that hard-rejects
unrecognized types by default, "just in case," reintroduces the asymmetric-bug failure mode this
whole model exists to prevent — the other operand never gets the fair shot rule 3 is supposed to
guarantee it.

## Worked examples from the builtin types

- **Symmetric rule 1, real logic in both branches:** `byte + int → byte`, `int + byte → byte`.
  `int`'s hook declines (delegates) for any `byte`/`rune` rhs (see `core/int.go`'s allowlisted
  `default` case); `byte`'s hook implements both directions — non-reflected (`byte op int`) and
  reflected (`int op byte`, reached when `int` delegates into it) — computing the same
  wraps-mod-256 result either way, since the operator is commutative. See `core/byte.go`.
- **Asymmetric rule 1, reflected branch omits a case:** `rune ± int → rune` is owned entirely by
  `rune` (raising when the result leaves the code-point space — see `runeArithResult`); `int`
  declines (delegates) for any `rune` rhs. `rune - rune → int` and `int - rune` is a deliberate
  non-definition — `rune`'s reflected branch's `Int` case has no `token.Sub` arm at all, so it
  falls straight to that branch's terminal decline. See `core/rune.go`.
- **Rule 2 in the numeric family, mirrored on both sides:** `float` declares "I safely accept
  `int`" and `decimal` declares the same, independently — two hand-authored declarations, no shared
  rank table. Unlike every other family, each of `int`/`float`/`decimal` then implements *both*
  directions of its own pairings directly (see `core/int.go`'s `value.Float`/`value.Decimal` cases
  and `core/float.go`/`core/decimal.go`'s own `int` handling), so none of the three ever delegates
  into another for these pairings and their reflected branches decline for them. That is the
  sanctioned hot-path mirror, with the obligations it carries — see "Mirrored ownership" above.
  `float` and `decimal` deliberately do **not** accept each other for arithmetic (`0.1 + 2.5d` is a
  vm error, not a silently-wrong answer); ordering between them is defined and exact.
- **The text family: the receiver decides.** Every text operand of `+`/`-` is read as content
  encoded into the LEFT operand's representation, and the result is the left operand's type
  (`"ab" + u"cd"` → string, `u"ab" + bytes("cd")` → runes; the decode is total — an octet that is not a
  symbol becomes its escape, see `core/text_escape.go`).
  Each of `string`/`runes`/`bytes` therefore answers all text operands in its own non-reflected
  branch through one shared operand reader; the reflected branches carry only the scalar-on-left
  `+` cells (`b'a' + "bc"` → `"abc"`, ASCII-limited; `'a' + u"bc"`) and the cross-type comparisons.
  There is no reflected direction for `-` — a scalar has no content to remove from. Operand
  acceptance is an explicit allowlist, never a trusted `AsString` fallback — see `core/tools.go`'s
  `textOperandString`/`textOperandOctets` (`"a" + 5` stays a vm error; an int operand keeps its
  arithmetic reading).
- **`dict`/`record`, one-directional delegation:** `dict` is the more general of the pair and
  always wins a merge; `record`'s non-reflected branch declines (delegates) against `dict` so
  `dict`'s reflected branch — the only place this pairing is actually computed — can claim it.
  `record` never needs a reflected branch of its own beyond a bare decline, because `dict` never
  delegates into `record`. See `core/dict.go`, `core/record.go`.
- **The `undefined` contract in practice:** `core/undefined.go`'s `BinaryOp`/`UnaryOp` always match
  and unconditionally return `(Undefined, nil)` regardless of `op`/`v`/`other`/`reflected` — it
  never needs to inspect the other operand, which is what makes it "just work" whichever type
  delegates into it.
- **The catch-all contract in practice:** `core/error.go`'s `BinaryOp` needs **zero** special-casing
  for "decline to `undefined`," because its non-reflected branch *always* delegates unconditionally
  — since `undefined` never itself declines, delegating into it whenever `other.Type ==
  value.Undefined` satisfies "error yields to undefined" automatically, with no extra branch
  required.

## Extensibility for embedder types

An embedder using `SetValueType` (type IDs 64+) participates in this exact dispatch model with no
special casing — but **rule 3 is no longer automatic**. There used to be a central dispatcher that
retried the other operand for you whenever your hook declined; that mechanism is gone. If your
`BinaryOp` hook wants a pairing with some other type (built-in or another embedder type) to succeed
when *that other type* is the one that actually knows how to compute it, your hook must perform the
delegation itself:

```go
func myTypeBinaryOp(v core.Value, other core.Value, op token.Token, reflected bool) (core.Value, error) {
	if reflected {
		// answer for whatever pairings you accept when you're the delegate target, or:
		return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	switch other.Type {
	// your own rule 1/2 pairings
	default:
		return core.ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}
}
```

If you skip the `default` delegate call and return a decline error directly instead, that pairing
will simply fail — the other operand never gets a turn. That may be exactly what you want (see
"Hard rejection is now possible" above), but it's a choice you now make explicitly rather than a
guarantee the framework used to provide for free. The `undefined`-wildcard contract above applies
identically to embedder types; nothing else is required.

The same applies to `Equal`, with the simpler one-hop shape described above:

```go
func myTypeEqual(v core.Value, other core.Value, final bool) bool {
	switch other.Type {
	case MyType:
		return <direct comparison>
	// any other type you want to recognize directly
	default:
		if final {
			return false
		}
		return core.ValueTypes[other.Type].Equal(other, v, true)
	}
}
```

No error return, ever — an unrecognized type is `false`, not a panic or an error. And the same
wildcard warning applies here too: gate recognition on an explicit `other.Type` allowlist, never on
"does `other` implement some conversion I use," or `undefined`/`error` will leak in via whatever
default `AsX()` behavior they happen to expose.

**What you can rely on from the other side.** Every builtin type's `BinaryOp` delegates on any
pairing it doesn't recognize (including `defaultBinaryOp`, for the builtin slots that register no
hook of their own), and so does every builtin `Equal` — `undefined` being the one exception, for
`Equal` only. So a pairing you implement in *your* hook works in **both** written orders: `myType op
builtin` runs your non-reflected branch directly, and `builtin op myType` reaches your reflected
branch (or your `Equal` with `final: true`) through the builtin's delegate. You own every pairing
between your type and a builtin, in both directions, because a builtin's allowlist can never name
your type ID — which is also why you must handle both branches, not just the non-reflected one. The
two exceptions where a builtin rejects without delegating are `byte`/`int` bitwise and
`float`/`decimal` arithmetic; both name specific builtin types, so neither can affect an embedder
type.

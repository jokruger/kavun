# Member-function matrix

Ground truth, not aspiration: every cell below comes from grepping the `case "..."` labels inside each type's
`MethodCall` switch in `core/*.go` (refreshed 2026-08-17, against the finished value-semantics/mutation
redesign — baseline was 2026-08-15, before any of it landed), not from the prose docs in `docs/types/*.md` —
those have already drifted (e.g. `docs/types.md` currently claims `map()` is "arrays only"; `bytes.go` and
`runes.go` both implement it). Treat this file as the audit tool for "did we actually implement X for every type
in this family," not as a design decision by itself — a `✗` may be a real gap or an intentional exclusion; it
just means "go decide, don't assume."

`✓` = has a real `case` in that type's `MethodCall` switch. `—` = no such case (member-call form doesn't exist for
that type today, even if a free builtin or a Go-internal hook of the same name exists elsewhere).

## Changes since the 2026-08-15 baseline

**Correction (2026-08-17, same day as the refresh):** the first pass at this refresh used
`grep -oE 'case "[a-z_]+"'`, which only captures the *first* quoted label on a `case "a", "b":` line — it silently
missed the second name whenever two method names share one switch arm. That's exactly the shape
`copy`/`copy_shallow` and `freeze_shallow`/`freeze` use on every scalar-ish type (`case "copy", "copy_shallow":`,
`case "freeze_shallow", "freeze":`, both returning the receiver unchanged since depth/mutation are moot for a
value with no nested `Value`s and no mutating op). The first draft of this section wrongly reported both methods
as scalar-only-missing "new findings requiring a decision." Re-extracted with a pattern that expands multi-label
`case` lines (confirmed against `grep -n 'case "[a-z_]*", "[a-z_]*"' core/*.go`, which is exhaustive — no other
method-name pair shares an arm anywhere in `core/`): **both `copy_shallow` and `freeze` are already universal
across every type except `record`**, exactly like `copy`/`format`/`repeat`/`freeze_shallow`. There was no open
design question here after all — corrected below, and the false "open question" removed. Lesson for future
maintainers of this file: re-verify any `MethodCall` extraction against a script that actually calls the method
before treating a `✗` as real, per the standing project practice of confirming claims empirically rather than
trusting a single grep pass.

**Renamed the same day, separately from the correction above:** `freeze_in_place()` → `freeze_shallow()` (member
call and free builtin), because the `_in_place` suffix wrongly implied it behaves like
`append_in_place`/`splice_in_place`/`delete_in_place` — mutates the receiver's shared body, visible without
reassignment. It structurally can't: `Immutable` lives on the `Value` header, not the body, so this operation
is genuinely pure (like `copy_shallow()`) and always needs `x = x.freeze_shallow()` to have any effect. This
table (and the rest of `core/*.go`) already reflects the new name throughout, including in the correction note
above, which quotes current source using the new name even though the correction itself predates the rename.

**Also added the same day, found missing during this same audit pass:** `sort_in_place()`/`reverse_in_place()`
on `array`/`bytes`/`runes` — the only two Seq-shaped operations with no mutating twin despite
`docs/conventions.md` citing them by name as the canonical example of the `_in_place` convention (previously a
stale example describing behavior that didn't exist; now accurate). Both mutate the receiver's backing storage
directly and reject an immutable receiver, same shape as `append_in_place`/`splice_in_place`. See Table 2.

**Real correctness bugs found and fixed while wiring these new methods' `IsMethodPure` declarations:**
`array`/`bytes`/`runes`'s `IsMethodPure` only ever excluded `"append_in_place"` — stale since `"splice_in_place"`
was added later (P5-002) without updating it, and would have stayed stale for the two new methods just added
too. `dict`'s `IsMethodPure` blanket-returned `true` for *everything*, including the already-existing
`"delete_in_place"` — the same class of bug, just never caught. All three now explicitly enumerate every
mutating method name they have. Per the same reasoning as the original `append_in_place` fix (P5-001): checked
empirically whether this was live-exploitable today, and it isn't — `isFoldableExpr`'s `MethodCall` case
requires a scalar-literal receiver, which `array`/`bytes`/`runes`/`dict` composite literals never are, so
`IsMethodPure` was never actually consulted for these types regardless of what it returned. Fixed anyway as the
correct contract, not a live-bug fix, per that precedent — see `docs/purity.md`, which had the same kind of
stale claim ("array and dict do not override `IsMethodPure` at all") corrected in the same pass.

**Audited but not implemented — flagged for a decision, not resolved here:** `dedup()`/`unique()` on
`array`/`bytes`/`runes` are structurally similar to `sort`/`reverse` (same-type, same-shape, in-place-compatible
via compact-then-truncate) and could plausibly get `dedup_in_place()`/`unique_in_place()` twins by the same
reasoning — but this wasn't asked for and isn't as clearly "obviously missing" as `sort`/`reverse` were (no
existing doc already claims they exist). Recorded here rather than added speculatively.

The redesign landed six new member-function twins across the `array`/`bytes`/`runes` ("Seq-shaped") family —
`append`/`append_in_place`, `splice`/`splice_in_place`, `slice`/`slice_view`/`chunk_view`/`is_view` — plus
`dict` gaining `delete`/`delete_in_place` and `record`/`record_view`. Verified as a completeness check on the
implementation, per this file's own stated purpose:

- **`append`/`append_in_place` landed identically shaped on all three of `array`/`bytes`/`runes`** — confirmed,
  no asymmetry (the whole point of resolving P12).
- **`splice`/`splice_in_place` landed identically shaped on all three** — confirmed.
- **`slice`/`slice_view`/`chunk_view`/`is_view` landed identically shaped on all three** — confirmed.
- **`copy_shallow()` and `freeze()` are universal across every type except `record`**, same as
  `copy()`/`format()`/`repeat()`/`freeze_shallow()` — see the correction above. On every pure scalar
  (`bool`/`byte`/`rune`/`int`/`float`/`decimal`/`string`/`time`/`undefined`/`range`, plus the three callable
  types), both are deliberate no-ops sharing one switch arm with their "twin" (`copy`/`freeze_shallow`
  respectively) — genuinely redundant spellings, kept for member-call-surface consistency across every type
  rather than only where the distinction is observable.
- **Resolved 2026-08-17 (previously the one real gap this table found): `record` could not be frozen at all** —
  no member-call form (no `MethodCall` switch, same as `copy`/`delete`'s situation before those got free-function
  forms) and, unlike `copy`/`copy_shallow`/`delete`/`delete_in_place`, **no free-function form existed either**.
  Fixed by adding `freeze()`/`freeze_shallow()` as free builtins (`vm/builtins.go`), mirroring `copy`/`copy_shallow`
  exactly. No new Go-internal implementation was needed for `record` itself — `Value.Freeze()`/`Value.ToImmutable()`
  (`core/value.go`) are already fully generic over any `Value.Type`, and `record` already had a working `Copy`
  hook (needed by the free `copy()`/`copy_shallow()` builtins) and is already handled by
  `MarkImmutableDeep`'s type switch — so `freeze(some_record)` and `freeze_shallow(some_record)` work correctly,
  including deep immutability of nested fields, purely by exposing the existing generic machinery. Verified
  empirically (compiled and ran real scripts, including a nested-field deep-immutability check) before landing.
  `docs/language.md`'s "Collections and helpers" section updated to list both alongside `copy`/`copy_shallow`/
  `delete`/`delete_in_place`, with the same "why these stay free functions" rationale (record has no member
  functions at all — these are its only path).
- **`dict.record()`/`dict.record_view()`** (member methods) and the free-function-only
  **`dict()`/`dict_view()`/`record()`/`record_view()`** constructors now default to copying (shallow), with the
  `_view` forms as the explicit sharing twins (P19). `array`/`bytes`/`range` already had `.dict()`/`.record()`
  member methods before this redesign (unaffected structurally, though their copy-vs-share *default* wasn't in
  question the way dict/record's was) — they do **not** have `_view` member-call twins; the only path to a
  view-style array/bytes/range→dict/record conversion is the free `dict_view()`/`record_view()` builtins, and
  those two only accept a `dict` or `record` as input (not `array`/`bytes`/`range`) — so no such view path
  actually exists yet for those three source types. Not treated as a gap here since it was never in this
  redesign's scope; noted for whoever picks up type-conversion work next (see `TODO.md`).
- `record` remains the sole type with **zero** entries in its `MethodCall` switch — every operation on it goes
  through a free builtin (`copy`, `copy_shallow`, `delete`, `delete_in_place`, `freeze`, `freeze_shallow`,
  `format`, `dict`, `dict_view`) or the general `Access`/`Assign` hooks for its own fields. Unchanged in shape by
  this redesign (just gained two more free-function entry points); tracked as P14's own long-standing finding,
  not reopened here.

## Table 1 — Core (near-universal)

| function            | undefined | bool | int | float | decimal | rune | byte | string | runes | bytes | array | record | dict | range | time | error |
| ------------------- | --------- | ---- | --- | ----- | ------- | ---- | ---- | ------ | ----- | ----- | ----- | ------ | ---- | ----- | ---- | ----- |
| `copy()`            | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —²     | ✓    | ✓     | ✓    | ✓     |
| `copy_shallow()`    | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —²     | ✓    | ✓     | ✓    | ✓     |
| `format()`          | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —¹     | ✓    | ✓     | ✓    | ✓     |
| `repeat(n)`         | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —      | —    | —     | ✓    | —     |
| `freeze()`          | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —²     | ✓    | ✓     | ✓    | ✓     |
| `freeze_shallow()` | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —²     | ✓    | ✓     | ✓    | ✓     |

`copy`/`copy_shallow` and `freeze`/`freeze_shallow` are genuine no-op synonyms of each other on every scalar and
callable type (`bool`/`byte`/`rune`/`int`/`float`/`decimal`/`string`/`time`/`undefined`/`range`, plus
closures/functions) — a single shared `case` arm in each type's `MethodCall` switch, since "deep vs. shallow" and
"mutating vs. detaching" are both moot for a value with no nested `Value`s and nothing else can ever observe as
shared. Kept as real, separately-callable spellings anyway for member-call-surface consistency across every type.

¹ `record` has a `Format` **hook** (`recordTypeFormat`, `core/record.go:148`) used by f-strings/`format()`, but no
`case "format"` in `recordTypeMethodCall` — so `record_val.format(spec)` as a *member call* fails even though the
value participates correctly in format-spec formatting everywhere else. Worth deciding whether that's a real gap
or intentional (records have no member functions at all today by design — see Table 2's note).

² `record` has none of these as member calls (no `MethodCall` switch at all, see Table 2). All four are reachable
via free builtins of the same name instead (`copy`, `copy_shallow`, `freeze`, `freeze_shallow` — the last two
added 2026-08-17, see "Changes since baseline" above).

**Open questions this table raises**, not yet decided:
- `repeat()` is missing from `dict`/`range`/`error` while present on every other type including scalars — no
  obvious semantic reason `dict.repeat(n)` couldn't mean "n independent copies," unless it was simply never
  requested.
- Should `copy()`/`format()`/`repeat()` reach the same "truly universal except `record`" bar that
  `copy_shallow()`/`freeze()`/`freeze_shallow()` already do on `dict`/`range`/`error` where `repeat()` is
  currently missing, or is the asymmetry intentional per-function? Not decided.

## Table 2 — Sequence/collection operations

Columns limited to the types where at least one of these appears. `record` is included specifically to make its
**total absence** visible, not because it belongs in this family by any other measure.

| function             | array | bytes | runes | string | dict | range | record |
| -------------------- | ----- | ----- | ----- | ------ | ---- | ----- | ------ |
| `all(fn)`            | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `any(fn)`            | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `append(...)`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `append_in_place(...)` | ✓   | ✓     | ✓     | —      | —    | —     | —      |
| `avg()`              | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `chunk(size)`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `chunk_view(size)`   | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `contains(x)`        | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `count(fn)`          | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `dedup()`            | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `delete(key)`        | —     | —     | —     | —      | ✓    | —     | —      |
| `delete_in_place(key)`| —    | —     | —     | —      | ✓    | —     | —      |
| `filter(fn)`         | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `find(fn)`           | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `first()`            | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `flatten()`          | ✓     | —     | —     | —      | —    | —     | —      |
| `for_each(fn)`       | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `is_empty()`         | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `is_view()`          | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `join(sep)`          | ✓     | —     | ✓     | ✓      | —    | ✓     | —      |
| `keys()`             | —     | —     | —     | —      | ✓    | —     | —      |
| `last()`             | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `len()`              | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `lower()`            | —     | —     | ✓     | ✓      | —    | —     | —      |
| `map(fn)`            | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `max()`              | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `min()`              | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `partition(s)`       | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `reduce(i, fn)`      | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `reverse()`          | ✓     | ✓     | ✓     | ✓      | —    | —     | —      |
| `reverse_in_place()` | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `slice(i, j)`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `slice_view(i, j)`   | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `sort()`             | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `sort_in_place()`    | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `splice(...)`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `splice_in_place(...)` | ✓   | ✓     | ✓     | —      | —    | —     | —      |
| `split(sep)`         | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `split_lines()`      | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `sum()`              | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `trim(cutset)`       | —     | —     | ✓     | ✓      | —    | —     | —      |
| `unique()`           | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `upper()`            | —     | —     | ✓     | ✓      | —    | —     | —      |
| `values()`           | —     | —     | —     | —      | ✓    | —     | —      |

**Notable inconsistencies** (baseline findings, still true — unaffected by this redesign, since it never scoped
`string`'s or `record`'s missing Seq-shaped surface):
- **`string` is a much thinner Seq-shaped member than `bytes`/`runes`**, and now *more* visibly so — it also
  lacks every one of the redesign's six new methods (`append`/`append_in_place`/`splice`/`splice_in_place`/
  `slice_view`/`chunk_view`/`is_view`; it does gain the plain `slice(i, j)` sugar from Rule 10 as noted below).
  Some of the gap may be deliberate (string is immutable/byte-indexed, less "mutate a collection" shaped, and
  `+` already covers what `append` would do for it), but it's a real, larger-than-before gap between what the
  docs imply (`string`/`bytes`/`runes` as peer "String Types") and what exists.
- **`record` has zero member functions of any kind** — unchanged by this redesign (see Table 1's note for the
  specific consequence on `copy`/`copy_shallow`/`freeze`/`freeze_shallow`).
- `dict`/`range` get a narrow, hand-picked subset (no `filter`-adjacent aggregation on `range`, no ordering
  operations on `dict` since it's unordered) — these look intentional given each type's shape, but are recorded
  here so the intentionality is visible rather than implicit.

**Not shown as a new row, but worth noting:** `slice(i, j)` (Rule 10's plain member-function spelling, sugar over
the same now-always-copying `Slice` hook the `a[i:j]` operator uses) also landed on `string`, unlike every other
row added this pass — `string_val.slice(i, j)` works today. Left out of the table above's row list only because
it was missed during the initial table draft; recorded here rather than silently dropped. Treat `string` as
having `slice()` but none of `append`/`splice`/`slice_view`/`chunk_view`/`is_view`.

## Table 3 — Numeric-specific (classification, rounding, scale)

Unchanged by this redesign — recorded here for completeness, not re-verified beyond confirming no new rows
appeared.

| function                      | int | float | decimal | rune | byte |
| ------------------------------ | --- | ----- | ------- | ---- | ---- |
| `abs()`                        | ✓   | —     | ✓       | —    | —    |
| `sign()`                       | ✓   | ✓     | ✓       | —    | —    |
| `sqrt()`                       | —   | —     | ✓       | —    | —    |
| `is_zero()`                    | —   | —     | ✓       | —    | —    |
| `is_negative()`                | —   | —     | ✓       | —    | —    |
| `is_positive()`                | —   | —     | ✓       | —    | —    |
| `is_nan()`                     | —   | —     | ✓       | —    | —    |
| `canonical()`                  | —   | —     | ✓       | —    | —    |
| `error_details()`              | —   | —     | ✓       | —    | —    |
| `negate()`                     | —   | —     | ✓       | —    | —    |
| `next_up()` / `next_down()`    | —   | —     | ✓       | —    | —    |
| `rescale(scale)`               | —   | —     | ✓       | —    | —    |
| `round_*(scale)` (6 variants)  | —   | —     | ✓       | —    | —    |
| `scale()`                      | —   | —     | ✓       | —    | —    |
| `trunc(scale)`                 | —   | —     | ✓       | —    | —    |

**Notable inconsistency, still open:** `float` has `.sign()` but **not `.abs()`**, while `int` and `decimal` both
have `abs`. Still no evidence of an alternate spelling in `core/float.go` — still reads as a straight gap rather
than an intentional exclusion. Not touched by this redesign (never in scope); still on `TODO.md` as its own item.

## Table 4 — Member-call type-conversion matrix

Rows = source type, columns = target type reachable via `source_val.target_type_name()`. This is a distinct
concern from Tables 1-3 (data operations) — pure "can I get a `Y` out of an `X` via member-call syntax." Unchanged
by this redesign except the two cells noted in the footnote (dict/record's copy-vs-share default flipped, not a
reachability change, so no cell values actually differ from baseline).

| source →   | bool | byte | decimal | float | int | rune | string | time | array | bytes | dict | record | runes |
| ---------- | ---- | ---- | ------- | ----- | --- | ---- | ------ | ---- | ----- | ----- | ---- | ------ | ----- |
| `bool`     | ✓    | ✓    | —       | —     | ✓   | —    | ✓      | —    | —     | —     | —    | —      | —     |
| `int`      | ✓    | ✓    | ✓       | ✓     | ✓   | ✓    | ✓      | ✓    | —     | —     | —    | —      | —     |
| `float`    | —    | —    | ✓       | ✓     | ✓   | —    | ✓      | —    | —     | —     | —    | —      | —     |
| `decimal`  | —    | —    | ✓       | ✓     | ✓   | —    | ✓      | —    | —     | —     | —    | —      | —     |
| `rune`     | ✓    | ✓    | —       | —     | ✓   | ✓    | ✓      | —    | —     | —     | —    | —      | —     |
| `byte`     | ✓    | ✓    | ✓       | ✓     | ✓   | ✓    | ✓      | —    | —     | —     | —    | —      | —     |
| `string`   | ✓    | ✓    | ✓       | ✓     | ✓   | —    | ✓      | ✓    | ✓     | ✓     | ✓    | ✓      | ✓     |
| `runes`    | ✓    | ✓    | ✓       | ✓     | ✓   | —    | ✓      | ✓    | ✓     | ✓     | ✓    | ✓      | ✓     |
| `bytes`    | —    | —    | —       | —     | —   | —    | ✓      | —    | ✓     | ✓     | ✓    | ✓      | —     |
| `array`    | —    | —    | —       | —     | —   | —    | ✓      | —    | ✓     | ✓     | ✓    | ✓      | —     |
| `range`    | —    | —    | —       | —     | —   | —    | ✓      | —    | ✓     | ✓     | ✓    | ✓      | —     |
| `dict`     | —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | ✓    | ✓¹     | —     |
| `record`   | —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | —    | —      | —     |
| `time`     | ✓    | —    | —       | —     | ✓   | —    | ✓      | ✓    | —     | —     | —    | —      | —     |
| `error`    | —    | —    | —       | —     | —   | —    | ✓      | —    | —     | —     | —    | —      | —     |
| `undefined`| —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | —    | —      | —     |

¹ `dict_val.record()` now copies (shallow) by default (P19); `dict_val.record_view()` is the new sharing-twin
member call — same reachability as before, different default behavior. `record_view`/`dict_view` don't get their
own matrix cells since they're not spelled as a target *type name* — see the "Changes since baseline" section
above for the full picture, including why `array`/`bytes`/`range` can't reach a `_view`-style conversion to
`dict`/`record` even though they can reach the plain copying one.

**Notable asymmetries:** `string`/`runes` are the two "parse-from-text" hubs and can convert *to* almost
everything; `record` can convert *to* nothing (consistent with Table 2's finding — it has no member functions at
all) and can only be reached *from* `string`/`runes`/`array`/`bytes`/`range`/`dict`, never produced by a scalar
directly.

## Type-specific singleton groups (not shared with any other type)

- **`time`-only:** `day`, `hour`, `local`, `minute`, `month`, `month_name`, `nanosecond`, `second`, `unix`,
  `unix_nano`, `utc`, `week_day`, `week_day_name`, `year`, `year_day`, `zone_name`, `zone_offset`,
  `format_date`/`format_datetime`/`format_time`.
- **`error`-only:** `value`, `kind`, `is_runtime`, `is_fatal`.

---

## How to use this file

- **Docs-restructuring input:** Table 2/3 are the basis for the planned `docs/types/shared/sequence.md` and
  `.../numeric.md` family pages — each canonical function definition gets written once there; per-type pages
  keep only what's genuinely type-specific plus a link back for the shared ones.
- **Implementation-completeness tracking:** re-run the extraction whenever a new member function is added
  anywhere, and diff against this file's tables before considering that family "done" — a `✗` that should now be
  a `✓` is exactly the kind of drift this file exists to catch. **Use a pattern that expands multi-label `case`
  lines**, e.g. `awk '/func .*MethodCall\(/,/^}/' core/<type>.go | grep -oE 'case ("[a-z_]+"(, )?)+' | grep -oE
  '"[a-z_]+"'` — a naive `grep -oE 'case "[a-z_]+"'` silently drops the second name on a
  `case "a", "b":` line (this bit the 2026-08-17 refresh; see the correction note near the top of this file)
  and will reproduce the same false-gap mistake if reused as-is.
- **Not a decision document:** every `✗` flagged as "notable" above is a question, not a verdict. Resolving them
  (add the missing method? document the exclusion as intentional?) is separate follow-up work, tracked as its own
  step once this matrix itself is confirmed accurate.

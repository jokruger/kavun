# Member-function matrix

Ground truth, not aspiration: every cell below comes from grepping the `case "..."` labels inside each type's
`MethodCall` switch in `core/*.go` (verified 2026-08-15), not from the prose docs in `docs/types/*.md` — those
have already drifted (e.g. `docs/types.md` currently claims `map()` is "arrays only"; `bytes.go` and `runes.go`
both implement it). Treat this file as the audit tool for "did we actually implement X for every type in this
family," not as a design decision by itself — a `✗` may be a real gap or an intentional exclusion; it just means
"go decide, don't assume."

`✓` = has a real `case` in that type's `MethodCall` switch. `—` = no such case (member-call form doesn't exist for
that type today, even if a free builtin or a Go-internal hook of the same name exists elsewhere).

## Table 1 — Core (near-universal)

| function    | undefined | bool | int | float | decimal | rune | byte | string | runes | bytes | array | record | dict | range | time | error |
| ----------- | --------- | ---- | --- | ----- | ------- | ---- | ---- | ------ | ----- | ----- | ----- | ------ | ---- | ----- | ---- | ----- |
| `copy()`    | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —      | ✓    | ✓     | ✓    | ✓     |
| `format()`  | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —¹     | ✓    | ✓     | ✓    | ✓     |
| `repeat(n)` | ✓         | ✓    | ✓   | ✓     | ✓       | ✓    | ✓    | ✓      | ✓     | ✓     | ✓     | —      | —    | —     | ✓    | —     |

¹ `record` has a `Format` **hook** (`recordTypeFormat`, `core/record.go:148`) used by f-strings/`format()`, but no
`case "format"` in `recordTypeMethodCall` — so `record_val.format(spec)` as a *member call* fails even though the
value participates correctly in format-spec formatting everywhere else. Worth deciding whether that's a real gap
or intentional (records have no member functions at all today by design — see Table 2 note).

**Open questions this table raises**, not yet decided:
- `repeat()` is missing from `dict`/`range`/`error` while present on every other type including scalars — no
  obvious semantic reason `dict.repeat(n)` couldn't mean "n independent copies," unless it was simply never
  requested.

## Table 2 — Sequence/collection operations

Columns limited to the types where at least one of these appears. `record` is included specifically to make its
**total absence** visible, not because it belongs in this family by any other measure.

| function        | array | bytes | runes | string | dict | range | record |
| --------------- | ----- | ----- | ----- | ------ | ---- | ----- | ------ |
| `all(fn)`       | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `any(fn)`       | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `avg()`         | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `chunk(size)`   | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `contains(x)`   | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `count(fn)`     | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `dedup()`       | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `filter(fn)`    | ✓     | ✓     | ✓     | ✓      | ✓    | —     | —      |
| `find(fn)`      | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `first()`       | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `flatten()`     | ✓     | —     | —     | —      | —    | —     | —      |
| `for_each(fn)`  | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `is_empty()`    | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `join(sep)`     | ✓     | —     | ✓     | ✓      | —    | ✓     | —      |
| `keys()`        | —     | —     | —     | —      | ✓    | —     | —      |
| `last()`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `len()`         | ✓     | ✓     | ✓     | ✓      | ✓    | ✓     | —      |
| `lower()`       | —     | —     | ✓     | ✓      | —    | —     | —      |
| `map(fn)`       | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `max()`         | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `min()`         | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `partition(s)`  | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `reduce(i, fn)` | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `reverse()`     | ✓     | ✓     | ✓     | ✓      | —    | —     | —      |
| `sort()`        | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `split(sep)`    | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `split_lines()` | —     | ✓     | ✓     | ✓      | —    | —     | —      |
| `sum()`         | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `trim(cutset)`  | —     | —     | ✓     | ✓      | —    | —     | —      |
| `unique()`      | ✓     | ✓     | ✓     | —      | —    | —     | —      |
| `upper()`       | —     | —     | ✓     | ✓      | —    | —     | —      |
| `values()`      | —     | —     | —     | —      | ✓    | —     | —      |

**Notable inconsistencies:**
- **`string` is a much thinner Seq-shaped member than `bytes`/`runes`** despite `docs/types.md` grouping all
  three under "String Types" as peers: no `chunk`/`map`/`reduce`/`min`/`max`/`sum`/`avg`/`sort`/`dedup`/`unique`/
  `first`/`last`. Some of that may be deliberate (string is immutable/byte-indexed, less "mutate a collection"
  shaped), but it's a real gap between what the docs imply and what exists.
- **`record` has zero member functions of any kind** — confirmed by reading `recordTypeMethodCall`
  (`core/record.go:176`): it only ever dispatches to a field that happens to hold a callable value, never a
  builtin case. `docs/types.md` presents `record`/`dict` as close siblings ("Record and Dict Relationship"
  section in `docs/types/dict.md`), but `dict.copy()`/`dict.filter()`/etc. all work while the equivalent
  `record.*` call always fails. Whether this is permanent-by-design (records are "just data," dicts are "the
  queryable one") or an oversight is a real open question, not assumed here either way.
- `dict`/`range` get a narrow, hand-picked subset (no `filter`-adjacent aggregation on `range`, no ordering
  operations on `dict` since it's unordered) — these look intentional given each type's shape, but are recorded
  here so the intentionality is visible rather than implicit.

## Table 3 — Numeric-specific (classification, rounding, scale)

| function                     | int | float | decimal | rune | byte |
| ----------------------------- | --- | ----- | ------- | ---- | ---- |
| `abs()`                       | ✓   | —     | ✓       | —    | —    |
| `sign()`                      | ✓   | ✓     | ✓       | —    | —    |
| `sqrt()`                      | —   | —     | ✓       | —    | —    |
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

**Notable inconsistency:** `float` has `.sign()` but **not `.abs()`**, while `int` and `decimal` both have `abs`.
No evidence in `core/float.go` of an alternate spelling — this looks like a straight gap rather than an
intentional exclusion (there's no obvious reason a script author gets `float.sign()` but has to reach for a
stdlib `math` function to get an absolute value). Every other numeric-classification/rounding operation is
`decimal`-only by design — that split (exact-arithmetic type gets the full precision-sensitive toolkit,
`int`/`float` don't) reads as deliberate, not a gap.

## Table 4 — Member-call type-conversion matrix

Rows = source type, columns = target type reachable via `source_val.target_type_name()`. This is a distinct
concern from Table 1-3 (data operations) — pure "can I get a `Y` out of an `X` via member-call syntax."

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
| `dict`     | —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | ✓    | ✓      | —     |
| `record`   | —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | —    | —      | —     |
| `time`     | ✓    | —    | —       | —     | ✓   | —    | ✓      | ✓    | —     | —     | —    | —      | —     |
| `error`    | —    | —    | —       | —     | —   | —    | ✓      | —    | —     | —     | —    | —      | —     |
| `undefined`| —    | —    | —       | —     | —   | —    | —      | —    | —     | —     | —    | —      | —     |
| `bool`     | ✓    | ✓    | —       | —     | ✓   | —    | ✓      | —    | —     | —     | —    | —      | —     |

**Notable asymmetries:** `string`/`runes` are the two "parse-from-text" hubs and can convert *to* almost
everything; `record` can convert *to* nothing (consistent with Table 2's finding — it has no member functions at
all) and can only be reached *from* `string`/`runes`/`array`/`bytes`/`range`, never produced by a scalar
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
- **Implementation-completeness tracking:** re-run the `awk '/func .*MethodCall\(/,/^}/' core/<type>.go | grep -oE
  'case "[a-z_]+"'` extraction whenever a new member function is added anywhere, and diff against this file's
  tables before considering that family "done" — a `✗` that should now be a `✓` is exactly the kind of drift this
  file exists to catch.
- **Not a decision document:** every `✗` flagged as "notable" above is a question, not a verdict. Resolving them
  (add the missing method? document the exclusion as intentional?) is separate follow-up work, tracked as its own
  step once this matrix itself is confirmed accurate.

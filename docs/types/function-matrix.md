# Member-function matrix

Ground truth, not aspiration: every cell below comes from extracting the `case "..."` labels inside each
type's `MethodCall` switch in `core/*.go` (multi-label and multi-LINE `case` arms expanded — an earlier
regeneration missed labels that wrap across lines; re-verify with a probe script before trusting a `—`),
regenerated 2026-08-29 after the member-surface redesign fully landed. Treat it as the audit tool for
"does this member exist on that type" — the per-type pages in `docs/types/*.md` carry the contracts; this
file carries existence.

`✓` = a real `case` in that type's `MethodCall` switch. `—` = no member-call form on that type (a free
builtin or operator of the same meaning may still exist; `record` has NO member surface at all by design —
free builtins only — so it has no column).

`is_true` is answered once in `core/value.go` for every type (host-defined included) and therefore appears
in no switch; it is listed in its own section rather than per column.

## Universal (dispatched once in `core/value.go` for every type, host-defined included)

`is_true()` — every type, zero arguments, answers the type's truthiness; raises where truthiness is an
error state (NaN).

## Universal-by-switch

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `copy` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `freeze` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `format` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `copy_shallow` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | — | ✓ |
| `freeze_shallow` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | — | ✓ |

## Conversions (each carries the optional trailing `[default]`; on `undefined` the default is MANDATORY — the rescue answers it, and the no-default form raises *value is missing*)

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `bool` | ✓ | ✓ | ✓ | ✓ | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — | — | — |
| `int` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ | — | — | — | — |
| `float` | ✓ | ✓ | ✓ | — | — | — | ✓ | — | ✓ | ✓ | ✓ | — | — | — | — |
| `decimal` | ✓ | ✓ | ✓ | — | — | — | ✓ | — | ✓ | ✓ | ✓ | — | — | — | — |
| `byte` | ✓ | — | — | — | ✓ | ✓ | — | — | ✓ | — | — | — | — | — | — |
| `rune` | ✓ | — | — | — | ✓ | ✓ | — | — | ✓ | — | — | — | — | — | — |
| `string` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `runes` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `bytes` | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `array` | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `dict` | — | — | — | — | — | — | — | — | ✓ | — | — | — | ✓ | — | ✓ |
| `record` | — | — | — | — | — | — | — | — | ✓ | — | — | — | ✓ | — | ✓ |
| `record_view` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ |
| `range` | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ |
| `time` | ✓ | ✓ | ✓ | — | — | — | ✓ | — | ✓ | ✓ | ✓ | — | — | — | ✓ |
| `time_ms` | ✓ | — | — | — | — | — | — | — | — | — | — | — | — | — | — |
| `time_micro` | ✓ | — | — | — | — | — | — | — | — | — | — | — | — | — | — |
| `time_nano` | ✓ | — | — | — | — | — | — | — | — | — | — | — | — | — | — |
| `components` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | ✓ | — |
| `canonical` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |

## Size / membership / search

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `len` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `is_empty` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `contains` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `count` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `index` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `index_last` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `any` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `all` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `keys` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ |
| `values` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ |

## The add side

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `append` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `append_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `prepend` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `prepend_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `push` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `push_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `push_first` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `push_first_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `insert` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `insert_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `merge` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ |
| `merge_in_place` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | ✓ |

## The remove side

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `remove` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | ✓ |
| `remove_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | ✓ |
| `keep` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | ✓ |
| `keep_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | ✓ |

## Structural (text-trait sequence verbs)

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `trim` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `trim_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `trim_start` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `trim_start_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `trim_end` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `trim_end_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `remove_prefix` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `remove_prefix_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `remove_suffix` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `remove_suffix_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `has_prefix` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `has_suffix` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `replace` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `replace_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `pad_start` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `pad_start_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `pad_end` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `pad_end_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |

## Splitting / render

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `split` | — | — | ✓ | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — | — |
| `partition` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — | — |
| `split_lines` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — | — |
| `join` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — |
| `flatten` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | — | — |

## Order / dedup / slices

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `sort` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `sort_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `reverse` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `reverse_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `dedup` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `dedup_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `unique` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `unique_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `slice` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `slice_view` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `splice` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `splice_in_place` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `chunk` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `chunk_view` | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | — | — |
| `repeat` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |

## Transforms / iteration / aggregation

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `map` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | ✓ |
| `flat_map` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | — | — |
| `reduce` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `for_each` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `first` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `last` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `min` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `max` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `sum` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — |
| `avg` | — | — | — | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — |

## Casing / symbol classes (string/runes only)

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `lower` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `upper` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `case_fold` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `title_case` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `snake_case` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `kebab_case` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `camel_case` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |
| `pascal_case` | — | — | — | — | — | — | — | — | — | ✓ | ✓ | — | — | — | — |

## Text predicates

`is_valid` — every symbol is a real symbol, with no [octet escapes](string.md#undecodable-octets).
`is_ascii` — every octet/symbol is below `0x80`. `bytes` deliberately has no `is_valid`: every octet is a
valid octet, so the decode question is `b.string().is_valid()`.

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `is_valid` | — | — | — | — | — | ✓ | — | — | — | ✓ | ✓ | — | — | — | — |
| `is_ascii` | — | — | — | — | ✓ | ✓ | — | — | — | ✓ | ✓ | ✓ | — | — | — |

## Numeric predicates and float/decimal surfaces

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `is_nan` | ✓ | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `is_inf` | ✓ | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `is_zero` | — | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `is_negative` | — | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `is_positive` | — | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `abs` | ✓ | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `sign` | ✓ | ✓ | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `negate` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `sqrt` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `next_up` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `next_down` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |

## Decimal rounding / scale

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_ceiling` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_floor` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_down` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_up` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_half_down` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_half_up` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_half_even` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_to_multiple` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `round_significant` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `rescale` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `div_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `mul_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `mul_percent_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `mul_add_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `mul_div_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `sqrt_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `pow_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `nth_root_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `pow_rational_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `exp_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `ln_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `log10_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `log2_round` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `split_residual` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `allocate` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `allocate_residual` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `quo_rem` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `scale_by_pow10` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `copy_sign` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `clamp` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `is_integer` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `significant_digits` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `integer_digits` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `can_fit` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `pow` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |
| `scale` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |

`split` (decimal: `split(k, n)`) is listed with the sequence members above.

## error accessors

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `is_requirement` | — | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — |
| `is_runtime` | — | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — |
| `is_user` | — | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — |
| `kind` | — | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — |
| `value` | — | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — |
| `error_details` | — | — | ✓ | — | — | — | — | — | — | — | — | — | — | — | — |

## time accessors

| member | int | float | decimal | bool | byte | rune | time | error | undefined | string | runes | bytes | array | range | dict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `year` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `month` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `day` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `hour` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `minute` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `second` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `nanosecond` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `unix` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `unix_ms` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `unix_micro` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `unix_nano` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `month_name` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `week_day` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `week_day_name` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `year_day` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `local` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `utc` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `zone_name` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |
| `zone_offset` | — | — | — | — | — | — | ✓ | — | — | — | — | — | — | — | — |


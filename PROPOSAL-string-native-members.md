# PROPOSAL: string-native structural members (no whole-receiver decode)

Status: **proposal, not scheduled.** Written 2026-10-03 from the measurements recorded in `NOTES.md`. No
behaviour changes: every member keeps its documented contract (`docs/types/string.md`), and the existing suite is
the acceptance test. This is a performance change to `core/string.go` only.

## 1. The finding

`string` stores text as UTF-8 and answers symbol-level questions by decoding to `[]rune`. The shared sequence
engine (`core/generic_seq.go`) works on `Seq[rune]`, so `stringTypeMethodCall` materialises the whole receiver
for almost every member:

```go
case "trim", "trim_start", "trim_end", "has_prefix", "has_suffix",
	"remove_prefix", "remove_suffix", "replace", "pad_start", "pad_end":
	rs := DecodeText(*o)              // allocates len(s) runes — 4 bytes per symbol, O(n)
	seq := Seq[rune]{Elements: rs}
	return SeqStructuralMember(name, v, args,
		func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) }, // re-encodes the result
		...)
```

That is the right shape for members whose answer needs every symbol (`lower`, `reverse`, `sort`, `split`). It is
the wrong shape for members whose work is proportional to the *argument* or constant: a prefix test on a 6-byte
string decodes 6 runes, encodes the argument to runes, allocates a `[][]rune` run set, compares two runes, and
frees three allocations.

Measured (`benchmarks/dispatch_benchmark_test.go` and `tmp/dispatchbench`, i7-9750H, go1.27.1):

| member on `"abcdef"`      | ns/call | allocations | the work actually required |
| ------------------------- | ------: | ----------- | -------------------------- |
| `s.len()`                 |      13 | 0           | read the cached count (`Value.Data`) |
| `s.has_prefix("ab")`      |     145 | 3 per call, 2.5 GB over the benchmark run | compare 2 bytes |
| `s.trim()`                |     194 | 3           | scan from both ends until a non-blank symbol |
| `s.first()`               |  ~35 est. | 1         | decode one rune |

Trivial members cost 13 ns. These cost ten to fifteen times that, and the difference is entirely the decode.

## 2. Which members are affected

From `stringTypeMethodCall`, members that decode the whole receiver, classified by whether the answer needs it:

| needs every symbol — leave as is | needs the ends or the argument only — **this proposal** | needs a scan but no materialisation — extension |
| --- | --- | --- |
| `lower`, `upper`, `case_fold`, the `*_case` family, `reverse`, `sort`, `dedup`, `unique`, `map`, `flat_map`, `reduce`, `chunk`, `split`, `partition`, `keep`, `remove`, `any`, `all` with predicates, `runes`, `array` | `has_prefix`, `has_suffix`, `remove_prefix`, `remove_suffix`, `trim`, `trim_start`, `trim_end`, `pad_start`, `pad_end`, `first`, `last` | `contains`, `count`, `index`, `index_last`, `replace` with **string-run** arguments |

`min`/`max` need every symbol and stay. `len` already reads the cached count. `repeat` already works on bytes.

## 3. Why a byte-level path is exact

The text codec (`core/text_escape.go`) and UTF-8 give four properties the proposal rests on. Each one is stated
so the differential test in §6 can pin it.

**P1. A valid UTF-8 argument matches byte-wise exactly where it matches symbol-wise, against any receiver.**
UTF-8 is prefix-free per code point and a valid sequence starts with a non-continuation byte, so a byte-run equal to
a valid `p` begins and ends on decode boundaries of the receiver whatever surrounds it, and decodes to exactly
`p`'s symbols. Conversely the receiver's symbols round-trip to their own bytes (an escape rune encodes back to
its octet), so a symbol match implies the byte match. `strings.HasPrefix`, `HasSuffix`, `Index`, `Count` and
`Contains` are therefore exact for valid arguments. This holds **even when the receiver is invalid** — the only
thing that must be valid is the argument.

**P2. An invalid argument may not be matched byte-wise.** `"\xe2\x82"` as bytes is a prefix of `"€"`, but it
decodes to two escapes while `"€"` decodes to one symbol: no symbol match. Rule: an argument that is not valid
UTF-8 sends the call to the existing engine. `utf8.ValidString` on a short argument is a few nanoseconds; the
case is rare (an argument built from repaired-but-not-yet-fixed bytes).

**P3. One-symbol decoding at either end is exact for every receiver, valid or not.** Go's decoders answer
`(RuneError, 1)` for any invalid encoding in both directions, so forward and backward decoding agree on every
boundary, and the codec's rule "an undecodable octet is one escape" is applied per octet. Two helpers wrap the
standard decoders with the escape mapping:

```go
// textFirst decodes the receiver's first symbol and its width in bytes, mapping an undecodable octet to its
// escape exactly as DecodeText does. PURE by contract.
func textFirst(s string) (rune, int) {
	r, w := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && w <= 1 {
		return OctetEscapeRune(s[0]), 1
	}
	return r, w
}

// textLast is the mirror at the end. PURE by contract.
func textLast(s string) (rune, int) {
	r, w := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && w <= 1 {
		return OctetEscapeRune(s[len(s)-1]), 1
	}
	return r, w
}
```

Because the escape mapping is inside the helper, an element set that *contains* an escape rune (`s.trim(s[1])`,
trimming a stray octet) works without a fallback.

**P4. A substring result needs no re-encoding and its symbol count is known.** `remove_prefix`/`remove_suffix`/
`trim*` answer substrings of the receiver; `newStringValueCounted(sub, count)` already exists for slice paths and
skips the recount. The count is `v.Data` minus the symbols removed, which the scan has counted.

## 4. Design

One file, `core/string_text.go`, holding the byte-level implementations; `stringTypeMethodCall` routes the
affected cases there and falls back to the engine when P2 says so. The engine is untouched: array, runes and bytes
keep their behaviour and the four-type contract stays enforced by `value_binaryop_matrix_test.go` and the type
tests.

**Argument reading.** The anchored and trim members accept the same argument classes as today
(`runesEncodeMatchArg`: rune, ASCII byte, int-as-code-point, string, runes, bytes). A small reader produces the
argument as text bytes and says whether the fast path may use it:

```go
// textArg reads one match argument as UTF-8 bytes. ok=false means "not valid UTF-8 — use the engine"; the
// error cases (a callable, a non-ASCII byte, an int outside the rune domain) are the engine's to raise, so
// they also answer ok=false and let the engine produce today's exact message.
func textArg(a Value) (s string, elementClass bool, ok bool) {
	switch a.Type {
	case value.String:
		s = *(*string)(a.Ptr)
		return s, false, utf8.ValidString(s)
	case value.Rune:
		return EncodeRuneText(rune(a.Data)), true, RuneIsValid(rune(a.Data))
	case value.Byte:
		if a.Data > 0x7F { return "", true, false }
		return string(rune(a.Data)), true, true
	case value.Int:
		r, ok := a.AsRune()
		return EncodeRuneText(r), true, ok && RuneIsValid(r)
	case value.Runes:
		s = EncodeText((*Runes)(a.Ptr).Elements)      // small: the argument, not the receiver
		return s, false, utf8.ValidString(s)
	case value.Bytes:
		s = string((*Bytes)(a.Ptr).Elements)
		return s, false, utf8.ValidString(s)
	}
	return "", false, false
}
```

A string argument costs one validity scan and nothing else. The homogeneity rule (all elements or all runs) is
the engine's; the fast path only needs the *run set* reading, so for `has_prefix`/`has_suffix` it reads every
argument through `textArg` and treats each as a run (an element is a one-symbol run, which is what the engine
does too). If any argument answers `ok=false`, the whole call goes to the engine.

**Anchored members.**

```go
func stringAnchored(name string, v Value, args []Value, suffix, remove bool) (Value, bool, error) {
	s := *(*string)(v.Ptr)
	best := -1 // byte length of the longest matching run, as the engine's "best" is in symbols
	bestSyms := 0
	for _, a := range args {
		p, _, ok := textArg(a)
		if !ok {
			return Undefined, false, nil // engine
		}
		match := (!suffix && strings.HasPrefix(s, p)) || (suffix && strings.HasSuffix(s, p))
		if match && len(p) > best {
			best, bestSyms = len(p), utf8.RuneCountInString(p)
		}
	}
	if !remove {
		return BoolValue(best >= 0), true, nil
	}
	if best <= 0 {
		return v, true, nil // a miss answers the receiver; it is immutable, so no copy is needed
	}
	if suffix {
		return newStringValueCounted(s[:len(s)-best], int64(v.Data)-int64(bestSyms)), true, nil
	}
	return newStringValueCounted(s[best:], int64(v.Data)-int64(bestSyms)), true, nil
}
```

Argument-count errors keep the engine's text by being raised before the fast path (`len(args) == 0` is the one
case for the anchored members).

**Trim family.** The set is either the blank set (no arguments) or elements; a run argument raises, in the engine.

```go
func stringTrim(v Value, inSet func(rune) bool, start, end bool) Value {
	s := *(*string)(v.Ptr)
	removed := 0
	if start {
		for len(s) > 0 {
			r, w := textFirst(s)
			if !inSet(r) { break }
			s, removed = s[w:], removed+1
		}
	}
	if end {
		for len(s) > 0 {
			r, w := textLast(s)
			if !inSet(r) { break }
			s, removed = s[:len(s)-w], removed+1
		}
	}
	return newStringValueCounted(s, int64(v.Data)-int64(removed))
}
```

With arguments, `inSet` is membership in the decoded element set (a handful of runes; the engine's own element
reader can build it). With none, `inSet` is `IsBlankRune`, with an ASCII shortcut first (`IsBlankByte` on a byte
below 0x80 answers the same thing without `unicode.IsSpace`).

**`first` / `last`.** `textFirst(s)` / `textLast(s)` on a non-empty receiver; the empty case keeps
`emptySeqResult`.

**Pad family.** Width is in symbols and `v.Data` holds the count, so no decode is needed: validate `n` and the
one fill element as the engine does (`tripleFillElement`), then `strings.Repeat(fill, n-count) + s` (or the
mirror) — one allocation for the result instead of three.

**Extensions** (same properties, separate stage): `contains`/`count` with run arguments → `strings.Contains`/
`strings.Count` under P1; `index`/`index_last` with a run → `strings.Index` then `utf8.RuneCountInString(s[:i])`
for the symbol offset, O(i) with no allocation; `replace(run, run)` → `strings.ReplaceAll` when both arguments
are valid. Predicate (callable) arguments and element-set arguments keep the engine. The sentence in
`docs/types/string.md` under `index` ("materializes the runes — the documented cost of compact storage") is
amended when this stage lands.

## 5. Expected effect

The primitives were measured on the 6-byte receiver (`tmp/dispatchbench/prims_test.go`):

| primitive | ns |
| --- | ---: |
| `strings.HasPrefix` | 3.9 |
| `utf8.DecodeRuneInString` / `DecodeLastRuneInString` | 0.5 / 1.7 |
| `utf8.ValidString` | 8.7 |
| `strings.TrimFunc` with `unicode.IsSpace` | 36 |
| `DecodeText` (today's receiver decode) | 20 + allocation |

Adding the 13 ns member-call floor:

| member | today | expected | allocations |
| --- | ---: | ---: | --- |
| `has_prefix("ab")` | 145 | ~25 | 0 |
| `remove_prefix("ab")` | ~150 | ~25 | 0 (substring) |
| `trim()` | 194 | ~30 with the ASCII shortcut, ~50 without | 0 (substring) |
| `first()` / `last()` | ~35 | ~14 | 0 |
| `pad_start(10)` | 3 allocs | 1 alloc | the result only |

The win scales with receiver length: today's cost is O(n) in symbols for every call; the fast path is O(argument)
or O(removed symbols). For a 1 KB string `has_prefix` goes from microseconds to the same ~25 ns.

## 6. Tests

- The existing suite (`kavun_test.go`, string sections) pins every contract and every error text; it must pass
  unchanged.
- **Differential test**, new: for each affected member, generate receivers that mix valid text, NBSP and other
  Unicode blanks, NUL, stray octets ≥ 0x80, truncated multibyte sequences and encoded surrogates; generate
  arguments of every accepted class, valid and invalid; assert `fastPath(s, args) == engine(s, args)` on value,
  symbol count and error. This is what makes P1–P3 facts rather than reasoning.
- **Boundary-consistency test**, new: for random byte strings, walking with `textFirst` from the left and with
  `textLast` from the right both reproduce `DecodeText(s)` exactly.
- `BenchmarkMemberCall/str_has_prefix` in `benchmarks/` is the regression guard; add `str_trim` beside it.

## 7. Stages

- **S1** `has_prefix`, `has_suffix`, `remove_prefix`, `remove_suffix`, `first`, `last` — pure byte operations,
  the differential test harness is built here.
- **S2** `trim`, `trim_start`, `trim_end` — `textFirst`/`textLast` and the blank-set shortcut.
- **S3** `pad_start`, `pad_end`.
- **S4** the extensions of §4, each with its own differential cases.

Each stage is a `CHANGELOG.md` performance entry with no migration note, because nothing a script can observe
changes.

## 8. Risks

- **Exactness.** The whole proposal rests on P1–P3. The fallback on invalid arguments (P2) and the differential
  test (§6) are the guards; if a case is found where the engine and the fast path disagree, the engine is right
  by definition and the fast path narrows.
- **Two implementations of one contract.** This is the cost the four-type engine was built to avoid. It is
  accepted here only because `string`'s storage is the one of the four that is not already materialised; the
  routing is in one place (`stringTypeMethodCall`) and the engine remains the reference implementation that the
  differential test compares against.
- **Go decoder semantics.** P3 relies on the documented `(RuneError, 1)` behaviour of `utf8.DecodeRuneInString`
  and `DecodeLastRuneInString`, stable since Go 1.0; the boundary test pins it against the toolchain in use.

## 9. Alternatives considered

- **Cache the decoded runes in the string value.** The header's `Ptr` would point at `{s string; rs []rune}`
  filled lazily. Rejected: roughly fivefold memory for every string that is ever touched by a symbol member, and
  the first call still pays the O(n) decode; the win here is avoiding the O(n) altogether.
- **Store strings as `[]rune`.** Rejected: fourfold memory for all text, and it reverses the compact-storage
  decision the type page documents.
- **Make the engine generic over a symbol cursor** instead of `Seq[rune]`. Rejected: interface-driven iteration is
  slower than today's materialisation for the members that *do* need every symbol, which are the majority.

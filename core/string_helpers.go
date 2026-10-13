package core

// string's helpers: argument readers and member bodies shared by the entries of TypeString.Methods
// (core/string_members.go). Hooks and constructors stay in string.go.

import (
	"slices"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all — the same readings as on runes (see the
// runes match members; the arguments are read by runesReadMatchArgs), over the string's SYMBOLS. No _in_place
// twins: a string is immutable by construction.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match,
// counted in symbols. The argument's type selects the reading:
//   - no argument      the first / last significant symbol (not text.IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one symbol, compared with ==; a value that is not one symbol raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// stringLocate is the body of index and index_last; name is the member called, for the errors.
func stringLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	elems := text.DecodeText(*(*string)(v.Ptr))

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !text.IsBlankRune(e) {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, nil)
	}

	needle, dflt := args[0], args[1:]

	if needle.IsCallable() {
		if err := checkElemCallback(name, needle); err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := callElem(vm, needle, i, RuneValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, dflt)
	}

	if needle.Type == value.String || needle.Type == value.Runes || needle.Type == value.Bytes {
		run, ok := needle.AsRunes()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "text content", needle.TypeName())
		}
		return locatorAnswer(seq.IndexRun(elems, run, last), dflt)
	}

	enc, isElement, err := runesEncodeMatchArg(name, needle)
	if err != nil {
		return Undefined, err
	}
	if !isElement || len(enc) != 1 {
		return Undefined, seq.ErrNotOneElement(name)
	}
	idx := -1
	for i, e := range elems {
		if RuneValue(e).Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// textMatchArg is runesEncodeMatchArg's reading — text content, an element class (rune / ASCII byte / in-range
// int) or a run class (string / runes / bytes) — answered as TEXT instead of symbols, for the members that read
// the receiver in place. Same acceptance, same errors, byte for byte. A string argument costs nothing; a rune,
// runes or bytes argument is encoded (an allocation the size of the ARGUMENT — never of the receiver).
//
// Reading this text symbol by symbol (text.TextFirst/TextLast) yields the symbols runesEncodeMatchArg answers,
// with one exception the codec cannot avoid: a runes argument whose escapes spell a valid sequence once encoded
// (the escapes for E2 82 A9 read as "₩"). Such a run of symbols can never occur in a string, so matchable is
// false and the caller answers a miss without searching; for every other argument matchable is true.
// PURE by contract.
func textMatchArg(name string, a Value) (txt string, isElement bool, matchable bool, err error) {
	switch a.Type {
	case value.Rune:
		return text.EncodeRuneText(rune(a.Data)), true, true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return "", false, false, text.ErrOctetNotSymbol(name, a.Data)
		}
		return text.EncodeRuneText(rune(a.Data)), true, true, nil // ASCII: one octet
	case value.Int:
		r, ok := a.AsRune()
		if !ok {
			return "", false, false, text.ErrIntNotSymbol(name, int64(a.Data))
		}
		return text.EncodeRuneText(r), true, true, nil
	case value.String:
		return *(*string)(a.Ptr), false, true, nil
	case value.Runes:
		rs := (*Runes)(a.Ptr).Elements
		txt := text.EncodeText(rs)
		return txt, false, text.RunesAreValid(rs) || text.TextSpellsRunes(txt, rs), nil
	case value.Bytes:
		return string((*Bytes)(a.Ptr).Elements), false, true, nil
	}
	return "", false, false, text.ErrNotTextContent(name, a.TypeName())
}

// ---------------------------------------------------------------------------
// The structural members: the trim family, the anchored pair (has_/remove_ prefix/suffix), replace, and the pads.
// Every argument is text content (runesEncodeMatchArg); none of them takes a predicate.
//   - trim*            a set of single symbols, stripped repeat-while; no argument = the blank set (text.IsBlankRune)
//   - *_prefix/suffix  a homogeneous set of runs; the longest anchored run wins; remove_* removes it once
//   - replace(old, new) every occurrence of the run old, leftmost and non-overlapping; an empty old matches nothing
//   - pad_*(n[, fill]) n counts symbols; the fill is exactly one symbol, the space by default
// ---------------------------------------------------------------------------

// stringReadElementSet reads a set of single symbols (the trim family's set, the pads' fill): each argument must be
// one symbol. A text-typed argument or a function raises with refuse, the member's own statement of what it
// takes.
func stringReadElementSet(name string, args []Value, refuse string) ([]rune, error) {
	set := make([]rune, 0, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
		enc, isElement, err := runesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		if !isElement {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
		if len(enc) != 1 {
			return nil, seq.ErrNotOneElement(name)
		}
		set = append(set, enc[0])
	}
	return set, nil
}

// stringReadRunSet reads a homogeneous set of runs (the anchored pair, split, partition): an element-typed argument
// is a run of its encoding, but one call may not mix element-typed and text-typed arguments. A function raises
// with onFunction, the member's own statement of what it takes.
func stringReadRunSet(name string, args []Value, onFunction string) ([][]rune, error) {
	runs := make([][]rune, 0, len(args))
	sawElement, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		}
		run, isElement, err := runesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		if isElement {
			sawElement = true
		} else {
			sawText = true
		}
		runs = append(runs, run)
	}
	if sawElement && sawText {
		return nil, seq.ErrMixedSet(name)
	}
	return runs, nil
}

// stringTrimmed answers the receiver without its leading (start) and/or trailing (end) symbols that belong to the
// set. The name is the member called, for the errors.
func stringTrimmed(name string, v Value, args []Value, start, end bool) ([]rune, error) {
	set, err := stringReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e rune) bool {
		if len(args) == 0 {
			return text.IsBlankRune(e)
		}
		return slices.Contains(set, e)
	}
	elems := text.DecodeText(*(*string)(v.Ptr))
	lo, hi := 0, len(elems)
	if start {
		for lo < hi && inSet(elems[lo]) {
			lo++
		}
	}
	if end {
		for hi > lo && inSet(elems[hi-1]) {
			hi--
		}
	}
	return (elems[lo:hi]), nil
}

// stringAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func stringAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := stringReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := text.DecodeText(*(*string)(v.Ptr))
	best := -1
	for _, r := range runs {
		if len(r) > len(elems) || len(r) <= best {
			continue
		}
		at := elems[:len(r)]
		if suffix {
			at = elems[len(elems)-len(r):]
		}
		if slices.Equal(at, r) {
			best = len(r)
		}
	}
	return best, nil
}

// stringWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func stringWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]rune, error) {
	best, err := stringAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := text.DecodeText(*(*string)(v.Ptr))
	switch {
	case best <= 0:
		return (elems), nil
	case suffix:
		return (elems[:len(elems)-best]), nil
	default:
		return (elems[best:]), nil
	}
}

// stringReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); each
// argument is read on its own, an element being a run of one. The name is the member called, for the errors.
func stringReplaced(name string, v Value, args []Value) ([]rune, error) {
	if len(args) != 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].IsCallable() || args[1].IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	old, _, err := runesEncodeMatchArg(name, args[0])
	if err != nil {
		return nil, err
	}
	repl, _, err := runesEncodeMatchArg(name, args[1])
	if err != nil {
		return nil, err
	}
	elems := text.DecodeText(*(*string)(v.Ptr))
	out := make([]rune, 0, len(elems))
	for i := 0; i < len(elems); {
		if len(old) > 0 && i+len(old) <= len(elems) && slices.Equal(elems[i:i+len(old)], old) {
			out = append(out, repl...)
			i += len(old)
		} else {
			out = append(out, elems[i])
			i++
		}
	}
	return out, nil
}

// stringPadded answers the receiver filled at the front (start) or the end up to n symbols; a width at or below the
// length leaves it unchanged. The name is the member called, for the errors.
func stringPadded(name string, v Value, args []Value, start bool) ([]rune, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := ' '
	if len(args) == 2 {
		set, err := stringReadElementSet(name, args[1:], "one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return nil, err
		}
		fill = set[0]
	}
	elems := text.DecodeText(*(*string)(v.Ptr))
	if n <= int64(len(elems)) {
		return (elems), nil
	}
	width, err := seq.PadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]rune{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// ---------------------------------------------------------------------------
// split(...seps) and partition(...seps): a separator is a run (an element being a run of one), a homogeneous set
// of runs, a predicate on single symbols, or — no argument — the blank set (text.IsBlankRune). Runs match leftmost-longest
// and never overlap; an empty run matches nothing.
// ---------------------------------------------------------------------------

// stringPiece is one piece of split/partition, as a string.
func stringPiece(elems []rune) Value {
	return NewStringValue(text.EncodeText(elems))
}

// ---------------------------------------------------------------------------
// The callback members: map, flat_map, reduce (for_each is stringForEach, below). A per-element callback is f/1(symbol) or
// f/2(index, symbol); reduce's is f/2(acc, symbol) or f/3(acc, index, symbol).
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// The add and edit members: append, prepend, push, push_first, chunk, splice, insert (slice is the Slice hook's
// member spelling). No _in_place twins and no views: a string is immutable by construction. The add side reads
// text content: append/prepend/splice's inserts are runs (an element being a run of one), push/push_first/insert
// take exactly one symbol per argument.
// ---------------------------------------------------------------------------

// stringAddItems reads the add side's operands (append, prepend, splice's inserts): every argument is text
// content, concatenated in argument order — mixing elements and runs is fine here (x.append("ab", 'c') is
// x + "ab" + 'c').
func stringAddItems(name string, args []Value) ([]rune, error) {
	items := make([]rune, 0, len(args))
	for _, a := range args {
		enc, _, err := runesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		items = append(items, enc...)
	}
	return items, nil
}

// stringPushItems reads push/push_first/insert's elements: each argument must be exactly one symbol — a text
// argument raises even at length 1, which is the member's purpose (append/prepend take runs).
func stringPushItems(name string, args []Value) ([]rune, error) {
	return stringReadElementSet(name, args, "one element (a sequence argument never reads as an element here; append/prepend take runs)")
}

// runeSpanToByteSpan translates the rune-offset span [si, ei) into the byte-offset span of str that contains
// exactly those symbols. Offsets must already be normalized. O(1) on ASCII, one scan otherwise.
func runeSpanToByteSpan(v Value, str string, si, ei int64) (int64, int64) {
	if stringASCIIFastPath(v, str) {
		return si, ei
	}
	bs, be := int64(len(str)), int64(len(str))
	j := int64(0)
	for bi := range str { // range yields the byte offset of each rune in turn
		if j == si {
			bs = int64(bi)
		}
		if j == ei {
			be = int64(bi)
			break
		}
		j++
	}
	return bs, be
}

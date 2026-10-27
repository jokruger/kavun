package core

// string's helpers: argument readers and member bodies shared by the entries of TypeString.Methods
// (core/string_members.go). Hooks and constructors stay in string.go.

import (
	"slices"
	"strings"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all — the same readings as on runes (see the
// runes match members), over the string's SYMBOLS. contains, count and the locators read the receiver in place
// (stringReadMatchArgs answers the runs as text, matched with text.RunAt/FindRun); keep, remove, any and all
// build or visit every symbol and decode once (runesReadMatchArgs). No _in_place twins: a string is immutable
// by construction.
// ---------------------------------------------------------------------------

// stringMatch is a match member's argument list after reading, as text; exactly one reading is set.
type stringMatch struct {
	significant bool          // no argument
	pred        *ElemCallback // a predicate
	runs        []string      // contains/count: a set of runs, as text
}

// stringReadMatchArgs reads contains's and count's arguments — runesReadMatchArgs's run reading, answered as text
// (the symbol-set reading is any's/all's, which stay on runesReadMatchArgs).
// PURE by contract.
func stringReadMatchArgs(vm VM, name string, args []Value) (stringMatch, error) {
	if len(args) == 0 {
		return stringMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return stringMatch{}, seq.ErrPredicateAmongMany(name)
		}
		cb, err := newElemCallback(vm, name, args[0])
		if err != nil {
			return stringMatch{}, err
		}
		return stringMatch{pred: cb}, nil
	}

	runs := make([]string, 0, len(args))
	sawSymbol, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return stringMatch{}, seq.ErrFunctionInSet(name)
		}
		run, isSymbol, matchable, err := textMatchArg(name, a)
		if err != nil {
			return stringMatch{}, err
		}
		if isSymbol {
			sawSymbol = true
		} else {
			sawText = true
		}
		if matchable { // a run that can never occur in a string matches nothing; runs stays non-nil
			runs = append(runs, run)
		}
	}
	if sawSymbol && sawText {
		return stringMatch{}, seq.ErrMixedSet(name)
	}
	return stringMatch{runs: runs}, nil
}

// matches reports whether symbol r at index i matches the element-wise reading (significant or predicate).
func (m *stringMatch) matches(i int, r rune) (bool, error) {
	if m.significant {
		return !text.IsBlankRune(r), nil
	}
	res, err := m.pred.Call(i, RuneValue(r))
	if err != nil {
		return false, err
	}
	return res.IsTrue()
}

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match,
// counted in symbols. The argument's type selects the reading:
//   - no argument      the first / last significant symbol (not text.IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one symbol, compared with ==; a value that is not one symbol raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// stringLocate is the body of index and index_last; name is the member called, for the errors. A run is searched
// as text (text.FindRun) and its octet offset converted to the symbol offset by counting the symbols before it —
// O(position), no decode; the other readings walk the symbols one at a time.
// PURE by contract.
func stringLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
	}
	s := *(*string)(v.Ptr)

	if len(args) == 0 {
		idx := -1
		for off, i := 0, 0; off < len(s); i++ {
			r, w := text.TextFirst(s[off:])
			off += w
			if !text.IsBlankRune(r) {
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
		cb, err := newElemCallback(vm, name, needle)
		if err != nil {
			return Undefined, err
		}
		idx := -1
		for off, i := 0, 0; off < len(s); i++ {
			r, w := text.TextFirst(s[off:])
			off += w
			res, err := cb.Call(i, RuneValue(r))
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
		run, _, matchable, err := textMatchArg(name, needle)
		if err != nil {
			return Undefined, err
		}
		if !matchable {
			return locatorAnswer(-1, dflt)
		}
		at := text.FindRun(s, run, last)
		if at < 0 {
			return locatorAnswer(-1, dflt)
		}
		return locatorAnswer(text.TextRuneCount(s[:at]), dflt)
	}

	enc, isElement, err := runesEncodeMatchArg(name, needle)
	if err != nil {
		return Undefined, err
	}
	if !isElement || len(enc) != 1 {
		return Undefined, seq.ErrNotOneElement(name)
	}
	idx := -1
	for off, i := 0, 0; off < len(s); i++ {
		r, w := text.TextFirst(s[off:])
		off += w
		if r == enc[0] {
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
// set — a substring, with the symbol count carried over; the receiver itself when nothing is stripped. The ends
// are read one symbol at a time (text.TextFirst/TextLast), so the cost is the symbols removed, never the
// receiver's length; an undecodable octet is its escape here too, so a stray octet can be trimmed by passing it
// as the set (s.trim(s[0])). The name is the member called, for the errors.
// PURE by contract.
func stringTrimmed(name string, v Value, args []Value, start, end bool) (Value, error) {
	set, err := stringReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return Undefined, err
	}
	inSet := func(r rune) bool {
		if len(args) == 0 {
			return text.IsBlankRune(r)
		}
		return slices.Contains(set, r)
	}
	s := *(*string)(v.Ptr)
	lo, hi, removed := 0, len(s), 0
	if start {
		for lo < hi {
			r, w := text.TextFirst(s[lo:hi])
			if !inSet(r) {
				break
			}
			lo, removed = lo+w, removed+1
		}
	}
	if end {
		for hi > lo {
			r, w := text.TextLast(s[lo:hi])
			if !inSet(r) {
				break
			}
			hi, removed = hi-w, removed+1
		}
	}
	if removed == 0 {
		return v, nil
	}
	return newStringValueCounted(s[lo:hi], int64(v.Data)-int64(removed)), nil
}

// stringAnchored answers the longest run of the argument set anchored at the start (or, with suffix, the end) of
// the receiver — its width in octets and in symbols — or octets -1 when none is. The empty run is anchored
// everywhere. The set is read as stringReadRunSet reads it (a function raises with onFunction; mixing elements
// and runs raises after every argument was read), but argument by argument, matching as it goes: nothing is
// materialised and a miss costs the argument's length, not the receiver's. The name is the member called, for
// the errors.
// PURE by contract.
func stringAnchored(name string, v Value, args []Value, suffix bool) (octets, symbols int, err error) {
	const onFunction = "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)"
	s := *(*string)(v.Ptr)
	octets = -1
	sawElement, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return 0, 0, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		}
		p, isElement, matchable, err := textMatchArg(name, a)
		if err != nil {
			return 0, 0, err
		}
		if isElement {
			sawElement = true
		} else {
			sawText = true
		}
		if !matchable {
			continue // a run that can never occur in a string matches nothing
		}
		var o, n int
		var ok bool
		if suffix {
			o, n, ok = text.SuffixRun(s, p)
		} else {
			o, n, ok = text.PrefixRun(s, p)
		}
		if ok && o > octets {
			octets, symbols = o, n
		}
	}
	if sawElement && sawText {
		return 0, 0, seq.ErrMixedSet(name)
	}
	return octets, symbols, nil
}

// stringWithoutAnchored answers the receiver without its longest matching prefix (or suffix), removed once — a
// substring, with the symbol count carried over; unchanged (the receiver itself, which is immutable) when
// nothing matches. The name is the member called, for the errors.
// PURE by contract.
func stringWithoutAnchored(name string, v Value, args []Value, suffix bool) (Value, error) {
	octets, symbols, err := stringAnchored(name, v, args, suffix)
	if err != nil {
		return Undefined, err
	}
	if octets <= 0 {
		return v, nil
	}
	s := *(*string)(v.Ptr)
	if suffix {
		return newStringValueCounted(s[:len(s)-octets], int64(v.Data)-int64(symbols)), nil
	}
	return newStringValueCounted(s[octets:], int64(v.Data)-int64(symbols)), nil
}

// stringReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]),
// leftmost and non-overlapping; each argument is read on its own, an element being a run of one; an empty old
// matches nothing. A valid-UTF-8 old is strings.ReplaceAll (a valid run's octet matches are its symbol matches);
// one holding octet escapes is rebuilt symbol by symbol. The replacement needs no validity: its text is appended
// as is — building joins octets, so it may complete a sequence with its neighbours, as any string building does —
// and the result is recounted. The name is the member called, for the errors.
// PURE by contract.
func stringReplaced(name string, v Value, args []Value) (Value, error) {
	if args[0].IsCallable() || args[1].IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	old, _, matchable, err := textMatchArg(name, args[0])
	if err != nil {
		return Undefined, err
	}
	repl, _, _, err := textMatchArg(name, args[1]) // new is appended as text: building joins octets
	if err != nil {
		return Undefined, err
	}
	if !matchable {
		return v, nil // an old run that can never occur in a string: nothing to replace
	}
	s := *(*string)(v.Ptr)
	if len(old) == 0 {
		return v, nil
	}
	if text.TextIsValid(old) {
		return NewStringValue(strings.ReplaceAll(s, old, repl)), nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if o, _, ok := text.PrefixRun(s[i:], old); ok {
			b.WriteString(repl)
			i += o
			continue
		}
		_, w := text.TextFirst(s[i:])
		b.WriteString(s[i : i+w])
		i += w
	}
	return NewStringValue(b.String()), nil
}

// stringPadded answers the receiver filled at the front (start) or the end up to n symbols; a width at or below the
// length answers the receiver unchanged. The width is in symbols and the receiver's count is cached, so nothing is
// decoded: the result is built in one allocation, the fill (one symbol — an escape is one octet) repeated
// width-count times around the receiver's text. A valid fill symbol is complete and cannot join its neighbours, so
// the result has exactly width symbols; an escape fill can complete a sequence with the receiver's edge octets
// (a string is its octets), so that result is recounted. The name is the member called, for the errors.
// PURE by contract.
func stringPadded(name string, v Value, args []Value, start bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	fill := ' '
	if len(args) == 2 {
		set, err := stringReadElementSet(name, args[1:], "one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return Undefined, err
		}
		fill = set[0]
	}
	count := int64(v.Data)
	if n <= count {
		return v, nil
	}
	width, err := seq.PadWidth(name, n) // the ceiling is checked BEFORE any allocation (makeslice panics, this raises)
	if err != nil {
		return Undefined, err
	}
	s := *(*string)(v.Ptr)
	fillText := text.EncodeRuneText(fill)
	var b strings.Builder
	b.Grow(len(s) + (width-int(count))*len(fillText))
	if !start {
		b.WriteString(s)
	}
	for i := int(count); i < width; i++ {
		b.WriteString(fillText)
	}
	if start {
		b.WriteString(s)
	}
	if text.IsEscapeRune(fill) {
		return NewStringValue(b.String()), nil
	}
	return newStringValueCounted(b.String(), int64(width)), nil
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

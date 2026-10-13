package text

import (
	"fmt"
	"strings"

	"github.com/jokruger/kavun/errs"
)

// The three refusals of the TEXT-CONTENT reading — a member argument that is text content as symbols
// (core's runesEncodeMatchArg) or as text (core's textMatchArg) — built in one place so every reader raises
// byte-identical messages. name is the member called; typeName the offending argument's type name.

// ErrOctetNotSymbol: a byte argument above ASCII, which is not one symbol.
func ErrOctetNotSymbol(name string, b uint64) error {
	return errs.NewInvalidValueError(fmt.Sprintf("(%s) an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", name, b))
}

// ErrIntNotSymbol: an int argument that is not a code point in the rune domain.
func ErrIntNotSymbol(name string, i int64) error {
	return errs.NewInvalidValueError(fmt.Sprintf("(%s) an int reads as one symbol and must be a valid code point, got %d", name, i))
}

// ErrNotTextContent: an argument that is not text content at all.
func ErrNotTextContent(name, typeName string) error {
	return errs.NewInvalidArgumentTypeError(name, "argument", "text content (symbols, octets, or text)", typeName)
}

// TextSpellsRunes reports whether s reads back, symbol by symbol, as exactly rs. It is false only when escapes in
// rs spell a valid sequence once encoded (the escapes for E2 82 A9 read as "₩"): such a run of symbols can never
// occur in any text, because text always reads those octets as the one symbol. No allocation.
// PURE by contract.
func TextSpellsRunes(s string, rs []rune) bool {
	for _, want := range rs {
		r, w := TextFirst(s)
		if w == 0 || r != want {
			return false
		}
		s = s[w:]
	}
	return len(s) == 0
}

// PrefixRun reports whether the text p, read symbol by symbol, is a prefix of s: the width of the match in
// octets and in symbols, and whether it matched. Both sides are read with TextFirst, so an undecodable octet in
// either is its escape and matches only itself — exactly the symbol reading, with no decode of the receiver and
// no validity test of the argument. The empty p matches with width 0.
// PURE by contract.
func PrefixRun(s, p string) (octets, symbols int, ok bool) {
	for len(p) > 0 {
		if len(s) == 0 {
			return 0, 0, false
		}
		rs, ws := TextFirst(s)
		rp, wp := TextFirst(p)
		if rs != rp {
			return 0, 0, false
		}
		s, p = s[ws:], p[wp:]
		octets, symbols = octets+ws, symbols+1
	}
	return octets, symbols, true
}

// SuffixRun is PrefixRun at the end, reading both sides with TextLast.
// PURE by contract.
func SuffixRun(s, p string) (octets, symbols int, ok bool) {
	for len(p) > 0 {
		if len(s) == 0 {
			return 0, 0, false
		}
		rs, ws := TextLast(s)
		rp, wp := TextLast(p)
		if rs != rp {
			return 0, 0, false
		}
		s, p = s[:len(s)-ws], p[:len(p)-wp]
		octets, symbols = octets+ws, symbols+1
	}
	return octets, symbols, true
}

// RunAt answers the width in octets of the longest run in runs that matches s at its start, 0 when none does —
// seq.RunLengthAt on text. Longest wins so a set of runs is a true SET; the empty run never matches here.
// PURE by contract.
func RunAt(s string, runs []string) int {
	best := 0
	for _, p := range runs {
		if len(p) <= best {
			continue
		}
		if o, _, ok := PrefixRun(s, p); ok && o > best {
			best = o
		}
	}
	return best
}

// FindRun answers the octet offset of the first (or, with last, the final) occurrence of the run p in s, or -1.
// The empty run is never found: a locator needs a position. A valid-UTF-8 p is searched with
// strings.Index/LastIndex — a valid run starts with a lead octet, so an octet match can only begin on a symbol
// boundary of s and is a symbol match whatever s holds; any other p (one holding octet escapes) is tried at
// every symbol boundary with PrefixRun. A symbol match implies an octet match, so p longer than s is a miss.
// PURE by contract.
func FindRun(s, p string, last bool) int {
	if len(p) == 0 || len(p) > len(s) {
		return -1
	}
	if TextIsValid(p) {
		if last {
			return strings.LastIndex(s, p)
		}
		return strings.Index(s, p)
	}
	idx := -1
	for i := 0; i+len(p) <= len(s); {
		if _, _, ok := PrefixRun(s[i:], p); ok {
			idx = i
			if !last {
				break
			}
		}
		_, w := TextFirst(s[i:])
		i += w
	}
	return idx
}

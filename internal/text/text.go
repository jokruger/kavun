package text

import (
	"bytes"
	"slices"
	"unicode"
	"unicode/utf8"
)

// MapRunesCase applies a per-symbol case mapping, leaving the octet escapes alone — an octet that is not a
// symbol has no case, and Go's strings.ToUpper/ToLower would replace it with U+FFFD on the way through.
// PURE by contract.
func MapRunesCase(rs []rune, f func(rune) rune) []rune {
	out := make([]rune, len(rs))
	for i, r := range rs {
		if IsEscapeRune(r) {
			out[i] = r
			continue
		}
		out[i] = f(r)
	}
	return out
}

// ByteSymbolString is a byte's TEXT CONTENT — the one-octet text that holds it. TOTAL: below 0x80 that text
// is the octet's ASCII symbol, and at or above it the text holds the octet itself, which reads back as the
// octet's escape and converts back to this same byte. The old Latin-1 leak the partial version guarded against
// (equality reading "\xFF" as "ÿ") cannot happen now: "\xFF" is one ESCAPE symbol, never U+00FF, so
// b'\xff' == "\xff" is true and b'\xff' == "ÿ" is false, which is what both should be.
// The RENDER of a byte stays its number (format(b) -> "65").
func ByteSymbolString(b byte) (string, bool) {
	return string([]byte{b}), true
}

// IsBlankRune: the symbol types' blank set — NUL plus Unicode whitespace. The blank set is one notion,
// NUL ∪ whitespace, projected into each receiver's ELEMENT DOMAIN: symbols (string/runes) take the Unicode
// White_Space class, octets (bytes) the ASCII subset — all the whitespace an octet can express — so the two
// sets agree everywhere both domains overlap and neither type imports the other's limitation.
func IsBlankRune(r rune) bool {
	return r == 0 || unicode.IsSpace(r)
}

// IsBlankByte is the octet projection of the same notion: NUL plus ASCII whitespace, a fixed set of literal
// octets — deciding anything wider would require decoding, which octets never get.
func IsBlankByte(b byte) bool {
	switch b {
	case 0, ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// SplitBytesWhitespace splits bs on runs of ASCII whitespace, dropping empty
// pieces. Equivalent to bytes.Fields.
func SplitBytesWhitespace(bs []byte) [][]byte {
	return bytes.Fields(bs)
}

// SplitLinesString splits s on \n, \r\n or \r. A trailing line terminator
// does not produce an extra empty trailing element. Empty s yields nil.
func SplitLinesString(s string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, 8)
	i := 0
	start := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case '\n':
			out = append(out, s[start:i])
			i++
			start = i
		case '\r':
			out = append(out, s[start:i])
			i++
			if i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
		default:
			i++
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// SplitLinesBytes is the []byte counterpart of SplitLinesString.
func SplitLinesBytes(bs []byte) [][]byte {
	if len(bs) == 0 {
		return nil
	}
	out := make([][]byte, 0, 8)
	i := 0
	start := 0
	for i < len(bs) {
		c := bs[i]
		switch c {
		case '\n':
			out = append(out, bs[start:i])
			i++
			start = i
		case '\r':
			out = append(out, bs[start:i])
			i++
			if i < len(bs) && bs[i] == '\n' {
				i++
			}
			start = i
		default:
			i++
		}
	}
	if start < len(bs) {
		out = append(out, bs[start:])
	}
	return out
}

// AppendJSONString encodes given string as JSON string according to
// https://www.json.org/img/string.png
// Implementation is inspired by https://github.com/json-iterator/go
func AppendJSONString(b []byte, val string) []byte {
	valLen := len(val)
	buf := bytes.NewBuffer(b)
	buf.WriteByte('"')

	// write string, the fast path, without utf8 and escape support
	i := 0
	for ; i < valLen; i++ {
		c := val[i]
		if c > 31 && c != '"' && c != '\\' {
			buf.WriteByte(c)
		} else {
			break
		}
	}
	if i == valLen {
		buf.WriteByte('"')
		return buf.Bytes()
	}
	encodeStringSlowPath(buf, i, val, valLen)
	buf.WriteByte('"')
	return buf.Bytes()
}

// encodeStringSlowPath is ported from Go 1.14.2 encoding/json package.
// U+2028 U+2029 JSONP security holes can be fixed with addition call to
// json.html_escape() thus it is removed from the implementation below.
// Note: Invalid runes are not checked as they are checked in original
// implementation.
func encodeStringSlowPath(buf *bytes.Buffer, i int, val string, valLen int) {
	start := i
	for i < valLen {
		if b := val[i]; b < utf8.RuneSelf {
			if safeSet[b] {
				i++
				continue
			}
			if start < i {
				buf.WriteString(val[start:i])
			}
			buf.WriteByte('\\')
			switch b {
			case '\\', '"':
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteString(`u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		i++
		continue
	}
	if start < valLen {
		buf.WriteString(val[start:])
	}
}

// safeSet holds the value true if the ASCII character with the given array position can be represented inside a JSON string without any further escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

var hex = "0123456789abcdef"

// CaseSegmentWritten splits text on the WRITTEN boundaries only — runs of whitespace, '_' and '-',
// discarded, empty words dropped. The label rendering (title_case) segments with this: a label preserves the
// author's emphasis, and that covers word boundaries the author did not write — case transitions stay inside
// words ("iPhone" → "IPhone", "hELLO" → "HELLO"). The identifier renderings use the full segmenter below.
func CaseSegmentWritten(rs []rune) [][]rune {
	var words [][]rune
	var cur []rune
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			if len(cur) > 0 {
				words = append(words, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		words = append(words, cur)
	}
	return words
}

// CaseSegmentWords splits text into words for the *_case members. The boundary set is CLOSED:
//   - runs of whitespace, '_' and '-' delimit words and are discarded;
//   - a lower→upper transition starts a word (helloWorld → hello|World);
//   - in an upper run followed by a lower, the last upper starts the word
//     (HTTPServer → HTTP|Server, parseXMLFile → parse|XML|File);
//   - everything else — digits, apostrophes, periods — stays inside the word,
//     which is what keeps "don't" one word (an OPEN boundary set is how Go's
//     deprecated strings.Title produced "Don'T");
//   - empty words are dropped.
func CaseSegmentWords(rs []rune) [][]rune {
	var words [][]rune
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, cur)
			cur = nil
		}
	}
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			flush()
			continue
		}
		if len(cur) > 0 {
			prev := cur[len(cur)-1]
			switch {
			case unicode.IsLower(prev) && unicode.IsUpper(r):
				flush()
			case unicode.IsUpper(prev) && unicode.IsLower(r) && len(cur) >= 2 && unicode.IsUpper(cur[len(cur)-2]):
				last := prev
				cur = cur[:len(cur)-1]
				flush()
				cur = []rune{last}
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

// CaseJoinLower renders words lowercased and joined by sep: snake_case ('_') and kebab_case ('-').
func CaseJoinLower(words [][]rune, sep rune) []rune {
	out := make([]rune, 0, 16)
	for wi, w := range words {
		if wi > 0 {
			out = append(out, sep)
		}
		for _, r := range w {
			out = append(out, unicode.ToLower(r))
		}
	}
	return out
}

// CaseJoinCapitalized renders words lowercased with each word's first symbol uppercased, joined with no
// separator: pascal_case, and camel_case when firstLower keeps the very first symbol lowercase.
func CaseJoinCapitalized(words [][]rune, firstLower bool) []rune {
	out := make([]rune, 0, 16)
	for wi, w := range words {
		for i, r := range w {
			if i == 0 && !(firstLower && wi == 0) {
				out = append(out, unicode.ToUpper(r))
			} else {
				out = append(out, unicode.ToLower(r))
			}
		}
	}
	return out
}

// CaseJoinTitle renders words joined by spaces with each word's first symbol uppercased and the rest kept as
// written: title_case.
func CaseJoinTitle(words [][]rune) []rune {
	out := make([]rune, 0, 16)
	for wi, w := range words {
		if wi > 0 {
			out = append(out, ' ')
		}
		for i, r := range w {
			if i == 0 {
				out = append(out, unicode.ToUpper(r))
			} else {
				out = append(out, r)
			}
		}
	}
	return out
}

// FoldRuneCanonical maps a symbol to one canonical member of its simple case-folding orbit — the smallest
// LOWERCASE member, or the smallest member when the orbit has none — so a.case_fold() == b.case_fold() answers
// exactly Go's strings.EqualFold while the render reads like every mainstream casefold ("Hello" → "hello";
// S/s/ſ all → s). The representative must come from INSIDE the orbit: naively lowercasing the minimum would
// merge İ (an orbit of one) into i's orbit and break the equality. A TRANSFORM, which composes (a dict key, a
// sort basis, a dedup argument) where a comparison predicate cannot; not the same as lowering — the fold
// predicates differ from lower-comparison in both directions (ſ/s fold-equal but lower-unequal; İ/i the reverse).
func FoldRuneCanonical(r rune) rune {
	minAll := r
	minLower := rune(-1)
	f := r
	for {
		if f < minAll {
			minAll = f
		}
		if unicode.IsLower(f) && (minLower < 0 || f < minLower) {
			minLower = f
		}
		f = unicode.SimpleFold(f)
		if f == r {
			break
		}
	}
	if minLower >= 0 {
		return minLower
	}
	return minAll
}

// RemoveAllBytes is `-` on an octet receiver: every occurrence of the run, leftmost non-overlapping;
// the empty run removes nothing.
func RemoveAllBytes(l, r []byte) []byte {
	if len(r) == 0 {
		return slices.Clone(l)
	}
	return bytes.ReplaceAll(l, r, nil)
}

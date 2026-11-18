package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/kavun/internal/text"
)

// The rune `'…'` and runes `u"…"` literals accept one spelling Go's strconv refuses: `\udcNN` (or `\U0000DCNN`),
// NN = 80..ff, the octet escape rune U+DCNN (internal/text/escape.go). Every other escape reads as strconv reads
// it, so these two unquoters handle the escape-rune spelling themselves and hand everything else to strconv.

// escapeRuneEscape reads a `\uXXXX` / `\UXXXXXXXX` escape at the start of s. ok is true only when it is
// well-formed and names an escape rune (U+DC80–U+DCFF); n is its length in s.
func escapeRuneEscape(s string) (r rune, n int, ok bool) {
	if len(s) < 2 || s[0] != '\\' {
		return 0, 0, false
	}
	switch s[1] {
	case 'u':
		n = 6
	case 'U':
		n = 10
	default:
		return 0, 0, false
	}
	if len(s) < n {
		return 0, 0, false
	}
	var x rune
	for i := 2; i < n; i++ {
		d := digitVal(rune(s[i]))
		if d >= 16 {
			return 0, 0, false
		}
		x = x*16 + rune(d)
	}
	if !text.IsEscapeRune(x) {
		return 0, 0, false
	}
	return x, n, true
}

// unquoteRuneLit unquotes the contents of a rune literal (between the quotes): an escape-rune spelling answers
// that escape rune, anything else is strconv.UnquoteChar's reading.
func unquoteRuneLit(body string) (rune, error) {
	if r, n, ok := escapeRuneEscape(body); ok && n == len(body) {
		return r, nil
	}
	code, _, _, err := strconv.UnquoteChar(body, '\'')
	return code, err
}

// unquoteRunesLit unquotes a runes literal `"…"` (the leading u already stripped). Every escape except the
// escape-rune spelling produces octets, exactly as in a string literal, and each maximal run of octets is decoded
// with text.DecodeText (so `\xc3\xa9` reads as one symbol and a lone `\xff` as its escape). An escape-rune
// spelling is appended as that rune between the runs — never fused with its neighbours.
func unquoteRunesLit(lit string) ([]rune, error) {
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		return nil, strconv.ErrSyntax
	}
	s := lit[1 : len(lit)-1]
	out := make([]rune, 0, len(s))
	var octets []byte
	flush := func() {
		if len(octets) > 0 {
			out = append(out, text.DecodeOctets(octets)...)
			octets = octets[:0]
		}
	}
	for len(s) > 0 {
		if s[0] != '\\' {
			if s[0] == '"' || s[0] == '\n' {
				return nil, strconv.ErrSyntax
			}
			octets = append(octets, s[0]) // source octets as written: the decode reads them with the run
			s = s[1:]
			continue
		}
		if r, n, ok := escapeRuneEscape(s); ok {
			flush()
			out = append(out, r)
			s = s[n:]
			continue
		}
		code, multibyte, tail, err := strconv.UnquoteChar(s, '"')
		if err != nil {
			return nil, err
		}
		if multibyte { // \u / \U: the symbol's UTF-8 octets
			octets = utf8.AppendRune(octets, code)
		} else { // \xNN, octal, the short escapes: one octet
			octets = append(octets, byte(code))
		}
		s = tail
	}
	flush()
	return out, nil
}

// errByteLitOctets refuses a byte literal whose content is not exactly one octet.
type errByteLitOctets struct{ octets []byte }

func (e errByteLitOctets) Error() string {
	var b strings.Builder
	for _, o := range e.octets {
		fmt.Fprintf(&b, `\x%02x`, o)
	}
	return fmt.Sprintf(`a byte literal is one octet; write \xNN (this one is %d octets: %s)`, len(e.octets), b.String())
}

// unquoteByteLit unquotes the contents of a byte literal (between the quotes), read as `b"…"` reads its
// content — a character contributes its UTF-8 octets, `\xNN` and octal one octet, `\u`/`\U` the UTF-8 of the code
// point, a short escape its octet — and answers the one octet it must come to. Anything else is an error: an
// errByteLitOctets when the content is well-formed but not one octet long.
func unquoteByteLit(body string) (byte, error) {
	var octets []byte
	for s := body; len(s) > 0; {
		if s[0] != '\\' {
			octets = append(octets, s[0]) // source octets as written
			s = s[1:]
			continue
		}
		code, multibyte, tail, err := strconv.UnquoteChar(s, '\'')
		if err != nil {
			return 0, err
		}
		if multibyte {
			octets = utf8.AppendRune(octets, code)
		} else {
			octets = append(octets, byte(code))
		}
		s = tail
	}
	if len(octets) != 1 {
		return 0, errByteLitOctets{octets}
	}
	return octets[0], nil
}

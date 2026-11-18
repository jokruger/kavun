package text

import (
	"strconv"
	"strings"
)

// The quoted renders of `rune`, `runes` and `byte` are SOURCE TEXT: each reads back, as a literal, to the value
// it came from. Go's strconv cannot do that for an octet escape — it renders U+DC80–U+DCFF as U+FFFD, and the
// octets of adjacent escapes re-read as the one symbol they spell — so an escape is written `\udcNN` (lowercase
// hex, as Python's repr writes it), the one spelling `'…'` and `u"…"` accept for it.

// appendEscapeRuneLiteral writes the `\udcNN` spelling of an escape rune.
func appendEscapeRuneLiteral(b *strings.Builder, r rune) {
	const hexDigits = "0123456789abcdef"
	b.WriteString(`\udc`)
	b.WriteByte(hexDigits[(r>>4)&0xF])
	b.WriteByte(hexDigits[r&0xF])
}

// QuoteRune renders r as a rune literal that reads back to r: an escape as `'\udcNN'`, every other rune as
// strconv.QuoteRune renders it.
// PURE by contract.
func QuoteRune(r rune) string {
	if !IsEscapeRune(r) {
		return strconv.QuoteRune(r)
	}
	var b strings.Builder
	b.WriteByte('\'')
	appendEscapeRuneLiteral(&b, r)
	b.WriteByte('\'')
	return b.String()
}

// QuoteRunes renders rs as a runes literal (`u"…"`) that reads back to rs, rune for rune: an escape as
// `\udcNN`, which the literal inserts as that rune and never fuses with its neighbours; every other run as
// strconv.Quote renders it. rs must hold only runes in the domain (RuneInDomain), as every runes value does.
// PURE by contract.
func QuoteRunes(rs []rune) string {
	var b strings.Builder
	b.Grow(len(rs) + 3)
	b.WriteString(`u"`)
	start := 0
	flush := func(end int) {
		if end > start {
			q := strconv.Quote(string(rs[start:end]))
			b.WriteString(q[1 : len(q)-1])
		}
	}
	for i, r := range rs {
		if IsEscapeRune(r) {
			flush(i)
			appendEscapeRuneLiteral(&b, r)
			start = i + 1
		}
	}
	flush(len(rs))
	b.WriteByte('"')
	return b.String()
}

// QuoteByte renders o as a byte literal that reads back to o: `b'A'` for printable ASCII, the short escape
// (`b'\n'`, `b'\\'`, a quote as backslash-quote, …) where one exists, `b'\xNN'` (lowercase hex) for every other octet.
// PURE by contract.
func QuoteByte(o byte) string {
	const hexDigits = "0123456789abcdef"
	var body string
	switch o {
	case '\a':
		body = `\a`
	case '\b':
		body = `\b`
	case '\f':
		body = `\f`
	case '\n':
		body = `\n`
	case '\r':
		body = `\r`
	case '\t':
		body = `\t`
	case '\v':
		body = `\v`
	case '\'':
		body = `\'`
	case '\\':
		body = `\\`
	default:
		if o >= 0x20 && o < 0x7F {
			body = string(rune(o))
		} else {
			body = `\x` + string(hexDigits[o>>4]) + string(hexDigits[o&0xF])
		}
	}
	return "b'" + body + "'"
}

package reference

import (
	"strings"
	"unicode"
)

// Slug answers the anchor GitHub gives a Markdown heading, the fragment a link to it names. Its input is the
// heading's text — without its `#` markers, its inline markup already stripped (a heading written with `splice` in
// code is "splice"; Slug drops a backtick left in it, as any punctuation). The text is trimmed and lower-cased,
// every space becomes a hyphen, and every character that is not a letter, a mark, a decimal digit, a connector
// punctuation (`_`, `‿`, …) or a hyphen is dropped — so "²", "½" and "Ⅳ" are dropped too: "Rate and charge
// tables" → "rate-and-charge-tables".
//
// The suffix GitHub adds to the second heading of the same slug on a page ("-1") is the page's business, not the
// heading's: a heading whose slug another one already has is a link target only by that suffix.
func Slug(heading string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r == ' ':
			sb.WriteByte('-')
		case r == '-', unicode.IsLetter(r), unicode.IsMark(r), unicode.IsDigit(r), unicode.Is(unicode.Pc, r):
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

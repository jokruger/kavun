package format

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// FormatStringLike implements the shared rendering for `string`, `runes` and `bytes` per the format mini-language.
//
// raw — the underlying text payload.
// byteUnits — true for `bytes` (precision counts bytes), false for `string` / `runes` (precision counts runes).
//
// Recognized verbs: empty/'s' raw text; 'v' source form; 'q' Kavun-quoted string; 'b' / 'B' standard / URL-safe base64;
// 'x' / 'X' lower / upper hex of the underlying bytes; 'u' percent-encoded URL component (RFC 3986 unreserved set).
//
// Sign / Grouping / ZeroPad / CoerceZero / Bare are parse errors. Precision truncates the source before encoding.
// A non-empty Tail (with a generic verb) is also rejected — string-like types do not accept verb-with-tail combinations.
// Default alignment is AlignLeft.
// Verb 'v' must be processed by caller.
func FormatStringLike(typeName string, sp fspec.FormatSpec, raw string, byteUnits bool) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(typeName, sp)
	}
	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(typeName, sp)
	}

	src := raw
	if sp.HasPrec {
		if sp.Precision < 0 {
			return "", errs.NewUnsupportedFormatSpec(typeName, sp)
		}
		n := int(sp.Precision)
		if byteUnits {
			if n > len(src) {
				n = len(src)
			}
			src = src[:n]
		} else {
			i, cnt := 0, 0
			for cnt < n && i < len(src) {
				_, w := utf8.DecodeRuneInString(src[i:])
				i += w
				cnt++
			}
			src = src[:i]
		}
	}

	var body string
	switch sp.Verb {
	case 0, 's':
		body = src

	case 'q':
		body = strconv.Quote(src)

	case 'b':
		body = base64.StdEncoding.EncodeToString([]byte(src))

	case 'B':
		body = base64.RawURLEncoding.EncodeToString([]byte(src))

	case 'x':
		body = hex.EncodeToString([]byte(src))

	case 'X':
		body = strings.ToUpper(hex.EncodeToString([]byte(src)))

	case 'u':
		body = PercentEncodeComponent(src)

	default:
		return "", errs.NewUnsupportedFormatSpec(typeName, sp)
	}

	return fspec.ApplyGenerics(body, sp, fspec.AlignLeft), nil
}

// PercentEncodeComponent encodes s as an RFC 3986 URL component: bytes outside the unreserved set
// (A-Z / a-z / 0-9 / '-' / '_' / '.' / '~') are percent-encoded. Equivalent to JavaScript's encodeURIComponent.
func PercentEncodeComponent(s string) string {
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'A' && c <= 'Z'),
			(c >= 'a' && c <= 'z'),
			(c >= '0' && c <= '9'),
			c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0F])
		}
	}
	return b.String()
}

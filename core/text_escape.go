package core

import (
	"strings"
	"unicode/utf8"
)

// Text that is not valid UTF-8 is data too. A file read off disk, a legacy export, a truncated network frame —
// a script has to be able to hold it, look at it, and repair it, and a conversion that raises or substitutes
// U+FFFD makes all three impossible. So decoding is TOTAL: every octet a decoder cannot read as a symbol
// becomes its own reserved rune instead, and encoding turns that rune back into exactly the octet it came from.
//
// The reserved range is U+DC80–U+DCFF, one value per octet 0x80–0xFF. It is exact and it cannot collide:
//
//   - Only an octet ≥ 0x80 can ever be undecodable — an ASCII octet is always a symbol — so 128 values suffice.
//   - Every undecodable octet decodes with width 1, so n bad octets always become n escapes, never fewer.
//   - U+DC80–U+DCFF are low surrogates: never Unicode scalar values, so no well-formed text can produce one,
//     and an escape can never be mistaken for content.
//
// The consequence worth stating plainly: bytes → string → runes → bytes returns the original octets, byte for
// byte, and nothing along the way raises. (The scheme is Python's surrogateescape, PEP 383; Rust's WTF-8
// solves the same problem the same way.)
const (
	escapeLo rune = 0xDC80
	escapeHi rune = 0xDCFF

	surrogateLo rune = 0xD800
	surrogateHi rune = 0xDFFF
)

// IsEscapeRune reports whether r is one of the 128 reserved octet escapes.
// PURE by contract.
func IsEscapeRune(r rune) bool { return r >= escapeLo && r <= escapeHi }

// EscapeRuneOctet returns the octet an escape rune stands for. Meaningless for any other rune.
// PURE by contract.
func EscapeRuneOctet(r rune) byte { return byte(r - escapeLo + 0x80) }

// OctetEscapeRune returns the escape rune reserved for an undecodable octet.
// PURE by contract.
func OctetEscapeRune(b byte) rune { return escapeLo + rune(b) - 0x80 }

// RuneIsValid reports whether r is a real symbol — a Unicode scalar value. An escape is deliberately NOT
// valid: it stands for an octet that is not a symbol, which is exactly what a script needs to test for.
// PURE by contract.
func RuneIsValid(r rune) bool {
	return r >= 0 && r <= utf8.MaxRune && !(r >= surrogateLo && r <= surrogateHi)
}

// RuneInDomain reports whether r may exist as a `rune` value at all: a scalar value, or an escape. Everything
// else — a high surrogate, a negative, anything past U+10FFFF — has no octets to be and is refused where it
// would enter, so that every rune that exists can be encoded and no conversion out of `rune`/`runes` can fail.
// PURE by contract.
func RuneInDomain(r rune) bool { return RuneIsValid(r) || IsEscapeRune(r) }

// IntInRuneDomain is RuneInDomain over an int64, checking the wider range before the narrowing conversion —
// rune(1<<32 + 65) would otherwise truncate to 'A' and pass.
// PURE by contract.
func IntInRuneDomain(i int64) bool {
	return i >= 0 && i <= int64(utf8.MaxRune) && RuneInDomain(rune(i))
}

// DecodeText is `[]rune(s)` with the escape: it never substitutes U+FFFD and never loses an octet.
// PURE by contract.
func DecodeText(s string) []rune {
	if IsASCIIText(s) { // the common case: one octet per symbol, no decode needed
		out := make([]rune, len(s))
		for i := 0; i < len(s); i++ {
			out[i] = rune(s[i])
		}
		return out
	}
	out := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && w <= 1 {
			out = append(out, OctetEscapeRune(s[i]))
			i++
			continue
		}
		out = append(out, r)
		i += w
	}
	return out
}

// DecodeOctets is DecodeText over a byte slice.
// PURE by contract.
func DecodeOctets(b []byte) []rune { return DecodeText(string(b)) }

// EncodeText is `string(rs)` with the escape: an escape rune contributes the single octet it stands for, so a
// value that came from DecodeText round-trips exactly. A rune outside the domain cannot reach here — see
// RuneInDomain — but is written as U+FFFD rather than panicking if one ever does.
// PURE by contract.
func EncodeText(rs []rune) string {
	var b strings.Builder
	b.Grow(len(rs))
	for _, r := range rs {
		if IsEscapeRune(r) {
			b.WriteByte(EscapeRuneOctet(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// EncodeOctets is EncodeText answering the octets directly.
// PURE by contract.
func EncodeOctets(rs []rune) []byte {
	out := make([]byte, 0, len(rs))
	var buf [utf8.UTFMax]byte
	for _, r := range rs {
		if IsEscapeRune(r) {
			out = append(out, EscapeRuneOctet(r))
			continue
		}
		out = append(out, buf[:utf8.EncodeRune(buf[:], r)]...)
	}
	return out
}

// EncodeRuneText is `string(r)` with the escape — one symbol, or the one octet an escape stands for.
// PURE by contract.
func EncodeRuneText(r rune) string {
	if IsEscapeRune(r) {
		return string([]byte{EscapeRuneOctet(r)})
	}
	return string(r)
}

// TextRuneCount counts symbols the way DecodeText produces them: one per symbol, one per undecodable octet.
// utf8.RuneCountInString already counts each bad octet as one, so the two agree by construction.
// PURE by contract.
func TextRuneCount(s string) int { return utf8.RuneCountInString(s) }

// TextIsValid reports whether s decodes with no escapes — i.e. it is well-formed UTF-8.
// PURE by contract.
func TextIsValid(s string) bool { return utf8.ValidString(s) }

// RunesAreValid reports whether every element is a real symbol (no escapes).
// PURE by contract.
func RunesAreValid(rs []rune) bool {
	for _, r := range rs {
		if !RuneIsValid(r) {
			return false
		}
	}
	return true
}

// IsASCIIText reports whether every octet is ASCII. This is the fast-path test for the text types: it implies
// well-formed, one octet per symbol, and byte offsets equal symbol offsets. Note what it is NOT: a rune count
// equal to the octet count, which an undecodable octet also satisfies while being none of those things.
// PURE by contract.
func IsASCIIText(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// RunesAreASCII reports whether every element is an ASCII symbol.
// PURE by contract.
func RunesAreASCII(rs []rune) bool {
	for _, r := range rs {
		if r >= utf8.RuneSelf || r < 0 {
			return false
		}
	}
	return true
}

// OctetsAreASCII reports whether every octet is ASCII.
// PURE by contract.
func OctetsAreASCII(b []byte) bool {
	for _, c := range b {
		if c >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

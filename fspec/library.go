package fspec

import (
	"strings"
	"unicode/utf8"
)

// ApplyGenerics pads and aligns an already-rendered body to satisfy the purely generic Width / Fill / Align fields of
// spec. It is the second stage of the format pipeline: a type's Format method first renders the value (handling Sign,
// Grouping, Precision, CoerceZero, Verb, Tail and any prefix such as "0x") and then calls ApplyGenerics on the result
// to obtain the final string.
//
// defaultAlign supplies the type-default alignment when spec.Align is unset (per spec: AlignLeft for non-numeric types,
// AlignRight for numeric types).
//
// Width is measured in runes; if the body already meets or exceeds spec.Width, the body is returned unchanged. The fill
// character defaults to ' ' (or '0' when the parser set the ZeroPad shortcut, which already lands as Fill='0' +
// Align='=').
//
// For Align == AlignSign the helper inserts fill *between* any leading sign character ('+', '-', ' ') and any
// conventional integer prefix ("0b", "0o", "0x", "0X") and the rest of the body, e.g. "+0000123" or "0x0000002A". On
// bodies that have neither, AlignSign degenerates to right-alignment.
func ApplyGenerics(body string, s FormatSpec, defaultAlign Align) string {
	if !s.HasWidth {
		return body
	}
	bodyWidth := utf8.RuneCountInString(body)
	pad := int(s.Width) - bodyWidth
	if pad <= 0 {
		return body
	}

	fill := s.Fill
	if fill == 0 {
		fill = ' '
	}
	align := s.Align
	if align == AlignNone {
		align = defaultAlign
		if align == AlignNone {
			align = AlignLeft
		}
	}

	switch align {
	case AlignLeft:
		return body + RepeatRune(fill, pad)
	case AlignRight:
		return RepeatRune(fill, pad) + body
	case AlignCenter:
		left := pad / 2
		right := pad - left
		return RepeatRune(fill, left) + body + RepeatRune(fill, right)
	case AlignSign:
		split := SignAwareSplit(body)
		return body[:split] + RepeatRune(fill, pad) + body[split:]
	default:
		return RepeatRune(fill, pad) + body
	}
}

// SignAwareSplit returns the byte index at which fill should be inserted for AlignSign: just after an optional leading
// sign character and an optional conventional integer prefix ("0b", "0o", "0x", "0X").
func SignAwareSplit(body string) int {
	i := 0
	if len(body) > 0 {
		switch body[0] {
		case '+', '-', ' ':
			i = 1
		}
	}
	if len(body) >= i+2 && body[i] == '0' {
		switch body[i+1] {
		case 'b', 'o', 'x', 'X':
			i += 2
		}
	}
	return i
}

// RepeatRune returns a new string consisting of n copies of r.
func RepeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	if r < utf8.RuneSelf {
		return strings.Repeat(string(byte(r)), n)
	}
	return strings.Repeat(string(r), n)
}

// GroupDigits inserts sep every groupSize digits (counted from the right) into the digit string. groupSize must be > 0;
// digits must consist only of digit/letter characters (no leading sign or prefix). Returns digits unchanged when sep ==
// 0 or groupSize <= 0.
func GroupDigits(digits string, sep byte, groupSize int) string {
	if sep == 0 || groupSize <= 0 || len(digits) <= groupSize {
		return digits
	}
	first := len(digits) % groupSize
	if first == 0 {
		first = groupSize
	}
	var b strings.Builder
	b.Grow(len(digits) + (len(digits)-1)/groupSize)
	b.WriteString(digits[:first])
	for i := first; i < len(digits); i += groupSize {
		b.WriteByte(sep)
		b.WriteString(digits[i : i+groupSize])
	}
	return b.String()
}

// SignPrefix returns the sign character that should precede a non-negative numeric body, given the requested Sign mode.
// For negative values, callers should emit the leading '-' from the value itself and pass an empty prefix (this helper
// is a no-op via the SignMinus / SignDefault branches when negative=true).
func SignPrefix(sign Sign, negative bool) string {
	if negative {
		return ""
	}
	switch sign {
	case SignPlus:
		return "+"
	case SignSpace:
		return " "
	}
	return ""
}

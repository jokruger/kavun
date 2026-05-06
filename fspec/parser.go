package fspec

import (
	"fmt"
	"math"
	"strings"
)

// Parse parses the format mini-language expression (see docs/format-mini-language.md).
func Parse(text string) (FormatSpec, error) {
	var spec FormatSpec

	// fast-path
	switch text {
	case "":
		return spec, nil
	case "d":
		spec.Verb = 'd'
		return spec, nil
	case "s":
		spec.Verb = 's'
		return spec, nil
	case "v":
		spec.Verb = 'v'
		return spec, nil
	}

	// split off the tail at the first '#'
	generic, tail, hasTail := strings.Cut(text, "#")
	if hasTail {
		spec.Verb = '#'
		spec.Tail = tail
	}
	if generic == "" {
		return spec, nil
	}

	runes := []rune(generic)
	p := 0
	n := len(runes)

	// [fill] align
	// fill+align requires the second rune to be an align character and the first rune to be anything except '{' or '}'
	// otherwise, a single align rune at position 0 is taken as the alignment with default fill.
	if n >= 2 && isAlign(runes[1]) && runes[0] != '{' && runes[0] != '}' {
		spec.Fill = runes[0]
		spec.Align = Align(runes[1])
		p = 2
	} else if n >= 1 && isAlign(runes[0]) {
		spec.Align = Align(runes[0])
		p = 1
	}

	// sign
	if p < n && (runes[p] == '+' || runes[p] == '-' || runes[p] == ' ') {
		spec.Sign = Sign(runes[p])
		p++
	}

	// width (decimal digits)
	// a leading '0' without explicit align enables the sign-aware zero-pad shortcut.
	if p < n && isDigit(runes[p]) {
		if runes[p] == '0' && spec.Align == AlignNone {
			spec.ZeroPad = true
			spec.Fill = '0'
			spec.Align = AlignSign
		}
		w := 0
		for p < n && isDigit(runes[p]) {
			w = w*10 + int(runes[p]-'0')
			if w > math.MaxInt16 {
				return spec, fmt.Errorf("fspec: width out of range in %q", text)
			}
			p++
		}
		spec.Width = int16(w)
		spec.HasWidth = true
	}

	// grouping
	if p < n && (runes[p] == ',' || runes[p] == '_') {
		spec.Grouping = byte(runes[p])
		p++
	}

	// '.' precision
	if p < n && runes[p] == '.' {
		p++
		if p >= n || !isDigit(runes[p]) {
			return spec, fmt.Errorf("fspec: precision requires digits in %q", text)
		}
		pr := 0
		for p < n && isDigit(runes[p]) {
			pr = pr*10 + int(runes[p]-'0')
			if pr > math.MaxInt16 {
				return spec, fmt.Errorf("fspec: precision out of range in %q", text)
			}
			p++
		}
		spec.Precision = int16(pr)
		spec.HasPrec = true
	}

	// flag* — order-independent set of single-character symbol flags, each at most once.
	// '~' = CoerceZero (negative-zero coercion for float / decimal)
	// '!' = Bare (suppress conventional integer prefix "0b" / "0o" / "0x" / "0X")
	// Symbol flags keep the verb namespace open for letters; new flags should also be non-letter symbols.
	for p < n {
		switch runes[p] {
		case '~':
			if spec.CoerceZero {
				return spec, fmt.Errorf("fspec: duplicate flag '~' in %q", text)
			}
			spec.CoerceZero = true
			p++
			continue
		case '!':
			if spec.Bare {
				return spec, fmt.Errorf("fspec: duplicate flag '!' in %q", text)
			}
			spec.Bare = true
			p++
			continue
		}
		break
	}

	// verb (single ASCII letter or '%' at the end). When `#tail` is also present, the letter overrides the tentative
	// Verb='#' set earlier so the type's Format method receives both the explicit verb and the opaque tail. Types
	// that do not accept a tail along with a generic verb must reject the combination.
	if p < n {
		r := runes[p]
		if !isVerbChar(r) {
			return spec, fmt.Errorf("fspec: unexpected %q in %q", string(r), text)
		}
		if p != n-1 {
			return spec, fmt.Errorf("fspec: trailing characters %q in %q",
				string(runes[p+1:]), text)
		}
		spec.Verb = byte(r)
		p++
	}

	if p != n {
		return spec, fmt.Errorf("fspec: trailing characters in %q", text)
	}

	return spec, nil
}

func isAlign(r rune) bool {
	return r == '<' || r == '>' || r == '^' || r == '='
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isVerbChar reports whether r may serve as a single-character verb. Verbs are normally ASCII letters, but '%' is a
// special-case verb for float / decimal (multiply by 100, append '%').
func isVerbChar(r rune) bool {
	return isASCIILetter(r) || r == '%'
}

// HasUnconsumedTail reports whether the spec carries a tail that was not consumed via Verb='#'. Type Format methods
// that don't accept a tail alongside a generic verb should reject when this is true. Verbs 'v' and 'T' are
// universal "ignore everything else" verbs and therefore exempted.
func (s FormatSpec) HasUnconsumedTail() bool {
	return s.Tail != "" && s.Verb != '#' && s.Verb != 'v' && s.Verb != 'T'
}

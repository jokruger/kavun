package text

import (
	"fmt"

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

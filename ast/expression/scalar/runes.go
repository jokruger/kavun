package scalar

import "github.com/jokruger/kavun/core"

// Runes represents a unicode string literal (u"...").
type Runes struct {
	Value    []rune
	ValuePos core.Pos
	Literal  string
}

func (e *Runes) Pos() core.Pos {
	return e.ValuePos
}

func (e *Runes) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'u' prefix
}

func (e *Runes) String() string {
	return "u" + e.Literal
}

func (e *Runes) IsUndefinedLiteral() bool {
	return false
}

func (e *Runes) IsScalarLiteral() bool {
	return true
}

func (e *Runes) IsCompositeLiteral() bool {
	return false
}

func (e *Runes) IsCallExpression() bool {
	return false
}

func (e *Runes) LiteralToValue() (core.Value, bool) {
	return core.NewRunesValue(e.Value, true), true
}

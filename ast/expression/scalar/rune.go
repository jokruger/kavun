package scalar

import "github.com/jokruger/kavun/core"

// Rune represents a character literal.
type Rune struct {
	Value    rune
	ValuePos core.Pos
	Literal  string
}

func (e *Rune) Pos() core.Pos {
	return e.ValuePos
}

func (e *Rune) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Rune) String() string {
	return e.Literal
}

func (e *Rune) IsUndefinedLiteral() bool {
	return false
}

func (e *Rune) IsScalarLiteral() bool {
	return true
}

func (e *Rune) IsCompositeLiteral() bool {
	return false
}

func (e *Rune) IsCallExpression() bool {
	return false
}

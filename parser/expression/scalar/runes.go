package scalar

import "github.com/jokruger/kavun/core"

// Runes represents a unicode string literal (u"...").
type Runes struct {
	Value    []rune
	ValuePos core.Pos
	Literal  string
}

func (e *Runes) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Runes) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Runes) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'u' prefix
}

func (e *Runes) String() string {
	return "u" + e.Literal
}

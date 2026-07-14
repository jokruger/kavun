package scalar

import "github.com/jokruger/kavun/core"

// Rune represents a character literal.
type Rune struct {
	Value    rune
	ValuePos core.Pos
	Literal  string
}

func (e *Rune) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Rune) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Rune) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Rune) String() string {
	return e.Literal
}

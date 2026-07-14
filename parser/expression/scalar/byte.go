package scalar

import "github.com/jokruger/kavun/core"

// Byte represents a byte literal.
type Byte struct {
	Value    byte
	ValuePos core.Pos
	Literal  string
}

func (e *Byte) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Byte) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Byte) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'b' prefix
}

func (e *Byte) String() string {
	return "b" + e.Literal
}

package scalar

import "github.com/jokruger/kavun/core"

// Float represents a floating point literal.
type Float struct {
	Value    float64
	ValuePos core.Pos
	Literal  string
}

func (e *Float) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Float) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Float) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Float) String() string {
	return e.Literal
}

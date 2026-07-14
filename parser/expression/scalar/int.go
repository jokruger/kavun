package scalar

import "github.com/jokruger/kavun/core"

// Int represents an integer literal.
type Int struct {
	Value    int64
	ValuePos core.Pos
	Literal  string
}

func (e *Int) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Int) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Int) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Int) String() string {
	return e.Literal
}

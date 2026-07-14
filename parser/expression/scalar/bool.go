package scalar

import "github.com/jokruger/kavun/core"

// Bool represents a boolean literal.
type Bool struct {
	Value    bool
	ValuePos core.Pos
	Literal  string
}

func (e *Bool) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Bool) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Bool) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Bool) String() string {
	return e.Literal
}

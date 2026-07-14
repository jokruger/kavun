package scalar

import "github.com/jokruger/kavun/core"

// String represents a string literal.
type String struct {
	Value    string
	ValuePos core.Pos
	Literal  string
}

func (e *String) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *String) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *String) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *String) String() string {
	return e.Literal
}

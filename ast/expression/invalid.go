package expression

import "github.com/jokruger/kavun/core"

// Invalid represents invalid expression.
type Invalid struct {
	From core.Pos
	To   core.Pos
}

func (e *Invalid) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Invalid) Pos() core.Pos {
	return e.From
}

// End returns the position of first character immediately after the node.
func (e *Invalid) End() core.Pos {
	return e.To
}

func (e *Invalid) String() string {
	return "<bad expression>"
}
